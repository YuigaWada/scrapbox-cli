package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const configPath = "config.json"

type Config struct {
	Project string
}

func (conf *Config) save() error {
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}

	defer file.Close()
	err = json.NewEncoder(file).Encode(conf)
	return err
}

func LoadConfig() (Config, error) {
	var config Config
	file, err := os.Open(configPath)
	defer file.Close()
	if err != nil {
		fmt.Println("error: Register your project name first!")
		return Config{}, err
	}

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		fmt.Println("error: Your config file is not valid!")
		return Config{}, err
	}
	// return config
	return config, nil
}
