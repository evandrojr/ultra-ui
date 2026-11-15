// Package config carrega as preferencias do ultra-ui.
package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
)

// App descreve um app grafico que pode virar uma aba.
type App struct {
	Label       string   `json:"label"`
	Command     []string `json:"command"`
	WindowClass string   `json:"window_class,omitempty"`
	// Floating força a janela a abrir flutuante (padrao: true para abas GUI).
	Floating *bool `json:"floating,omitempty"`
}

// Config sao as preferencias persistidas em <config-dir>/ultra-ui.json.
type Config struct {
	// WorkspacePrefix prefixa os workspaces do Hyprland criados pelo plugin.
	WorkspacePrefix string `json:"workspace_prefix"`
	// Scratchpad e o workspace especial para onde as GUIs sao movidas quando
	// uma aba de terminal e focada.
	Scratchpad string `json:"scratchpad"`
	// SidebarWidthPx e a largura reservada a barra lateral do herdr, usada
	// para ancorar a GUI na area principal.
	SidebarWidthPx int `json:"sidebar_width_px"`
	// Anchor ativa o posicionamento automatico da janela GUI.
	Anchor bool `json:"anchor"`
	// Apps e o catalogo de apps disponiveis no seletor.
	Apps map[string]App `json:"apps"`
}

// Default retorna a configuracao padrao, incluindo apenas apps instalados.
func Default() Config {
	cfg := Config{
		WorkspacePrefix: "ultra:",
		Scratchpad:      "special:ultra-ui",
		SidebarWidthPx:  240,
		Anchor:          true,
		Apps:            map[string]App{},
	}
	add := func(id, label, class string, command ...string) {
		if len(command) == 0 {
			return
		}
		if _, err := exec.LookPath(command[0]); err == nil {
			cfg.Apps[id] = App{Label: label, Command: command, WindowClass: class}
		}
	}
	add("browser", "Chromium", "chromium", "chromium")
	add("files", "Dolphin", "org.kde.dolphin", "dolphin")
	add("files-thunar", "Arquivos (thunar)", "thunar", "thunar")
	add("editor", "Zed", "dev.zed.Zed", "zed")
	add("btop", "btop", "btop", "btop")
	add("imv", "imv", "imv", "imv")
	add("lazygit", "lazygit", "lazygit", "lazygit")
	return cfg
}

// Dir resolve o diretorio de config: HERDR_PLUGIN_CONFIG_DIR ou
// ~/.config/ultra-ui.
func Dir() string {
	if dir := os.Getenv("HERDR_PLUGIN_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ultra-ui"
	}
	return filepath.Join(home, ".config", "ultra-ui")
}

// Load le <dir>/ultra-ui.json por cima dos padroes. Ausencia de arquivo nao
// e erro: assegura apenas que o catalogo de apps existe.
func Load(dir string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(filepath.Join(dir, "ultra-ui.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	var onDisk struct {
		WorkspacePrefix *string        `json:"workspace_prefix"`
		Scratchpad      *string        `json:"scratchpad"`
		SidebarWidthPx  *int           `json:"sidebar_width_px"`
		Anchor          *bool          `json:"anchor"`
		Apps            map[string]App `json:"apps"`
	}
	if err := json.Unmarshal(data, &onDisk); err != nil {
		return cfg, err
	}
	if onDisk.WorkspacePrefix != nil {
		cfg.WorkspacePrefix = *onDisk.WorkspacePrefix
	}
	if onDisk.Scratchpad != nil {
		cfg.Scratchpad = *onDisk.Scratchpad
	}
	if onDisk.SidebarWidthPx != nil {
		cfg.SidebarWidthPx = *onDisk.SidebarWidthPx
	}
	if onDisk.Anchor != nil {
		cfg.Anchor = *onDisk.Anchor
	}
	for id, app := range onDisk.Apps {
		cfg.Apps[id] = app
	}
	return cfg, nil
}
