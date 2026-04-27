package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os/exec"
	"slices"
	"strings"

	"github.com/godbus/dbus/v5"
)

type SinkInput struct {
	Properties struct {
		AppName string `json:"application.name"`
	} `json:"properties"`
	Corked bool `json:"corked"`
}

func main() {

	cmd := exec.Command("pactl", "subscribe")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)

	wasBrowserPlaying := false

	targetApps := loadConfig()
	var pausedApps []string

	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "sink-input") {

			cmd := exec.Command("pactl", "-f", "json", "list", "sink-inputs")
			output, err := cmd.Output()
			if err != nil {
				log.Println(err)
				continue
			}

			var inputs []SinkInput

			if err := json.Unmarshal(output, &inputs); err != nil {
				log.Println(err)
				continue
			}

			var isBrowserPlaying = false
			for _, input := range inputs {
				if slices.Contains(targetApps, input.Properties.AppName) && !input.Corked {
					isBrowserPlaying = true
					break
				}
			}

			if isBrowserPlaying && !wasBrowserPlaying {
				wasBrowserPlaying = true
				pausedApps = pauseOthers(targetApps, conn)
			} else if !isBrowserPlaying && wasBrowserPlaying {
				wasBrowserPlaying = false
				resumeOthers(pausedApps, conn)
				pausedApps = nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}
