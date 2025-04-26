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
		fmt.Printf("Got signal: %s\n", signal.Name)
		for i, body := range signal.Body {
			fmt.Printf("  Arg %d: %#v\n", i, body)
		}
	}
}
