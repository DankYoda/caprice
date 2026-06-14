package main

import (
	"fmt"

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

	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant

	err = obj.Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects",
		0,
	).Store(&managed)

	if err != nil {
		panic(err)
	}
	var allWifiAdapters []map[string]dbus.Variant
	for _, ifaces := range managed {
		device, hasDevice := ifaces["net.connman.iwd.Device"]
		_, hasStation := ifaces["net.connman.iwd.Station"]
		if hasDevice && hasStation {
			allWifiAdapters = append(allWifiAdapters, device)
			fmt.Println("Wi-Fi interface:", device)
		}
	}
}
