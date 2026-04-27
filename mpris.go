package main

import (
	"log"
	"strings"

	"github.com/godbus/dbus/v5"
)

func pauseOthers(ignoreList []string, conn *dbus.Conn) []string {
	var interruptedPlayers []string

	busObj := conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")

	var names []string

	err := busObj.Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		log.Println(err)
	}

	method := "org.mpris.MediaPlayer2.Player.Pause"

	for _, name := range names {
		if strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
			lowerBusName := strings.ToLower(name)

			var shouldIgnore = false
			for _, target := range ignoreList {
				if strings.Contains(lowerBusName, strings.ToLower(target)) {
					shouldIgnore = true
					break
				}
			}
			if shouldIgnore {
				continue
			}

			obj := conn.Object(name, "/org/mpris/MediaPlayer2")
			variant, err := obj.GetProperty("org.mpris.MediaPlayer2.Player.PlaybackStatus")
			if err == nil {
				status, ok := variant.Value().(string)
				if !ok {
					continue
				}
				if status == "Playing" {
					call := obj.Call(method, 0)
					if call.Err != nil {
						continue
					} else {
						interruptedPlayers = append(interruptedPlayers, name)
					}
				}
			}
		}
	}
	return interruptedPlayers
}

func resumeOthers(playersToResume []string, conn *dbus.Conn) {

	for _, name := range playersToResume {
		obj := conn.Object(name, "/org/mpris/MediaPlayer2")

		call := obj.Call("org.mpris.MediaPlayer2.Player.Play", 0)
		if call.Err != nil {
			log.Println(call.Err)
		}
	}
}
