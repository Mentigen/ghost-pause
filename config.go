package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	TargetApps    []string `json:"target_apps"`
	IgnorePlayers []string `json:"ignore_players"`
	PauseDelayMs  int      `json:"pause_delay_ms"`
}

var defaultConfig = Config{
	TargetApps: []string{"Zen", "Firefox", "Google Chrome", "Chromium", "Brave"},
}

func loadConfig(path string) Config {
	if path == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			log.Println(err)
			return defaultConfig
		}
		path = filepath.Join(configDir, "ghost-pause/config.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Println("config not found, using defaults:", err)
		return defaultConfig
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Println("config parse error, using defaults:", err)
		return defaultConfig
	}

	if len(cfg.TargetApps) == 0 {
		cfg.TargetApps = defaultConfig.TargetApps
	}
	return cfg
}
