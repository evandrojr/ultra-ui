// Package clipboard implementa copiar/colar universal via Wayland.
package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"ultra-ui/internal/herdr"
)

// Copy pede ao painel que copie a selecao. TUIs e terminais costumam tratar
// ctrl+shift+c; quando o terminal ja tem Super+c, o binding do foot entra
// antes e este caminho vira redundante.
func Copy(paneID string) error {
	if paneID == "" {
		return fmt.Errorf("painel ausente")
	}
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return fmt.Errorf("wl-copy indisponivel")
	}
	return herdr.PaneSendKeys(paneID, "ctrl+shift+c")
}

// Paste cola o clipboard do Wayland no painel como texto literal.
func Paste(paneID string) error {
	if paneID == "" {
		return fmt.Errorf("painel ausente")
	}
	path, err := exec.LookPath("wl-paste")
	if err != nil {
		return fmt.Errorf("wl-paste indisponivel")
	}
	out := &bytes.Buffer{}
	cmd := exec.Command(path, "--no-newline")
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-paste: %w", err)
	}
	text := strings.TrimRight(out.String(), "\n")
	if text == "" {
		return nil
	}
	return herdr.PaneSendText(paneID, text)
}
