// Package workspaces implementa a busca rapida de workspaces ("spaces") do
// herdr: lista os workspaces, deixa o usuario digitar o nome num seletor e
// devolve o workspace_id escolhido para focar.
package workspaces

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ultra-ui/internal/herdr"
	"ultra-ui/internal/menu"
)

// Pick abre o seletor de workspaces e devolve o workspace_id escolhido.
func Pick() (string, error) {
	ws, err := herdr.Workspaces()
	if err != nil {
		return "", fmt.Errorf("listar workspaces: %w", err)
	}
	if len(ws) == 0 {
		return "", errors.New("nenhum workspace")
	}
	if menu.Available() {
		chosen, err := menu.Select("Ir para workspace", buildOptions(ws))
		if err != nil {
			return "", err
		}
		id := resolve(chosen, ws)
		if id == "" {
			return "", fmt.Errorf("selecao invalida: %q", chosen)
		}
		return id, nil
	}
	return pickTerminal(ws)
}

// buildOptions monta as opcoes do menu: o rotulo como texto visivel e o
// numero como subtexto estavel, que desambigua rotulos repetidos e vira a
// chave de volta para o workspace_id.
func buildOptions(ws []herdr.WorkspaceSummary) []string {
	options := make([]string, 0, len(ws))
	for _, w := range ws {
		options = append(options, fmt.Sprintf("\t%s\t#%d", w.Label, w.Number))
	}
	return options
}

// resolve transforma a selecao do menu ("<rotulo>\t#<numero>") de volta em
// workspace_id. Tambem aceita um rotulo puro como fallback.
func resolve(chosen string, ws []herdr.WorkspaceSummary) string {
	chosen = strings.TrimSpace(chosen)
	if i := strings.LastIndex(chosen, "\t#"); i >= 0 {
		n, err := strconv.Atoi(chosen[i+2:])
		if err == nil {
			for _, w := range ws {
				if w.Number == n {
					return w.WorkspaceID
				}
			}
		}
	}
	for _, w := range ws {
		if w.Label == chosen {
			return w.WorkspaceID
		}
	}
	return ""
}

// pickTerminal e o fallback sem sessao grafica: lista numerada no stdout.
func pickTerminal(ws []herdr.WorkspaceSummary) (string, error) {
	fmt.Println("Workspaces:")
	for i, w := range ws {
		cur := ""
		if w.Focused {
			cur = " *"
		}
		fmt.Printf("  %d) %s%s\n", i+1, w.Label, cur)
	}
	fmt.Print("Escolha: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	for i, w := range ws {
		if line == fmt.Sprintf("%d", i+1) || strings.EqualFold(line, w.Label) {
			return w.WorkspaceID, nil
		}
	}
	return "", fmt.Errorf("escolha invalida: %q", line)
}
