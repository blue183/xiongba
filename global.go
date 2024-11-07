package grpc

import (
	"xiongba/grpc/config"
	"xiongba/grpc/discover"
)

var (
	Discovery      *discover.EtcdDiscovery
	DiscoverConfig *config.RemoteCenter
)
