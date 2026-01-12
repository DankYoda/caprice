package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	//gets all interfaces
	interfaces, err := net.Interfaces()
	wifiInterface := make([]net.Interface, 0)
	if err != nil {
		panic(err)
	}
	// returns all wifi interfaces
	for _, inter := range interfaces {
		if strings.Contains(inter.Name, "wlan") || strings.Contains(inter.Name, "wifi") || strings.Contains(inter.Name, "wlp") {
			wifiInterface = append(wifiInterface, inter)
		}
	}
	fmt.Printf("Hello and welcome, %v!\n", wifiInterface)
}
