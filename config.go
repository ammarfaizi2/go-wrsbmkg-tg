package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

var config struct {
	Token         string   `yaml:"BOT_TOKEN"`
	ChatID        int      `yaml:"CHAT_ID"`
	MinMag        float64  `yaml:"MIN_MAG"`
	FilterRegions []string `yaml:"FILTER_REGIONS"`
	MsgMemoryDir  string   `yaml:"MSG_MEMORY_DIR"`
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

	if len(config.FilterRegions) > 0 {
		for i, w := range config.FilterRegions {
			config.FilterRegions[i] = strings.ToLower(w)
		}
	}
}
