// Package menu envolve o seletor grafico do Omarchy (omarchy-menu-select),
// usado por apps e workspaces para escolhas rapidas na sessao grafica.
package menu

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Available indica se o seletor grafico existe no PATH.
func Available() bool {
	_, err := exec.LookPath("omarchy-menu-select")
	return err == nil
}

// Env devolve o ambiente minimo para o seletor grafico. O server do herdr nao
// tem WAYLAND_DISPLAY; descobrimos o socket do compositor em $XDG_RUNTIME_DIR.
func Env() []string {
	env := os.Environ()
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		runtime := os.Getenv("XDG_RUNTIME_DIR")
		if runtime == "" {
			runtime = "/run/user/" + strconv.Itoa(os.Getuid())
		}
		matches, _ := filepath.Glob(filepath.Join(runtime, "wayland-*"))
		if len(matches) > 0 {
			env = append(env, "WAYLAND_DISPLAY="+filepath.Base(matches[0]))
		}
	}
	if os.Getenv("OMARCHY_PATH") == "" {
		env = append(env, "OMARCHY_PATH=/usr/share/omarchy")
	}
	return env
}

// Select abre o menu grafico com as opcoes dadas e devolve o item escolhido
// (o texto exato da opcao). Opcoes podem ser "rotulo" ou
// "\trotulo\tsubtexto" — o subtexto aparece sob o rotulo, participa da busca
// e volta na selecao como "<rotulo>\t<subtexto>", servindo de chave estavel
// para rotulos repetidos.
func Select(prompt string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("nenhuma opcao")
	}
	path, err := exec.LookPath("omarchy-menu-select")
	if err != nil {
		return "", fmt.Errorf("menu indisponivel (%v)", err)
	}
	args := append([]string{prompt}, options...)
	cmd := exec.Command(path, args...)
	cmd.Env = Env()
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("menu cancelado ou indisponivel (%v)", err)
	}
	return strings.TrimSpace(string(out)), nil
}
