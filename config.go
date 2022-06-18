package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	project string
}

func (conf *Config) save(path string) {
	file, err := os.Create("config.json")
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	err = json.NewEncoder(file).Encode(conf)
	if err != nil {
		log.Fatal(err)
	}
}

func LoadConfig() Config {
	var config Config
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
		fmt.Println("error: Register your project name first!")
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatal(err)
		fmt.Println("error: Your config file is not valid!")
	}
	// return config
	return Config{"yuwd"}
}
