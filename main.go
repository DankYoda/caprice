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

	var managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

	err = obj.Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects",
		0,
	).Store(&managedObjects)

	if err != nil {
		panic(err)
	}

	for path, interfaces := range managedObjects {
		fmt.Println(path)
		for iface := range interfaces {
			fmt.Println("  ", iface)
		}
	}
}
