package discover

import (
	"context"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
	"xiongba/grpc/config"
)

var (
	loadBalanceItem DiscoverLoadBalanceItem
	once            sync.Once
	RemoteCenter    *config.RemoteCenter
	Discovery       *EtcdDiscovery
)

// EtcdDiscovery 服务发现
type EtcdDiscovery struct {
	Cli        *clientv3.Client  // etcd连接
	ServiceMap map[string]string // 服务列表(k-v列表)
	Lock       sync.RWMutex      // 读写互斥锁
}

func NewServiceDiscovery(endpoints ...string) (*EtcdDiscovery, error) {
	// 创建etcdClient对象
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &EtcdDiscovery{
		Cli:        cli,
		ServiceMap: make(map[string]string), // 初始化kvMap
	}, nil
}

// ServiceDiscovery 读取etcd的服务并开启协程监听kv变化
func (e *EtcdDiscovery) ServiceDiscovery(prefix ...string) error {
	// 根据服务名称的前缀，获取所有的注册服务
	for _, prefix := range prefix {
		resp, err := e.Cli.Get(context.Background(), prefix, clientv3.WithPrefix())
		if err != nil {
			return err
		}
		// 遍历key-value存储到本地map
		for _, kv := range resp.Kvs {
			e.PutService(string(kv.Key), string(kv.Value))
		}
	}
	return nil
}

// SetService 新增或修改本地服务
func (s *EtcdDiscovery) PutService(key, val string) {
	s.Lock.Lock()
	s.ServiceMap[key] = val
	s.Lock.Unlock()
	log.Println("put key :", key, "val:", val)
}

// DelService 删除本地服务
func (s *EtcdDiscovery) DelService(key string) {
	s.Lock.Lock()
	delete(s.ServiceMap, key)
	s.Lock.Unlock()
	log.Println("del key:", key)
}

// GetService 获取本地服务
func (s *EtcdDiscovery) GetService(serviceName string) (string, error) {
	switch RemoteCenter.Strategy {
	case RandomStrategy:
		return s.random(serviceName)
	case RoundRobinStrategy:
		return s.roundRobin(serviceName)
	default:
		return s.random(serviceName)
	}
}

// Close 关闭服务
func (e *EtcdDiscovery) Close() error {
	return e.Cli.Close()
}
func (s *EtcdDiscovery) random(serviceName string) (string, error) {
	s.Lock.RLock()
	resetRandomPos(serviceName)
	serviceAddr := loadBalanceItem.LoadBalanceMap[serviceName][loadBalanceItem.RandomPos]
	s.Lock.RUnlock()
	return serviceAddr, nil
}

func (s *EtcdDiscovery) roundRobin(serviceName string) (string, error) {

	s.Lock.RLock()
	resetPos(serviceName)
	serviceAddr := loadBalanceItem.LoadBalanceMap[serviceName][loadBalanceItem.PosMap[serviceName]]
	s.Lock.RUnlock()
	return serviceAddr, nil

}

// 定义负载均衡策略
const (
	RandomStrategy     = "random"
	RoundRobinStrategy = "roundRobin"
	//Weighted           = "weighted"
)

type DiscoverLoadBalanceItem struct {
	LoadBalanceMap map[string][]string
	//Indexes   []string
	Pos          int
	PosMap       map[string]int
	RandomPos    int
	RandomPosMap map[string]int
	Lock         sync.Mutex
}

func (s *EtcdDiscovery) InitLoanBalanceItem(serviceName ...string) {
	loadBalanceItem = DiscoverLoadBalanceItem{
		LoadBalanceMap: make(map[string][]string),
		PosMap:         make(map[string]int),
		RandomPosMap:   make(map[string]int),
		Lock:           sync.Mutex{},
	}
	for _, serviceName := range serviceName {
		indexes := []string{}
		for k, v := range s.ServiceMap {
			if strings.HasPrefix(k, serviceName) {
				indexes = append(indexes, v)
			}
		}
		loadBalanceItem.LoadBalanceMap[serviceName] = indexes
		loadBalanceItem.PosMap[serviceName] = 0
		loadBalanceItem.RandomPosMap[serviceName] = 0
	}
}
func resetPos(serviceName string) {
	loadBalanceItem.Lock.Lock()
	if loadBalanceItem.PosMap[serviceName] >= len(loadBalanceItem.LoadBalanceMap[serviceName])-1 {
		loadBalanceItem.PosMap[serviceName] = 0
	} else {
		loadBalanceItem.PosMap[serviceName] = loadBalanceItem.PosMap[serviceName] + 1
	}
	loadBalanceItem.Lock.Unlock()
}
func resetRandomPos(serviceName string) {
	loadBalanceItem.Lock.Lock()
	loadBalanceItem.RandomPosMap[serviceName] = rand.Intn(len(loadBalanceItem.LoadBalanceMap[serviceName]))
	loadBalanceItem.Lock.Unlock()
}

func InitDiscovery(path string) error {
	var err error
	RemoteCenter, err = config.JsonPathToGrpcServiceConfig(path)
	if err != nil {
		return err
	}
	Discovery, err = NewServiceDiscovery(RemoteCenter.Etcd3.Hosts)
	if err != nil {
		return err
	}
	err = Discovery.ServiceDiscovery(RemoteCenter.ServiceName...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return err
	}
	go func() {
		once.Do(func() {
			Discovery.InitLoanBalanceItem(RemoteCenter.ServiceName...)
		})
	}()
	Discovery.watchService(RemoteCenter.ServiceName...)
	return nil
}
func (s *EtcdDiscovery) watchService(prefix ...string) {
	for _, prefix := range prefix {
		go func() {
			watchRespChan := Discovery.Cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
			log.Printf("watching prefix:%s now...", prefix)
			for watchResp := range watchRespChan {
				for _, event := range watchResp.Events {
					switch event.Type {
					case mvccpb.PUT: // 发生了修改或者新增

						Discovery.PutService(string(event.Kv.Key), string(event.Kv.Value))
						Discovery.InitLoanBalanceItem(RemoteCenter.ServiceName...)
						//}
					case mvccpb.DELETE: //发生了删除
						Discovery.DelService(string(event.Kv.Key)) // ServiceMap中进行相应的删除
						Discovery.InitLoanBalanceItem(RemoteCenter.ServiceName...)
						//}
					}
				}
			}

		}()
	}
}
