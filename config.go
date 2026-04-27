package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	TargetApps []string `json:"target_apps"`
}

var fallback = []string{"Zen", "Firefox", "Google Chrome", "Chromium", "Brave"}

func loadConfig() []string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Println(err)
		return fallback
	}

	path := filepath.Join(configDir, "ghost-pause/config.json")

	file, err := os.ReadFile(path)
	if err != nil {
		log.Println("Error reading config file:", err)
		return fallback
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Println(err)
		return fallback
	}

	if len(config.TargetApps) == 0 {
		return fallback
	}
	return config.TargetApps
}
