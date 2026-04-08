package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Bind     string   `json:"Bind"`
	Peer     []string `json:"Peer"`
	Password string   `json:"Password"`
}

func Load(name string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
