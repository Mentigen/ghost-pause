package main

import (
	"log"
	"strings"

	"github.com/godbus/dbus/v5"
)

func pauseOthers(ignoreList []string, ignorePlayers []string, conn *dbus.Conn) []string {
	var interruptedPlayers []string

	busObj := conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")

	var names []string
	if err := busObj.Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		log.Println(err)
	}

	method := "org.mpris.MediaPlayer2.Player.Pause"

	for _, name := range names {
		if !strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
			continue
		}

		lowerBusName := strings.ToLower(name)

		ignore := false
		for _, target := range ignoreList {
			if strings.Contains(lowerBusName, strings.ToLower(target)) {
				ignore = true
				break
			}
		}
		if ignore {
			continue
		}

		for _, player := range ignorePlayers {
			if strings.Contains(lowerBusName, strings.ToLower(player)) {
				ignore = true
				break
			}
		}
		if ignore {
			continue
		}

		obj := conn.Object(name, "/org/mpris/MediaPlayer2")
		variant, err := obj.GetProperty("org.mpris.MediaPlayer2.Player.PlaybackStatus")
		if err != nil {
			continue
		}
		status, ok := variant.Value().(string)
		if !ok || status != "Playing" {
			continue
		}
		if call := obj.Call(method, 0); call.Err == nil {
			interruptedPlayers = append(interruptedPlayers, name)
		}
	}
	return interruptedPlayers
}

func resumeOthers(playersToResume []string, conn *dbus.Conn) {
	for _, name := range playersToResume {
		obj := conn.Object(name, "/org/mpris/MediaPlayer2")
		if call := obj.Call("org.mpris.MediaPlayer2.Player.Play", 0); call.Err != nil {
			log.Println(call.Err)
		}
	}
}
