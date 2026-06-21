package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/godbus/dbus/v5"
)

func main() {
	//discoverNetworks(conn, wifiAdapters[0].path)

	//bubble tea here
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type model struct {
	networks     []wifiNetwork
	wifiAdapters []wifiAdapter
	cursor       int
	selected     map[int]struct{}
}

func initialModel() model {
	return model{
		networks:     getNetworks(),
		wifiAdapters: getWifiAdapters(),

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyPressMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.networks)-1 {
				m.cursor++
			}

		// The "enter" key and the space bar toggle the selected state
		// for the item that the cursor is pointing at.
		case "enter", "space":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() tea.View {
	// The header
	s := "\nBehold, the wifi!\n\n"

	// Iterate over our choices
	for i, choice := range m.networks {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += "\nBehold, the Adapters!\n\n"

	for _, adapter := range m.wifiAdapters {
		s += fmt.Sprintf("%s\n", adapter)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return tea.NewView(s)
}

func getWifiAdapters() []wifiAdapter {
	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	obj := conn.Object("net.connman.iwd", "/")
	//Get the managed objects
	var managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

	err = obj.Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects",
		0,
	).Store(&managedObjects)

	if err != nil {
		panic(err)
	}
	//Get the wlans
	var wifiAdapters []wifiAdapter
	for path, ifaces := range managedObjects {
		if _, ok := ifaces["net.connman.iwd.Station"]; ok {
			wifiAdapters = append(wifiAdapters, wifiAdapter{
				name:     ifaces["net.connman.iwd.Device"]["Name"].Value().(string),
				address:  ifaces["net.connman.iwd.Device"]["Address"].Value().(string),
				powered:  ifaces["net.connman.iwd.Device"]["Powered"].Value().(bool),
				adapter:  ifaces["net.connman.iwd.Device"]["Adapter"].String(),
				scanning: ifaces["net.connman.iwd.Station"]["Scanning"].Value().(bool),
				state:    ifaces["net.connman.iwd.Station"]["State"].Value().(string),
				path:     path,
			})
		}
	}
	return wifiAdapters
}

type wifiAdapter struct {
	name     string
	address  string
	powered  bool
	adapter  string
	scanning bool
	state    string
	path     dbus.ObjectPath
}

func discoverNetworks(conn *dbus.Conn, stationPath dbus.ObjectPath) {
	//Discovers the networks
	for {
		stationObj := conn.Object(
			"net.connman.iwd",
			stationPath,
		)
		err := stationObj.Call(
			"net.connman.iwd.Station.Scan",
			0,
		)
		var results []struct {
			Path     dbus.ObjectPath
			Strength int16
		}

		stationObj.Call(
			"net.connman.iwd.Station.GetOrderedNetworks",
			0,
		).Store(&results)

		if err != nil {
			break
		}
	}
}

func getNetworks() []wifiNetwork {
	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	obj := conn.Object("net.connman.iwd", "/")

	//Get the managed objects
	var managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

	err = obj.Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects",
		0,
	).Store(&managedObjects)

	if err != nil {
		panic(err)
	}
	wifiAdapters := getWifiAdapters()
	stationObj := conn.Object(
		"net.connman.iwd",
		wifiAdapters[0].path,
	)
	var orderedNetworks []struct {
		Path     dbus.ObjectPath
		Strength int16
	}

	stationObj.Call(
		"net.connman.iwd.Station.GetOrderedNetworks",
		0,
	).Store(&orderedNetworks)

	networks := make([]wifiNetwork, 0)
	for path, ifaces := range managedObjects {
		network, err := ifaces["net.connman.iwd.Network"]
		if !err {
			continue
		}
		var signalStrength int16
		for _, value := range orderedNetworks {
			if value.Path == path {
				signalStrength = value.Strength
				break
			}
		}
		net := wifiNetwork{
			name:      network["Name"].Value().(string),
			connected: network["Connected"].Value().(bool),
			device:    network["Device"].String(),
			security:  network["Type"].Value().(string),
			strength:  2 * ((signalStrength / 100) + 100),
			path:      string(path),
		}
		networks = append(networks, net)
	}
	return networks
}

type wifiNetwork struct {
	name      string
	connected bool
	device    string
	security  string
	strength  int16
	path      string
}
