package main

import (
	"context"
	"fmt"
	"log"
	"time"

	//"net"

	"github.com/mdlayher/wifi"
)

func main() {
	//gets all interfaces
	//Creates a new client for scanning
	client, err := wifi.New()
	if err != nil {
		return
	}
	defer client.Close()
	// Get all available Wi-Fi interfaces
	interfaces, err := client.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	// Use the first available interface for scanning
	if len(interfaces) == 0 {
		log.Fatal("No Wi-Fi interfaces found")
	}
	ifi := interfaces[1]
	fmt.Printf("Using interface: %s (%s)\n", ifi.Name, ifi.HardwareAddr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Request a scan on the selected interface
	err = client.Scan(ctx, ifi)
	if err != nil {
		log.Fatal(err)
	}

	// Might need some async magic to let the client.Scan call finish
	aps, err := client.AccessPoints(ifi)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Available Access Points:")
	fmt.Printf("%-32s %-18s %-5s\n", "SSID", "BSSID", "Signal")
	for _, ap := range aps {
		//fmt.Printf("%v\n", *ap)
		fmt.Printf("%-32s %-18s %-5d dBm\n", ap.SSID, ap.BSSID.String(), ap.Signal)
	}
}
