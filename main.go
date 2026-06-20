package main

import (
	"fmt"
	"os"

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
	initializeNetworks(managedObjects)

	//bubble tea here
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
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
		if err != nil {
			break
		}
	}
}

func initializeNetworks(managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant) []wifiNetwork {
	networks := make([]wifiNetwork, 0)
	for _, ifaces := range managedObjects {
		network, err := ifaces["net.connman.iwd.Network"]
		if !err {
			continue
		}
		networks = append(networks, wifiNetwork{
			name:      network["Name"].String(),
			connected: network["Connected"].Value().(bool),
			device:    network["Device"].String(),
			connType:  network["Type"].String(),
		},
		)
	}
	return networks
}

type wifiNetwork struct {
	name      string
	connected bool
	device    string
	connType  string
}
