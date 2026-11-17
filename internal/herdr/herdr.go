// Package herdr e um wrapper sobre o binario do herdr.
package herdr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Bin resolve o binario: HERDR_BIN_PATH (injetado pelo herdr), depois
// ULTRA_UI_HERDR (teste), depois herdr.
func Bin() string {
	if b := os.Getenv("HERDR_BIN_PATH"); b != "" {
		return b
	}
	if b := os.Getenv("ULTRA_UI_HERDR"); b != "" {
		return b
	}
	return "herdr"
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

// Run executa um comando e devolve a saida bruta.
func Run(args ...string) (string, error) { return run(args...) }

// Available indica se o CLI responde.
func Available() bool {
	_, err := run("status", "server", "--json")
	return err == nil
}

// WorkspaceSummary e um workspace do herdr.
type WorkspaceSummary struct {
	WorkspaceID string `json:"workspace_id"`
	Number      int    `json:"number"`
	Label       string `json:"label"`
	Focused     bool   `json:"focused"`
	ActiveTabID string `json:"active_tab_id"`
	TabCount    int    `json:"tab_count"`
}

// TabSummary e uma aba do herdr.
type TabSummary struct {
	TabID       string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
	Focused     bool   `json:"focused"`
}

type workspaceListResult struct {
	Result struct {
		Workspaces []WorkspaceSummary `json:"workspaces"`
	} `json:"result"`
}

// Workspaces lista os workspaces.
func Workspaces() ([]WorkspaceSummary, error) {
	out, err := run("workspace", "list")
	if err != nil {
		return nil, err
	}
	var r workspaceListResult
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return nil, fmt.Errorf("herdr workspace list: %w", err)
	}
	return r.Result.Workspaces, nil
}

// WorkspaceFocus foca um workspace.
func WorkspaceFocus(workspaceID string) error {
	_, err := run("workspace", "focus", workspaceID)
	return err
}

// FocusedWorkspaceID devolve o workspace focado.
func FocusedWorkspaceID() (string, error) {
	ws, err := Workspaces()
	if err != nil {
		return "", err
	}
	for _, w := range ws {
		if w.Focused {
			return w.WorkspaceID, nil
		}
	}
	if len(ws) > 0 {
		return ws[0].WorkspaceID, nil
	}
	return "", errors.New("nenhum workspace no herdr")
}

type tabListResult struct {
	Result struct {
		Tabs []TabSummary `json:"tabs"`
	} `json:"result"`
}

// Tabs lista as abas (opcionalmente de um workspace).
func Tabs(workspaceID string) ([]TabSummary, error) {
	args := []string{"tab", "list"}
	if workspaceID != "" {
		args = append(args, "--workspace", workspaceID)
	}
	out, err := run(args...)
	if err != nil {
		return nil, err
	}
	var r tabListResult
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return nil, fmt.Errorf("herdr tab list: %w", err)
	}
	return r.Result.Tabs, nil
}

// TabFocusedID devolve a aba focada do workspace.
func TabFocusedID(workspaceID string) (string, error) {
	tabs, err := Tabs(workspaceID)
	if err != nil {
		return "", err
	}
	for _, t := range tabs {
		if t.Focused {
			return t.TabID, nil
		}
	}
	if len(tabs) > 0 {
		return tabs[0].TabID, nil
	}
	return "", errors.New("nenhuma aba no workspace")
}

// TabCreate cria uma aba e devolve o ID da aba e do painel.
func TabCreate(workspaceID, label, cwd string, focus bool) (tabID, paneID string, err error) {
	args := []string{"tab", "create"}
	if workspaceID != "" {
		args = append(args, "--workspace", workspaceID)
	}
	if label != "" {
		args = append(args, "--label", label)
	}
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	if focus {
		args = append(args, "--focus")
	} else {
		args = append(args, "--no-focus")
	}
	out, err := run(args...)
	if err != nil {
		return "", "", err
	}
	var r struct {
		Result struct {
			Tab struct {
				TabID string `json:"tab_id"`
			} `json:"tab"`
			RootPane struct {
				PaneID string `json:"pane_id"`
			} `json:"root_pane"`
			Pane struct {
				PaneID string `json:"pane_id"`
			} `json:"pane"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", "", fmt.Errorf("herdr tab create: %w", err)
	}
	paneID = r.Result.RootPane.PaneID
	if paneID == "" {
		paneID = r.Result.Pane.PaneID
	}
	return r.Result.Tab.TabID, paneID, nil
}

// TabFocus foca uma aba.
func TabFocus(tabID string) error {
	_, err := run("tab", "focus", tabID)
	return err
}

// TabClose fecha uma aba.
func TabClose(tabID string) error {
	_, err := run("tab", "close", tabID)
	return err
}

// TabRename renomeia uma aba.
func TabRename(tabID, label string) error {
	_, err := run("tab", "rename", tabID, "--label", label)
	return err
}

// PaneSendKeys envia teclas a um painel.
func PaneSendKeys(paneID string, keys ...string) error {
	args := append([]string{"pane", "send-keys", paneID}, keys...)
	_, err := run(args...)
	return err
}

// PaneRun executa um comando num painel.
func PaneRun(paneID string, command ...string) error {
	args := append([]string{"pane", "run", paneID}, command...)
	_, err := run(args...)
	return err
}

// PaneSendText envia texto literal para um painel.
func PaneSendText(paneID, text string) error {
	_, err := run("pane", "send-text", paneID, text)
	return err
}

// PaneCurrent devolve o pane_id, tab_id e workspace_id atuais.
func PaneCurrent() (paneID, tabID, workspaceID string, err error) {
	out, err := run("pane", "current")
	if err != nil {
		return "", "", "", err
	}
	var r struct {
		Result struct {
			Pane struct {
				PaneID      string `json:"pane_id"`
				TabID       string `json:"tab_id"`
				WorkspaceID string `json:"workspace_id"`
			} `json:"pane"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", "", "", fmt.Errorf("herdr pane current: %w", err)
	}
	return r.Result.Pane.PaneID, r.Result.Pane.TabID, r.Result.Pane.WorkspaceID, nil
}

// EnvWorkspaceID devolve o workspace injetado no ambiente, se houver.
func EnvWorkspaceID() string {
	if id := os.Getenv("HERDR_WORKSPACE_ID"); id != "" {
		return id
	}
	if ctx := os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"); ctx != "" {
		var m map[string]any
		if json.Unmarshal([]byte(ctx), &m) == nil {
			if id, _ := m["workspace_id"].(string); id != "" {
				return id
			}
		}
	}
	return ""
}

// EnvTabID devolve a aba injetada no ambiente, se houver.
func EnvTabID() string {
	if id := os.Getenv("HERDR_TAB_ID"); id != "" {
		return id
	}
	if ctx := os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"); ctx != "" {
		var m map[string]any
		if json.Unmarshal([]byte(ctx), &m) == nil {
			if id, _ := m["tab_id"].(string); id != "" {
				return id
			}
		}
	}
	return ""
}
