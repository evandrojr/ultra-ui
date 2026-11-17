package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ultra-ui/internal/config"
)

const mockClientsHead = `[
 {"address":"0x1","class":"foot","title":"foot","pid":1,"floating":false,"mapped":true,
  "at":[0,0],"size":[1920,1080],"workspace":{"id":5,"name":"5"},"monitor":0}`

func writeMockHyprctl(t *testing.T, dir string) {
	t.Helper()
	script := `#!/usr/bin/env bash
set -u
STATE="$(dirname "$0")"
key="${1:-}"
sub="${2:-}"
case "$key $sub" in
  "clients -j") cat "$STATE/clients.json" ;;
  "activeworkspace -j") cat "$STATE/activeworkspace.json" ;;
  "activeworkspace "*|"activeworkspace") echo "5" ;;
  "activewindow -j") cat "$STATE/activewindow.json" ;;
  "monitors -j") cat "$STATE/monitors.json" ;;
  "workspaces -j") cat "$STATE/workspaces.json" ;;
  "dispatch "*)
    echo "$*" >> "$STATE/dispatch.log"
    args="$*"
    if [[ $args == *'no_op'* ]]; then
      echo "ok"
      exit 0
    fi
    if [[ $args == *"exec"* ]]; then
      if [[ $args == *'exec_cmd('* ]]; then
        app=$(printf '%s' "$args" | sed -n 's/.*exec_cmd("\([^"]*\)").*/\1/p')
      else
        app="${args##*exec }"
      fi
      case "$app" in
        chromium*) cls="chromium"; addr="0xCHROMIUM" ;;
        thunar*)   cls="thunar";   addr="0xTHUNAR" ;;
        true*)     cls="teste-app"; addr="0xTESTE" ;;
        btop*)     cls="btop";     addr="0xBTOP" ;;
        *)         cls="$app";      addr="0x${app^^}" ;;
      esac
      python3 - "$STATE/clients.json" "$cls" "$addr" <<'PY'
import json,sys
path,cls,addr=sys.argv[1],sys.argv[2],sys.argv[3]
data=json.load(open(path))
data=[c for c in data if c["address"]!=addr]
data.append({"address":addr,"class":cls,"title":cls,"pid":4242,
             "floating":False,"mapped":True,"at":[0,0],"size":[1000,700],
             "workspace":{"id":5,"name":"5"},"monitor":0})
json.dump(data,open(path,"w"))
PY
    fi
    ;;
  *) echo "" ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "hyprctl"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeMockHerdr(t *testing.T, dir string) {
	t.Helper()
	script := `#!/usr/bin/env bash
set -u
STATE="$(dirname "$0")"
case "$*" in
  "status server --json") echo '{"status":"running","running":true}' ;;
  "workspace list") cat "$STATE/workspaces.json" ;;
  "tab list"|"tab list --workspace "*) cat "$STATE/tabs.json" ;;
  "pane current") cat "$STATE/current.json" ;;
  "tab create"*) echo '{"result":{"type":"tab_created","tab":{"tab_id":"w19:t2"},"pane":{"pane_id":"w19:p2"}}}' ;;
  "pane run"*) echo '{"result":{"type":"pane_command","ok":true}}' ;;
  "pane send-text"*) echo '{"result":{"type":"pane_command","ok":true}}' ;;
  "pane send-keys"*) echo '{"result":{"type":"pane_command","ok":true}}' ;;
  "tab close"*) echo '{"result":{"type":"tab_closed","ok":true}}' ;;
  "tab focus"*) echo '{"result":{"type":"tab_focused","ok":true}}' ;;
  "workspace focus"*) echo '{"result":{"type":"workspace_focused","ok":true}}' ;;
  *) echo '{}' ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "herdr"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func setup(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ULTRA_UI_HYPRCTL", filepath.Join(dir, "hyprctl"))
	t.Setenv("ULTRA_UI_HERDR", filepath.Join(dir, "herdr"))
	// O ambiente real (pane do herdr) injeta HERDR_BIN_PATH; sem limpa-lo os
	// testes batem no herdr real em vez do mock.
	t.Setenv("HERDR_BIN_PATH", "")
	t.Setenv("ULTRA_UI_STATE_DIR", filepath.Join(dir, "state"))
	t.Setenv("ULTRA_UI_CONFIG_DIR", dir)
	t.Setenv("HERDR_WORKSPACE_ID", "w19")
	t.Setenv("HERDR_TAB_ID", "w19:t1")

	writeMockHyprctl(t, dir)
	writeMockHerdr(t, dir)

	files := map[string]string{
		"ultra-ui.json":        `{"apps":{"teste":{"label":"App de teste","command":["true"],"window_class":"teste-app"}}}`,
		"clients.json":         mockClientsHead + "]",
		"activeworkspace.json": `{"id":5,"name":"5"}`,
		"activewindow.json":    `{"address":"0x1","class":"foot","title":"foot"}`,
		"monitors.json":        `[{"id":0,"name":"DP-1","width":1920,"height":1080,"focused":true}]`,
		"workspaces.json":      `{"result":{"type":"workspace_list","workspaces":[{"workspace_id":"w19","label":"ultra-ui","focused":true,"active_tab_id":"w19:t1","tab_count":1,"pane_count":1}]}}`,
		"tabs.json":            `{"result":{"type":"tab_list","tabs":[{"tab_id":"w19:t1","workspace_id":"w19","label":"1","focused":true,"pane_count":1}]}}`,
		"current.json":         `{"result":{"type":"pane_current","pane":{"pane_id":"w19:p1","tab_id":"w19:t1","workspace_id":"w19"}}}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	svc, err := NewService()
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, dir
}

func dispatchLog(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "dispatch.log"))
	if err != nil {
		return ""
	}
	return string(data)
}

func TestOpenAppCriaTabEEstado(t *testing.T) {
	svc, dir := setup(t)
	if err := svc.OpenApp("teste", "", ""); err != nil {
		t.Fatalf("OpenApp: %v", err)
	}
	g, ok := svc.st.Get("w19:t2")
	if !ok {
		t.Fatal("aba GUI nao registrada")
	}
	if g.WindowAddress != "0xTESTE" || g.App != "teste" {
		t.Fatalf("mapeamento = %+v", g)
	}
	log := dispatchLog(t, dir)
	if !strings.Contains(log, "true") {
		t.Fatalf("exec nao disparado:\n%s", log)
	}
	if !strings.Contains(log, "focus({ window = \"address:0xTESTE\" })") {
		t.Fatalf("foco nao disparado:\n%s", log)
	}
}

func TestFocusTabGUILevantaJanela(t *testing.T) {
	svc, dir := setup(t)
	if err := svc.OpenApp("teste", "", ""); err != nil {
		t.Fatalf("OpenApp: %v", err)
	}
	os.Remove(filepath.Join(dir, "dispatch.log"))
	if err := svc.FocusTab("w19:t2"); err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	log := dispatchLog(t, dir)
	if !strings.Contains(log, "focus({ window = \"address:0xTESTE\" })") {
		t.Fatalf("janela GUI nao focada:\n%s", log)
	}
}

func TestFocusTabTerminalEscondeGUI(t *testing.T) {
	svc, dir := setup(t)
	if err := svc.OpenApp("teste", "", ""); err != nil {
		t.Fatalf("OpenApp: %v", err)
	}
	os.Remove(filepath.Join(dir, "dispatch.log"))
	if err := svc.FocusTab("w19:t1"); err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	log := dispatchLog(t, dir)
	if !strings.Contains(log, "window.move({ workspace = \"special:ultra-ui\", window = \"address:0xTESTE\"") {
		t.Fatalf("GUI nao enviada ao scratchpad:\n%s", log)
	}
	if svc.st.ActiveGUI != "" {
		t.Fatalf("ActiveGUI = %q, esperado vazio", svc.st.ActiveGUI)
	}
}

func TestCloseCurrentFechaJanelaELimpaEstado(t *testing.T) {
	svc, dir := setup(t)
	if err := svc.OpenApp("teste", "", ""); err != nil {
		t.Fatalf("OpenApp: %v", err)
	}
	t.Setenv("HERDR_TAB_ID", "w19:t2")
	os.Remove(filepath.Join(dir, "dispatch.log"))
	if err := svc.CloseCurrent(); err != nil {
		t.Fatalf("CloseCurrent: %v", err)
	}
	if _, ok := svc.st.Get("w19:t2"); ok {
		t.Fatal("estado da GUI nao removido")
	}
	log := dispatchLog(t, dir)
	if !strings.Contains(log, "window.close({ window = \"address:0xTESTE\" })") {
		t.Fatalf("janela nao fechada:\n%s", log)
	}
}

func TestOpenAppSemSessaoGraficaFalha(t *testing.T) {
	svc, _ := setup(t)
	t.Setenv("ULTRA_UI_HYPRCTL", filepath.Join(t.TempDir(), "nao-existe"))
	err := svc.OpenApp("teste", "", "")
	if err == nil || !strings.Contains(err.Error(), "Hyprland") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestRestoreRecriaAbaQuandoJanenaSobrevive(t *testing.T) {
	svc, _ := setup(t)
	if err := svc.OpenApp("teste", "", ""); err != nil {
		t.Fatalf("OpenApp: %v", err)
	}
	// Restaura com a janela ainda presente.
	if err := svc.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if _, ok := svc.st.Get("w19:t2"); !ok {
		t.Fatal("aba GUI perdida no restore")
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := config.Default()
	if cfg.Scratchpad == "" || cfg.SidebarWidthPx <= 0 {
		t.Fatalf("defaults invalidos: %+v", cfg)
	}
	if len(cfg.Apps) == 0 {
		t.Fatal("catalogo de apps vazio")
	}
}
