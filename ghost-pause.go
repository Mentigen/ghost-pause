package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
)

var version = "dev"

type SinkInput struct {
	Properties struct {
		AppName string `json:"application.name"`
	} `json:"properties"`
	Corked bool `json:"corked"`
}

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	configFlag := flag.String("config", "", "path to config file (default: ~/.config/ghost-pause/config.json)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ghost-pause [flags] [command]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  start   re-enable auto-pausing in the running daemon\n")
		fmt.Fprintf(os.Stderr, "  stop    disable auto-pausing in the running daemon\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "stop":
			sendSignal(syscall.SIGUSR1)
			fmt.Println("auto-pausing disabled")
		case "start":
			sendSignal(syscall.SIGUSR2)
			fmt.Println("auto-pausing enabled")
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
			os.Exit(1)
		}
		return
	}

	runDaemon(*configFlag)
}

func runDaemon(configPath string) {
	cfg := loadConfig(configPath)

	if err := writePIDFile(); err != nil {
		log.Println("warning: could not write PID file:", err)
	}
	defer removePIDFile()

	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	pactlCmd := exec.Command("pactl", "subscribe")
	stdout, err := pactlCmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := pactlCmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer pactlCmd.Wait()

	lines := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2)

	pauseFired := make(chan struct{}, 1)
	var pauseTimer *time.Timer

	enabled := true
	wasBrowserPlaying := false
	var pausedApps []string

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return
			}
			if !enabled || !strings.Contains(line, "sink-input") {
				continue
			}

			out, err := exec.Command("pactl", "-f", "json", "list", "sink-inputs").Output()
			if err != nil {
				log.Println(err)
				continue
			}

			var inputs []SinkInput
			if err := json.Unmarshal(out, &inputs); err != nil {
				log.Println(err)
				continue
			}

			isBrowserPlaying := false
			for _, input := range inputs {
				if slices.Contains(cfg.TargetApps, input.Properties.AppName) && !input.Corked {
					isBrowserPlaying = true
					break
				}
			}

			if isBrowserPlaying && !wasBrowserPlaying {
				wasBrowserPlaying = true
				if cfg.PauseDelayMs > 0 {
					pauseTimer = time.AfterFunc(time.Duration(cfg.PauseDelayMs)*time.Millisecond, func() {
						pauseFired <- struct{}{}
					})
				} else {
					pausedApps = pauseOthers(cfg.TargetApps, cfg.IgnorePlayers, conn)
				}
			} else if !isBrowserPlaying && wasBrowserPlaying {
				wasBrowserPlaying = false
				if pauseTimer != nil {
					pauseTimer.Stop()
					pauseTimer = nil
				}
				resumeOthers(pausedApps, conn)
				pausedApps = nil
			}

		case <-pauseFired:
			if !wasBrowserPlaying || !enabled {
				continue
			}
			pauseTimer = nil
			pausedApps = pauseOthers(cfg.TargetApps, cfg.IgnorePlayers, conn)

		case sig := <-sigCh:
			switch sig {
			case syscall.SIGUSR1:
				enabled = false
				if pauseTimer != nil {
					pauseTimer.Stop()
					pauseTimer = nil
				}
				resumeOthers(pausedApps, conn)
				pausedApps = nil
				wasBrowserPlaying = false
				log.Println("auto-pausing disabled")
			case syscall.SIGUSR2:
				enabled = true
				log.Println("auto-pausing enabled")
			}
		}
	}
}

func pidFilePath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "user-"+strconv.Itoa(os.Getuid()))
	}
	return filepath.Join(dir, "ghost-pause.pid")
}

func writePIDFile() error {
	return os.WriteFile(pidFilePath(), []byte(strconv.Itoa(os.Getpid())), 0600)
}

func removePIDFile() {
	os.Remove(pidFilePath())
}

func sendSignal(sig syscall.Signal) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		log.Fatal("ghost-pause is not running (PID file not found)")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		log.Fatal("invalid PID file")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Fatal(err)
	}
	if err := proc.Signal(sig); err != nil {
		log.Fatalf("failed to send signal: %v", err)
	}
}
