package main

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
)

var config struct {
	Token        string  `yaml:"BOT_TOKEN"`
	ChatID       int     `yaml:"CHAT_ID"`
	MinMag       float64 `yaml:"MIN_MAG"`
	FilterRegion string  `yaml:"FILTER_REGION"`
}

func ReadConfig(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			panic("Config file does not exist.")
		}

		panic(fmt.Sprintf("error when reading %s: %s", filename, err))
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		panic(fmt.Sprintf("error when parsing %s: %s", filename, err))
	}
}
