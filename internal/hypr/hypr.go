// Package hypr e um wrapper enxuto sobre o binario hyprctl.
package hypr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Bin resolve o binario: ULTRA_UI_HYPRCTL (teste) ou hyprctl.
func Bin() string {
	if b := os.Getenv("ULTRA_UI_HYPRCTL"); b != "" {
		return b
	}
	return "hyprctl"
}

// Available indica se ha uma sessao Hyprland usavel.
func Available() bool {
	out, err := run("activeworkspace")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// Client e uma janela do Hyprland.
type Client struct {
	Address   string `json:"address"`
	Class     string `json:"class"`
	Title     string `json:"title"`
	Pid       int    `json:"pid"`
	Floating  bool   `json:"floating"`
	Mapped    bool   `json:"mapped"`
	At        [2]int `json:"at"`
	Size      [2]int `json:"size"`
	Workspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"workspace"`
	Monitor int `json:"monitor"`
}

// Workspace e um workspace do Hyprland.
type Workspace struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Monitor string `json:"monitor"`
	Windows int    `json:"windows"`
}

// ActiveWindowInfo e a janela atualmente focada.
type ActiveWindowInfo struct {
	Address  string `json:"address"`
	Class    string `json:"class"`
	Title    string `json:"title"`
	Pid      int    `json:"pid"`
	At       [2]int `json:"at"`
	Size     [2]int `json:"size"`
	Floating bool   `json:"floating"`
}

func run(args ...string) (string, error) {
	cmd := exec.Command(Bin(), args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), errors.New(msg)
	}
	return out.String(), nil
}

// Dispatch executa um dispatch do Hyprland.
func Dispatch(args ...string) error {
	_, err := run(append([]string{"dispatch"}, args...)...)
	return err
}

// DispatchSilent executa um dispatch sem falhar quando a janela/workspace
// nao existe mais.
func DispatchSilent(args ...string) {
	_, _ = run(append([]string{"dispatch"}, args...)...)
}

// Clients lista as janelas.
func Clients() ([]Client, error) {
	out, err := run("clients", "-j")
	if err != nil {
		return nil, err
	}
	var clients []Client
	if err := json.Unmarshal([]byte(out), &clients); err != nil {
		return nil, fmt.Errorf("hyprctl clients: %w", err)
	}
	return clients, nil
}

// Workspaces lista os workspaces.
func Workspaces() ([]Workspace, error) {
	out, err := run("workspaces", "-j")
	if err != nil {
		return nil, err
	}
	var ws []Workspace
	if err := json.Unmarshal([]byte(out), &ws); err != nil {
		return nil, fmt.Errorf("hyprctl workspaces: %w", err)
	}
	return ws, nil
}

// ActiveWorkspace retorna o workspace ativo.
func ActiveWorkspace() (Workspace, error) {
	out, err := run("activeworkspace", "-j")
	if err != nil {
		return Workspace{}, err
	}
	var ws Workspace
	if err := json.Unmarshal([]byte(out), &ws); err != nil {
		return Workspace{}, fmt.Errorf("hyprctl activeworkspace: %w", err)
	}
	return ws, nil
}

// ActiveWindow retorna a janela ativa.
func ActiveWindow() (ActiveWindowInfo, error) {
	out, err := run("activewindow", "-j")
	if err != nil {
		return ActiveWindowInfo{}, err
	}
	var w ActiveWindowInfo
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		return ActiveWindowInfo{}, fmt.Errorf("hyprctl activewindow: %w", err)
	}
	return w, nil
}

// Monitor e um monitor do Hyprland.
type Monitor struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Focused bool   `json:"focused"`
}

// ActiveMonitor retorna o monitor focado.
func ActiveMonitor() (Monitor, error) {
	out, err := run("monitors", "-j")
	if err != nil {
		return Monitor{}, err
	}
	var monitors []Monitor
	if err := json.Unmarshal([]byte(out), &monitors); err != nil {
		return Monitor{}, fmt.Errorf("hyprctl monitors: %w", err)
	}
	for _, m := range monitors {
		if m.Focused {
			return m, nil
		}
	}
	if len(monitors) > 0 {
		return monitors[0], nil
	}
	return Monitor{}, errors.New("nenhum monitor")
}

// Find localiza uma janela pelo address.
func Find(address string) (Client, bool) {
	clients, err := Clients()
	if err != nil {
		return Client{}, false
	}
	for _, c := range clients {
		if c.Address == address {
			return c, true
		}
	}
	return Client{}, false
}

// FindByClass localiza a janela mais recente de uma classe.
func FindByClass(class string) (Client, bool) {
	clients, err := Clients()
	if err != nil {
		return Client{}, false
	}
	for i := len(clients) - 1; i >= 0; i-- {
		if strings.EqualFold(clients[i].Class, class) && clients[i].Mapped {
			return clients[i], true
		}
	}
	return Client{}, false
}

// FocusWindow traz uma janela para o workspace e a foca.
func FocusWindow(address, workspace string) error {
	if workspace != "" {
		DispatchSilent("movetoworkspace", workspace+",address:"+address)
	}
	if err := Dispatch("focuswindow", "address:"+address); err != nil {
		return err
	}
	return nil
}

// HideWindow manda a janela para o scratchpad.
func HideWindow(address, scratchpad string) {
	if scratchpad == "" {
		scratchpad = "special:ultra-ui"
	}
	DispatchSilent("movetoworkspacesilent", scratchpad+",address:"+address)
}

// CloseWindow fecha a janela.
func CloseWindow(address string) error {
	return Dispatch("closewindow", "address:"+address)
}

// WaitNewWindow aguarda uma janela nova aparecer e a devolve.
func WaitNewWindow(before map[string]bool, timeout time.Duration) (Client, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		clients, err := Clients()
		if err == nil {
			for _, c := range clients {
				if !before[c.Address] && c.Mapped {
					return c, true
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return Client{}, false
}

// WaitClass aguarda uma janela da classe aparecer.
func WaitClass(class string, timeout time.Duration) (Client, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c, ok := FindByClass(class); ok {
			return c, true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return Client{}, false
}
