// Package apps knowns o catalogo de apps que podem virar abas GUI.
package apps

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"ultra-ui/internal/config"
)

// Entry e um item do catalogo.
type Entry struct {
	ID      string
	Label   string
	Command []string
	Class   string
}

// List devolve o catalogo ordenado por rotulo, sem apps indisponiveis.
func List(cfg config.Config) []Entry {
	entries := make([]Entry, 0, len(cfg.Apps))
	for id, app := range cfg.Apps {
		if len(app.Command) == 0 {
			continue
		}
		if _, err := exec.LookPath(app.Command[0]); err != nil {
			continue
		}
		entries = append(entries, Entry{
			ID:      id,
			Label:   app.Label,
			Command: app.Command,
			Class:   app.WindowClass,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Label) < strings.ToLower(entries[j].Label)
	})
	return entries
}

// env devolve o ambiente para o seletor grafico. O server do herdr nao tem
// WAYLAND_DISPLAY; descobrimos o socket do compositor.
func env() []string {
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

// Pick abre um seletor interativo e devolve o ID escolhido.
func Pick(cfg config.Config) (string, error) {
	entries := List(cfg)
	if len(entries) == 0 {
		return "", fmt.Errorf("nenhum app configurado")
	}
	if path, err := exec.LookPath("omarchy-menu-select"); err == nil {
		args := []string{"Abrir app grafico"}
		for _, e := range entries {
			args = append(args, e.Label)
		}
		cmd := exec.Command(path, args...)
		cmd.Env = env()
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err == nil {
			chosen := strings.TrimSpace(string(out))
			for _, e := range entries {
				if e.Label == chosen {
					return e.ID, nil
				}
			}
			// Alguns menus devolvem o indice ou texto extra; tenta casar por prefixo.
			for _, e := range entries {
				if strings.HasPrefix(chosen, e.Label) {
					return e.ID, nil
				}
			}
			return "", fmt.Errorf("selecao invalida: %q", chosen)
		}
		return "", fmt.Errorf("menu indisponivel (%v); use --app", err)
	}
	return pickTerminal(entries)
}

func pickTerminal(entries []Entry) (string, error) {
	fmt.Println("Apps disponiveis:")
	for i, e := range entries {
		fmt.Printf("  %d) %s\n", i+1, e.Label)
	}
	fmt.Print("Escolha: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	for i, e := range entries {
		if line == fmt.Sprintf("%d", i+1) || strings.EqualFold(line, e.Label) {
			return e.ID, nil
		}
	}
	return "", fmt.Errorf("escolha invalida: %q", line)
}
