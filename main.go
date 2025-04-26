package main

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
)

func main() {
	// Connect to the system bus
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
	}

	// Add a rule to only capture org.bluez signals
	call := conn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.bluez'",
	)
	if call.Err != nil {
		log.Fatalf("Failed to add match rule: %v", call.Err)
	}

	fmt.Println("Listening for org.bluez signals...")

	// Receive and handle signals
	ch := make(chan *dbus.Signal, 10)
	conn.Signal(ch)
	for signal := range ch {
		if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
			iface := signal.Body[0].(string)
			changedProps := signal.Body[1].(map[string]dbus.Variant)

			if iface == "org.bluez.MediaTransport1" {
				if volVariant, ok := changedProps["Volume"]; ok {
					volume := volVariant.Value().(uint16)
					fmt.Printf("Current Volume: %d\n", volume)
				}
			}
		}
	}
}
