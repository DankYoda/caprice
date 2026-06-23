package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mdlayher/wifi"
)

func main() {
	//interfaces := getWifiAdapters()
	//ScanForAccessPoints(interfaces[1])
	//bubble tea here
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type model struct {
	knownNetworks     []*wifi.BSS
	availableNetworks []*wifi.BSS
	adapters          []*wifi.Interface
	cursor            int
	selected          map[int]struct{}
}

func initialModel() model {
	interfaces := getWifiAdapters()
	return model{
		knownNetworks:     getKnownAccessPoints(interfaces[1]),
		availableNetworks: getKnownAccessPoints(interfaces[1]),
		adapters:          interfaces,
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
			if m.cursor < len(m.knownNetworks)-1 {
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
	s := "\nBehold, the known networks!\n\n"

	// Iterate over our choices
	for i, choice := range m.knownNetworks {

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
		s += fmt.Sprintf("%s [%s] %v\n", cursor, checked, choice)
	}

	s += "\nBehold, the Networks!\n\n"
	for _, network := range m.availableNetworks {
		s += fmt.Sprintf("%v\n", network)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return tea.NewView(s)
}

func getWifiAdapters() []*wifi.Interface {
	client, err := wifi.New()
	if err != nil {
		log.Fatalf("failed to open wifi client: %v", err)
	}
	defer client.Close()

	// Get all Wi-Fi interfaces
	interfaces, err := client.Interfaces()
	if err != nil || len(interfaces) == 0 {
		log.Fatalf("no wifi interfaces found: %v", err)
	}
	return interfaces
}

func ScanForAccessPoints(wifiInterface *wifi.Interface) {
	client, err := wifi.New()
	if err != nil {
		log.Fatalf("failed to open wifi client: %v", err)
	}
	defer client.Close()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(20*time.Second))
	defer cancel()
	err = client.Scan(ctx, wifiInterface)
	if err != nil {
		log.Fatalf("failed to get access points: %v", err)
	}
}

func getKnownAccessPoints(wifiInterface *wifi.Interface) []*wifi.BSS {
	client, err := wifi.New()
	if err != nil {
		log.Fatalf("failed to open wifi client: %v", err)
	}
	defer client.Close()
	accessPoints, err := client.AccessPoints(wifiInterface)

	if err != nil {
		log.Fatalf("failed to get access points: %v", err)
	}
	return accessPoints
}
