package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	nm "github.com/Wifx/gonetworkmanager/v3"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func main() {
	networkManager, err := nm.NewNetworkManager()
	if err != nil {
		log.Fatalf("failed to connect to NetworkManager: %v", err)
	}
	adapters := getWifiAdapters(networkManager)
	adapter := adapters[0]
	columns := []table.Column{
		{Title: "name", Width: 40},
		{Title: "Signal", Width: 40},
		{Title: "Security", Width: 65},
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
		table.WithHeight(20),
		table.WithWidth(150),
	)
	t.SetStyles(s)
	t.Focus()
	passwordInput := textinput.New()
	passwordInput.Prompt = "password: "
	passwordInput.EchoMode = textinput.EchoPassword
	p := tea.NewProgram(model{
		passwordInput:  passwordInput,
		networkTable:   t,
		selected:       -1,
		networkManager: networkManager,
		adapter:        adapter,
		status:         "loading networks...",
	})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type model struct {
	networkTable   table.Model
	selected       int
	networkManager nm.NetworkManager
	adapter        nm.DeviceWireless
	networks       []network
	scanning       bool
	status         string
	passwordInput  textinput.Model
	// prompting is set while the password input has focus; pending is the
	// network the password is for.
	prompting bool
	pending   network
}

// network is one SSID, represented by its strongest access point.
type network struct {
	ssid        string
	strength    uint8
	security    string
	accessPoint nm.AccessPoint
}

type scanResultMsg struct {
	networks []network
	err      error
}

type connectResultMsg struct {
	network network
	err     error
}

func (m model) Init() tea.Cmd {
	// Show NetworkManager's cached results without triggering a new scan.
	return loadCmd(m.adapter)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.prompting {
		if key, ok := msg.(tea.KeyPressMsg); ok {
			switch key.String() {
			case "esc":
				m.prompting = false
				m.passwordInput.Blur()
				m.passwordInput.Reset()
				m.networkTable.Focus()
				m.status = "cancelled"
				return m, nil
			case "enter":
				password := m.passwordInput.Value()
				m.prompting = false
				m.passwordInput.Blur()
				m.passwordInput.Reset()
				m.networkTable.Focus()
				m.status = "connecting to " + m.pending.ssid + "..."
				return m, connectWithPasswordCmd(m.networkManager, m.adapter, m.pending, password)
			}
		}
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "s":
			if m.scanning {
				return m, nil
			}
			m.scanning = true
			m.status = "scanning..."
			return m, scanCmd(m.adapter)
		case "enter":
			m.selected = m.networkTable.Cursor()
			if m.selected < 0 || m.selected >= len(m.networks) {
				return m, nil
			}
			selected := m.networks[m.selected]
			if findSavedConnection(selected.ssid) == nil && keyMgmt(selected.accessPoint) != "" {
				if !supportsPassword(selected.accessPoint) {
					m.status = selected.ssid + ": " + selected.security + " networks are not supported yet"
					return m, nil
				}
				return m.promptPassword(selected, "enter password for "+selected.ssid+" (esc to cancel)")
			}
			m.status = "connecting to " + selected.ssid + "..."
			return m, connectCmd(m.networkManager, m.adapter, selected)
		}
	case scanResultMsg:
		m.scanning = false
		if msg.err != nil {
			m.status = fmt.Sprintf("scan failed: %v", msg.err)
			return m, nil
		}
		m.networks = msg.networks
		rows := []table.Row{}
		for _, n := range m.networks {
			rows = append(rows, table.Row{n.ssid, strconv.Itoa(int(n.strength)), n.security})
		}
		m.networkTable.SetRows(rows)
		m.networkTable.SetCursor(0)
		m.status = fmt.Sprintf("%d networks (s to scan, enter to connect)", len(m.networks))
		return m, nil
	case connectResultMsg:
		if msg.err == nil {
			m.status = "connected to " + msg.network.ssid
			return m, nil
		}
		// A saved profile with a stale password, or a mistyped new one: ask again.
		if supportsPassword(msg.network.accessPoint) {
			return m.promptPassword(msg.network, fmt.Sprintf("failed to connect to %s: %v. Re-enter password (esc to cancel)", msg.network.ssid, msg.err))
		}
		m.status = fmt.Sprintf("failed to connect to %s: %v", msg.network.ssid, msg.err)
		return m, nil
	}
	m.networkTable, cmd = m.networkTable.Update(msg)
	return m, cmd
}

func (m model) promptPassword(n network, status string) (tea.Model, tea.Cmd) {
	m.pending = n
	m.prompting = true
	m.status = status
	m.networkTable.Blur()
	return m, m.passwordInput.Focus()
}

func (m model) View() tea.View {
	view := baseStyle.Render(m.networkTable.View()) + "\n  " + m.networkTable.HelpView() + "\n  " + m.status + "\n"
	if m.prompting {
		view += "  " + m.passwordInput.View() + "\n"
	}
	return tea.NewView(view)
}

func loadCmd(adapter nm.DeviceWireless) tea.Cmd {
	return func() tea.Msg {
		networks, err := getNetworks(adapter)
		return scanResultMsg{networks: networks, err: err}
	}
}

func scanCmd(adapter nm.DeviceWireless) tea.Cmd {
	return func() tea.Msg {
		ScanForAccessPoints(adapter)
		networks, err := getNetworks(adapter)
		return scanResultMsg{networks: networks, err: err}
	}
}

func connectCmd(networkManager nm.NetworkManager, adapter nm.DeviceWireless, n network) tea.Cmd {
	return func() tea.Msg {
		return connectResultMsg{network: n, err: connect(networkManager, adapter, n)}
	}
}

func connectWithPasswordCmd(networkManager nm.NetworkManager, adapter nm.DeviceWireless, n network, password string) tea.Cmd {
	return func() tea.Msg {
		return connectResultMsg{network: n, err: connectWithPassword(networkManager, adapter, n, password)}
	}
}

// connect activates a saved profile for the SSID if NetworkManager has one,
// otherwise it creates a new profile for an open (or OWE) network.
func connect(networkManager nm.NetworkManager, adapter nm.DeviceWireless, n network) error {
	if saved := findSavedConnection(n.ssid); saved != nil {
		activeConnection, err := networkManager.ActivateWirelessConnection(saved, adapter, n.accessPoint)
		if err != nil {
			return err
		}
		return waitForActivation(activeConnection)
	}
	settings := map[string]map[string]interface{}{}
	if mgmt := keyMgmt(n.accessPoint); mgmt != "" {
		settings["802-11-wireless-security"] = map[string]interface{}{"key-mgmt": mgmt}
	}
	return addAndActivate(networkManager, adapter, n, settings)
}

// connectWithPassword replaces any saved profile for the SSID with a new one
// using the given password.
func connectWithPassword(networkManager nm.NetworkManager, adapter nm.DeviceWireless, n network, password string) error {
	for saved := findSavedConnection(n.ssid); saved != nil; saved = findSavedConnection(n.ssid) {
		if err := saved.Delete(); err != nil {
			return fmt.Errorf("failed to remove old profile: %w", err)
		}
	}
	return addAndActivate(networkManager, adapter, n, map[string]map[string]interface{}{
		"802-11-wireless-security": {
			"key-mgmt": keyMgmt(n.accessPoint),
			"psk":      password,
		},
	})
}

// addAndActivate creates a profile and activates it. NetworkManager saves the
// profile immediately, so it is deleted again if activation fails; otherwise a
// wrong password would stick around and be reused next time.
func addAndActivate(networkManager nm.NetworkManager, adapter nm.DeviceWireless, n network, settings map[string]map[string]interface{}) error {
	activeConnection, err := networkManager.AddAndActivateWirelessConnection(settings, adapter, n.accessPoint)
	if err != nil {
		return err
	}
	profile, _ := activeConnection.GetPropertyConnection()
	if err := waitForActivation(activeConnection); err != nil {
		if profile != nil {
			_ = profile.Delete()
		}
		return err
	}
	return nil
}

// waitForActivation blocks until the connection is up or has failed.
// Activate* returns as soon as NetworkManager starts connecting, so without
// this a wrong password would still look like success.
func waitForActivation(activeConnection nm.ActiveConnection) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		state, err := activeConnection.GetPropertyState()
		if err != nil {
			// The active connection object disappears once it is torn down.
			return fmt.Errorf("connection failed (wrong password?)")
		}
		switch state {
		case nm.NmActiveConnectionStateActivated:
			return nil
		case nm.NmActiveConnectionStateDeactivating, nm.NmActiveConnectionStateDeactivated:
			return fmt.Errorf("connection failed (wrong password?)")
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for connection")
}

func findSavedConnection(ssid string) nm.Connection {
	settings, err := nm.NewSettings()
	if err != nil {
		return nil
	}
	connections, err := settings.ListConnections()
	if err != nil {
		return nil
	}
	for _, connection := range connections {
		connectionSettings, err := connection.GetSettings()
		if err != nil {
			continue
		}
		wireless, ok := connectionSettings["802-11-wireless"]
		if !ok {
			continue
		}
		if raw, ok := wireless["ssid"].([]byte); ok && string(raw) == ssid {
			return connection
		}
	}
	return nil
}

func getWifiAdapters(networkManager nm.NetworkManager) []nm.DeviceWireless {
	devices, err := networkManager.GetPropertyAllDevices()
	if err != nil {
		log.Fatalf("failed to list devices: %v", err)
	}
	var adapters []nm.DeviceWireless
	for _, device := range devices {
		deviceType, err := device.GetPropertyDeviceType()
		if err != nil || deviceType != nm.NmDeviceTypeWifi {
			continue
		}
		adapter, err := nm.NewDeviceWireless(device.GetPath())
		if err != nil {
			continue
		}
		adapters = append(adapters, adapter)
	}
	if len(adapters) == 0 {
		log.Fatalf("no wifi interfaces found")
	}
	return adapters
}

// ScanForAccessPoints asks NetworkManager to scan and waits for LastScan to
// change. NetworkManager rate-limits scans, so a rejected request is not fatal;
// we just fall back to its cached results.
func ScanForAccessPoints(adapter nm.DeviceWireless) {
	before, _ := adapter.GetPropertyLastScan()
	if err := adapter.RequestScan(); err != nil {
		return
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if last, err := adapter.GetPropertyLastScan(); err == nil && last != before {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// getNetworks collapses access points into one entry per SSID. A single
// network often has several access points (2.4/5 GHz radios, mesh nodes), so
// we keep the strongest one. Hidden networks with no SSID are skipped.
func getNetworks(adapter nm.DeviceWireless) ([]network, error) {
	accessPoints, err := adapter.GetAccessPoints()
	if err != nil {
		return nil, fmt.Errorf("failed to get access points: %w", err)
	}
	bySSID := map[string]int{}
	var networks []network
	for _, accessPoint := range accessPoints {
		ssid, err := accessPoint.GetPropertySSID()
		if err != nil || ssid == "" {
			continue
		}
		strength, _ := accessPoint.GetPropertyStrength()
		if i, ok := bySSID[ssid]; ok {
			if strength > networks[i].strength {
				networks[i].strength = strength
				networks[i].accessPoint = accessPoint
				networks[i].security = securityString(accessPoint)
			}
			continue
		}
		bySSID[ssid] = len(networks)
		networks = append(networks, network{
			ssid:        ssid,
			strength:    strength,
			security:    securityString(accessPoint),
			accessPoint: accessPoint,
		})
	}
	sort.Slice(networks, func(i, j int) bool { return networks[i].strength > networks[j].strength })
	return networks, nil
}

// keyMgmt returns the NetworkManager key-mgmt value for an access point, or
// "" for an open network.
func keyMgmt(accessPoint nm.AccessPoint) string {
	flags, _ := accessPoint.GetPropertyFlags()
	wpaFlags, _ := accessPoint.GetPropertyWPAFlags()
	rsnFlags, _ := accessPoint.GetPropertyRSNFlags()
	rsn := nm.Nm80211APSec(rsnFlags)
	switch {
	case rsn&nm.Nm80211APSecKeyMgmt8021X != 0 || nm.Nm80211APSec(wpaFlags)&nm.Nm80211APSecKeyMgmt8021X != 0:
		return "wpa-eap"
	case rsn&nm.Nm80211APSecKeyMgmtPSK != 0 || nm.Nm80211APSec(wpaFlags)&nm.Nm80211APSecKeyMgmtPSK != 0:
		// Also covers WPA2/WPA3 transition mode.
		return "wpa-psk"
	case rsn&nm.Nm80211APSecKeyMgmtSAE != 0:
		return "sae"
	case rsn&(nm.Nm80211APSecKeyMgmtOWE|nm.Nm80211APSecKeyMgmtOWETM) != 0:
		return "owe"
	case nm.Nm80211APFlags(flags)&nm.Nm80211APFlagsPrivacy != 0:
		return "none" // WEP
	}
	return ""
}

// supportsPassword reports whether we can connect by asking for a single
// password. Enterprise (802.1X) and WEP networks are not handled.
func supportsPassword(accessPoint nm.AccessPoint) bool {
	mgmt := keyMgmt(accessPoint)
	return mgmt == "wpa-psk" || mgmt == "sae"
}

func securityString(accessPoint nm.AccessPoint) string {
	flags, _ := accessPoint.GetPropertyFlags()
	wpaFlags, _ := accessPoint.GetPropertyWPAFlags()
	rsnFlags, _ := accessPoint.GetPropertyRSNFlags()
	var parts []string
	if nm.Nm80211APFlags(flags)&nm.Nm80211APFlagsPrivacy != 0 && wpaFlags == 0 && rsnFlags == 0 {
		parts = append(parts, "WEP")
	}
	if wpaFlags != 0 {
		parts = append(parts, "WPA")
	}
	rsn := nm.Nm80211APSec(rsnFlags)
	if rsn&nm.Nm80211APSecKeyMgmtSAE != 0 {
		parts = append(parts, "WPA3")
	}
	if rsn&nm.Nm80211APSecKeyMgmtPSK != 0 {
		parts = append(parts, "WPA2")
	}
	if rsn&nm.Nm80211APSecKeyMgmt8021X != 0 {
		parts = append(parts, "802.1X")
	}
	if rsn&(nm.Nm80211APSecKeyMgmtOWE|nm.Nm80211APSecKeyMgmtOWETM) != 0 {
		parts = append(parts, "OWE")
	}
	if len(parts) == 0 {
		return "open"
	}
	return strings.Join(parts, " ")
}
