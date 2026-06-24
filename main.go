package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mdlayher/wifi"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func main() {
	iface := getWifiAdapters()
	ScanForAccessPoints(iface[1])
	accessPoints := getKnownAccessPoints(iface[1])
	columns := []table.Column{
		{Title: "name", Width: 40},
		{Title: "Signal", Width: 40},
		{Title: "RSN", Width: 65},
	}
	rows := []table.Row{}
	for _, accessPoint := range accessPoints {
		rows = append(rows, table.Row{
			accessPoint.SSID,
			strconv.Itoa(int(2 * ((accessPoint.Signal / 100) + 100))),
			accessPoint.RSN.String(),
		})
	}
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(20),
		table.WithWidth(150),
	)
	t.SetStyles(s)
	t.Focus()
	p := tea.NewProgram(model{
		networkTable: t,
		selected:     -1,
	})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type model struct {
	networkTable table.Model
	selected     int
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":

		}
	}
	m.networkTable, cmd = m.networkTable.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return tea.NewView(baseStyle.Render(m.networkTable.View()) + "\n  " + m.networkTable.HelpView() + "\n")
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
