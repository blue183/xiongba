package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type RemoteCenter struct {
	ServiceName    []string `yaml:"serviceName" json:"serviceName" mapstructure:"serviceName"`
	Strategy       string   `yaml:"strategy" json:"strategy" mapstructure:"strategy"`
	ServiceID      string   `yaml:"serviceID" json:"serviceID" mapstructure:"serviceID"`
	ServiceAddress string   `yaml:"serviceAddress" json:"serviceAddress" mapstructure:"serviceAddress"`
	Expire         int64    `yaml:"expire" json:"expire" mapstructure:"expire"`
	Network        string   `yaml:"network" json:"network" mapstructure:"network"`
	Etcd3          Etcd3    `yaml:"etcd3" json:"etcd3" mapstructure:"etcd3"`
}

func JsonPathToGrpcServiceConfig(path string) (*RemoteCenter, error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error open JSON file:", err)
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var remoteCenter RemoteCenter
	err = decoder.Decode(&remoteCenter)
	if err != nil {
		fmt.Println("Error decoding JSON file:", err)
		return nil, err
	}
	return &remoteCenter, nil
}
