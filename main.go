package main

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

func main() {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	obj := conn.Object(
		"net.connman.iwd",
		"/",
	)
	//Get the managed objects
	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant

	err = obj.Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects",
		0,
	).Store(&managed)

	if err != nil {
		panic(err)
	}
	//Get the wlans
	var stationPaths []dbus.ObjectPath
	for path, ifaces := range managed {
		if _, ok := ifaces["net.connman.iwd.Station"]; ok {
			stationPaths = append(stationPaths, path)
		}
	}

	for _, p := range stationPaths {
		fmt.Println("Station:", p)
	}

	//Discovers the networks
	for {
		stationObj := conn.Object(
			"net.connman.iwd",
			stationPaths[0],
		)

		err := stationObj.Call(
			"net.connman.iwd.Station.Scan",
			0,
		)
		if err != nil {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	//Lists the networks
	for path, ifaces := range managed {
		network, ok := ifaces["net.connman.iwd.Network"]
		if !ok {
			continue
		}

		fmt.Println("Network:", path)

		for name, value := range network {
			fmt.Printf("  %s = %#v\n",
				name,
				value.Value(),
			)
		}
	}
}
