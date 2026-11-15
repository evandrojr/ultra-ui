// Package state persiste o mapeamento aba -> janela grafica.
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// GUITab e uma aba do herdr associada a uma janela do Hyprland.
type GUITab struct {
	WorkspaceID   string `json:"workspace_id"`
	TabID         string `json:"tab_id"`
	App           string `json:"app"`
	Label         string `json:"label"`
	HyprWorkspace string `json:"hypr_workspace"`
	WindowAddress string `json:"window_address"`
	WindowClass   string `json:"window_class,omitempty"`
	Anchored      bool   `json:"anchored,omitempty"`
}

// State e o conteudo de <state-dir>/ultra-ui/state.json.
type State struct {
	Version   int               `json:"version"`
	GUITabs   map[string]GUITab `json:"gui_tabs"`
	ActiveGUI string            `json:"active_gui,omitempty"`
}

// Dir resolve o diretorio de estado: HERDR_PLUGIN_STATE_DIR ou
// ~/.local/state/ultra-ui.
func Dir() string {
	if dir := os.Getenv("HERDR_PLUGIN_STATE_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ultra-ui-state"
	}
	return filepath.Join(home, ".local", "state", "ultra-ui")
}

func path() string { return filepath.Join(Dir(), "state.json") }

// Load le o estado. Sem arquivo, retorna um estado vazio.
func Load() (*State, error) {
	s := &State{Version: 1, GUITabs: map[string]GUITab{}}
	data, err := os.ReadFile(path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.GUITabs == nil {
		s.GUITabs = map[string]GUITab{}
	}
	if s.Version == 0 {
		s.Version = 1
	}
	return s, nil
}

// Save grava o estado atomicamente (tmp + rename) sob um flock.
func (s *State) Save() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, "state-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path())
}

// Get retorna a GUI de uma tab.
func (s *State) Get(tabID string) (GUITab, bool) {
	g, ok := s.GUITabs[tabID]
	return g, ok
}

// Put registra (ou substitui) a GUI de uma tab.
func (s *State) Put(g GUITab) {
	s.GUITabs[g.TabID] = g
}

// Delete remove a GUI de uma tab.
func (s *State) Delete(tabID string) {
	delete(s.GUITabs, tabID)
}

// Prune remove entradas de workspaces que nao existem mais.
func (s *State) Prune(validWorkspaces map[string]bool) {
	for id, g := range s.GUITabs {
		if !validWorkspaces[g.WorkspaceID] {
			delete(s.GUITabs, id)
		}
	}
}
