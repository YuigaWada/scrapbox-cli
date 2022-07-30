package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kirsle/configdir"
)

var dirPath = filepath.Join(configdir.LocalConfig(), "sbox")
var configPath = filepath.Join(dirPath, "config.json")

type Config struct {
	Project string
}

func (conf *Config) save() error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.Mkdir(dirPath, 0755)
		if err != nil {
			return nil
		}
	}

	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
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
