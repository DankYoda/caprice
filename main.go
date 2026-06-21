package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/godbus/dbus/v5"
)

func main() {
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
	stationPaths := getStations(managedObjects)
	discoverNetworks(conn, stationPaths[0])
	getNetworks(conn, stationPaths[0], managedObjects)

	//bubble tea here
	//p := tea.NewProgram(initialModel())
	//if _, err := p.Run(); err != nil {
	//	fmt.Printf("Alas, there's been an error: %v", err)
	//	os.Exit(1)
	//}
}

type model struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func initialModel() model {
	return model{
		// Our to-do list is a grocery list
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

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
			if m.cursor < len(m.choices)-1 {
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
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

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

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return tea.NewView(s)
}

func getStations(managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant) []dbus.ObjectPath {
	//Get the wlans
	var stationPaths []dbus.ObjectPath
	for path, ifaces := range managedObjects {
		if _, ok := ifaces["net.connman.iwd.Station"]; ok {
			stationPaths = append(stationPaths, path)
		}
	}
	return stationPaths
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

func getNetworks(conn *dbus.Conn, stationPath dbus.ObjectPath, managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant) []wifiNetwork {
	stationObj := conn.Object(
		"net.connman.iwd",
		stationPath,
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
			connType:  network["Type"].Value().(string),
			strength:  2 * ((signalStrength / 100) + 100),
			path:      string(path),
		}
		networks = append(networks, net)
		fmt.Println(net.name)
		fmt.Println(net.connected)
		fmt.Println(net.device)
		fmt.Println(net.strength)
		fmt.Println(net.path)
		fmt.Println(net.connType)
		fmt.Println("\n")
	}
	return networks
}

type wifiNetwork struct {
	name      string
	connected bool
	device    string
	connType  string
	strength  int16
	path      string
}
