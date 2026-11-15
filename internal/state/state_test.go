package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ULTRA_UI_STATE_DIR", dir)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.GUITabs) != 0 {
		t.Fatalf("estado vazio esperado, veio %d", len(s.GUITabs))
	}
	s.Put(GUITab{
		WorkspaceID:   "w19",
		TabID:         "w19:t2",
		App:           "browser",
		Label:         "Chromium",
		HyprWorkspace: "5",
		WindowAddress: "0xabc",
		Anchored:      true,
	})
	s.ActiveGUI = "w19:t2"
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load 2: %v", err)
	}
	g, ok := got.Get("w19:t2")
	if !ok {
		t.Fatal("aba GUI perdida")
	}
	if g.WindowAddress != "0xabc" || g.Label != "Chromium" || !g.Anchored {
		t.Fatalf("dados divergentes: %+v", g)
	}
	if got.ActiveGUI != "w19:t2" {
		t.Fatalf("ActiveGUI = %q", got.ActiveGUI)
	}
}

func TestPruneRemoveWorkspacesFechados(t *testing.T) {
	s := &State{Version: 1, GUITabs: map[string]GUITab{
		"w1:t1": {WorkspaceID: "w1", TabID: "w1:t1"},
		"w2:t1": {WorkspaceID: "w2", TabID: "w2:t1"},
	}}
	s.Prune(map[string]bool{"w2": true})
	if _, ok := s.Get("w1:t1"); ok {
		t.Fatal("aba de workspace fechado deveria ser removida")
	}
	if _, ok := s.Get("w2:t1"); !ok {
		t.Fatal("aba de workspace ativo deveria permanecer")
	}
}

func TestSaveCreatesDir(t *testing.T) {
	base := filepath.Join(t.TempDir(), "nested", "state")
	t.Setenv("ULTRA_UI_STATE_DIR", base)
	s := &State{Version: 1, GUITabs: map[string]GUITab{}}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "state.json")); err != nil {
		t.Fatalf("state.json nao criado: %v", err)
	}
}
