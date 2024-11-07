package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Etcd3 struct {
	Hosts         string `json:"hosts" yaml:"hosts"`
	AppConfigKey  string `json:"appConfigKey" yaml:"appConfigKey"`
	GrpcConfigKey string `json:"grpcConfigKey" yaml:"grpcConfigKey"`
}

func PathToEtcd(path string) (*Etcd3, error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error open JSON file:", err)
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var remoteConfig Etcd3
	err = decoder.Decode(&remoteConfig)
	if err != nil {
		fmt.Println("Error decoding JSON file:", err)
		return nil, err
	}
	return &remoteConfig, nil
}
