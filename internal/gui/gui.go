// Package gui implementa a logica das abas GUI do ultra-ui.
package gui

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ultra-ui/internal/config"
	"ultra-ui/internal/herdr"
	"ultra-ui/internal/hypr"
	"ultra-ui/internal/state"
)

type Service struct {
	cfg config.Config
	st  *state.State
}

// NewService carrega config e estado.
func NewService() (*Service, error) {
	cfg, err := config.Load(config.Dir())
	if err != nil {
		return nil, err
	}
	st, err := state.Load()
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg, st: st}, nil
}

// Config expoe a configuracao carregada.
func (s *Service) Config() config.Config { return s.cfg }

func (s *Service) save() {
	if err := s.st.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "ultra-ui: ao salvar estado: %v\n", err)
	}
}

// currentWorkspaceID resolve o workspace do herdr alvo.
func (s *Service) currentWorkspaceID() (string, error) {
	if id := herdr.EnvWorkspaceID(); id != "" {
		return id, nil
	}
	return herdr.FocusedWorkspaceID()
}

// currentTabID resolve a aba alvo.
func (s *Service) currentTabID() (string, error) {
	if id := herdr.EnvTabID(); id != "" {
		return id, nil
	}
	if _, tabID, _, err := herdr.PaneCurrent(); err == nil && tabID != "" {
		return tabID, nil
	}
	ws, err := s.currentWorkspaceID()
	if err != nil {
		return "", err
	}
	return herdr.TabFocusedID(ws)
}

// OpenApp cria uma aba GUI no workspace atual.
func (s *Service) OpenApp(appID, label, cwd string) error {
	if !hypr.Available() {
		return fmt.Errorf("sessao Hyprland indisponivel ( rode em uma sessao grafica )")
	}
	app, ok := s.cfg.Apps[appID]
	if !ok {
		return fmt.Errorf("app desconhecido %q; disponiveis: %s", appID, strings.Join(appIDs(s.cfg), ", "))
	}
	if len(app.Command) == 0 {
		return fmt.Errorf("app %q sem comando", appID)
	}
	if _, err := exec.LookPath(app.Command[0]); err != nil {
		return fmt.Errorf("comando %q nao encontrado para o app %q", app.Command[0], app.Label)
	}
	if label == "" {
		label = app.Label
	}
	wsID, err := s.currentWorkspaceID()
	if err != nil {
		return err
	}

	before := map[string]bool{}
	if clients, err := hypr.Clients(); err == nil {
		for _, c := range clients {
			before[c.Address] = true
		}
	}

	tabID, paneID, err := herdr.TabCreate(wsID, label, cwd, true)
	if err != nil {
		return fmt.Errorf("criar aba: %w", err)
	}

	if err := hypr.Exec(strings.Join(app.Command, " ")); err != nil {
		_ = herdr.TabClose(tabID)
		return fmt.Errorf("executar %q: %w", app.Command[0], err)
	}

	var win hypr.Client
	found := bool(false)
	// Primeiro procura uma janela genuinamente nova; so entao aceita a classe,
	// para nao associar uma janela que ja estava aberta.
	win, found = hypr.WaitNewWindow(before, 5*time.Second)
	if !found && app.WindowClass != "" {
		win, found = hypr.WaitClass(app.WindowClass, before, 3*time.Second)
	}
	if !found {
		// Ultima tentativa: a janela ativa pode ser a recem-aberta.
		if aw, err := hypr.ActiveWindow(); err == nil && !before[aw.Address] {
			if c, ok := hypr.Find(aw.Address); ok {
				win, found = c, true
			}
		}
	}
	if !found {
		_ = herdr.TabClose(tabID)
		return fmt.Errorf("janela do app %q nao apareceu; verifique se o comando abre uma janela", app.Label)
	}

	ws, _ := hypr.ActiveWorkspace()
	g := state.GUITab{
		WorkspaceID:   wsID,
		TabID:         tabID,
		App:           appID,
		Label:         label,
		HyprWorkspace: ws.Name,
		WindowAddress: win.Address,
		WindowClass:   win.Class,
	}
	if g.WindowClass == "" {
		g.WindowClass = app.WindowClass
	}
	s.st.Put(g)
	s.st.ActiveGUI = tabID
	s.save()

	if paneID != "" {
		_ = herdr.PaneRun(paneID, "ultra-ui", "gui", "sentinel", tabID, win.Address)
	}
	if err := s.raise(tabID); err != nil {
		fmt.Fprintf(os.Stderr, "ultra-ui: foco da janela: %v\n", err)
	}
	return nil
}

// raise move a janela para o workspace do terminal do herdr e a foca.
func (s *Service) raise(tabID string) error {
	g, ok := s.st.Get(tabID)
	if !ok {
		return nil
	}
	wsName := s.terminalWorkspace()
	if wsName == "" {
		ws, err := hypr.ActiveWorkspace()
		if err != nil {
			return err
		}
		wsName = ws.Name
	}
	if err := hypr.FocusWindow(g.WindowAddress, wsName); err != nil {
		return err
	}
	s.anchor(g.WindowAddress)
	g.Anchored = true
	g.HyprWorkspace = wsName
	s.st.Put(g)
	s.st.ActiveGUI = tabID
	s.save()
	return nil
}

// terminalWorkspace devolve o workspace do terminal que roda o herdr.
func (s *Service) terminalWorkspace() string {
	if term, ok := findTerminalWindow(); ok {
		return term.Workspace.Name
	}
	return ""
}

// anchor posiciona a janela na area principal do terminal, preservando a
// sidebar do herdr a esquerda.
func (s *Service) anchor(address string) {
	term, ok := findTerminalWindow()
	x, y, w, h := 0, 0, 0, 0
	if ok {
		x = term.At[0] + s.cfg.SidebarWidthPx
		y = term.At[1]
		w = term.Size[0] - s.cfg.SidebarWidthPx
		h = term.Size[1]
	} else if m, err := hypr.ActiveMonitor(); err == nil {
		x = s.cfg.SidebarWidthPx
		y = 0
		w = m.Width - s.cfg.SidebarWidthPx
		h = m.Height
	}
	if w <= 0 || h <= 0 {
		return
	}
	hypr.Float(address)
	hypr.MoveExact(address, x, y)
	hypr.ResizeExact(address, w, h)
	// O resize anima/desloca; aguarda assentar e reposiciona.
	time.Sleep(300 * time.Millisecond)
	hypr.MoveExact(address, x, y)
}

// FocusTab trata o evento tab.focused.
func (s *Service) FocusTab(tabID string) error {
	if tabID == "" {
		var err error
		tabID, err = s.currentTabID()
		if err != nil {
			return err
		}
	}
	if _, ok := s.st.Get(tabID); ok {
		if err := s.raise(tabID); err != nil {
			return err
		}
		return nil
	}
	// Aba de terminal: esconde as GUIs visiveis do workspace, independente do
	// ActiveGUI (que pode ter sido limpo por um evento anterior).
	wsID := herdr.EnvWorkspaceID()
	if wsID == "" {
		var err error
		wsID, err = s.currentWorkspaceID()
		if err != nil {
			return err
		}
	}
	for _, g := range s.st.GUITabs {
		if g.WorkspaceID != wsID {
			continue
		}
		if c, ok := hypr.Find(g.WindowAddress); ok && c.Workspace.Name == s.cfg.Scratchpad {
			continue
		}
		hypr.HideWindow(g.WindowAddress, s.cfg.Scratchpad)
	}
	s.st.ActiveGUI = ""
	s.save()
	return nil
}

// HideActive manda a GUI ativa para o scratchpad.
func (s *Service) HideActive() {
	if s.st.ActiveGUI == "" {
		return
	}
	if g, ok := s.st.Get(s.st.ActiveGUI); ok {
		hypr.HideWindow(g.WindowAddress, s.cfg.Scratchpad)
	}
	s.st.ActiveGUI = ""
	s.save()
}

// CloseCurrent fecha a janela GUI da aba atual.
func (s *Service) CloseCurrent() error {
	tabID, err := s.currentTabID()
	if err != nil {
		return err
	}
	g, ok := s.st.Get(tabID)
	if !ok {
		return fmt.Errorf("a aba atual nao e uma GUI")
	}
	if err := hypr.CloseWindow(g.WindowAddress); err != nil {
		// A janela pode ja ter fechado; segue o baile.
		_ = err
	}
	s.st.Delete(tabID)
	if s.st.ActiveGUI == tabID {
		s.st.ActiveGUI = ""
	}
	s.save()
	// Se a aba ficou sem painel, fecha a aba.
	if tabs, err := herdr.Tabs(g.WorkspaceID); err == nil {
		for _, t := range tabs {
			if t.TabID == tabID {
				return nil
			}
		}
	}
	_ = herdr.TabClose(tabID)
	return nil
}

// DetachCurrent desacopla a GUI e devolve a aba ao shell.
func (s *Service) DetachCurrent() error {
	tabID, err := s.currentTabID()
	if err != nil {
		return err
	}
	g, ok := s.st.Get(tabID)
	if !ok {
		return fmt.Errorf("a aba atual nao e uma GUI")
	}
	s.st.Delete(tabID)
	if s.st.ActiveGUI == tabID {
		s.st.ActiveGUI = ""
	}
	s.save()
	hypr.HideWindow(g.WindowAddress, s.cfg.Scratchpad)
	return nil
}

// SwitchTab foca a aba anterior (dir < 0) ou proxima (dir > 0) do workspace.
func (s *Service) SwitchTab(dir int) error {
	wsID := herdr.EnvWorkspaceID()
	if wsID == "" {
		var err error
		wsID, err = s.currentWorkspaceID()
		if err != nil {
			return err
		}
	}
	tabs, err := herdr.Tabs(wsID)
	if err != nil {
		return err
	}
	if len(tabs) == 0 {
		return fmt.Errorf("nenhuma aba no workspace")
	}
	current := 0
	for i, t := range tabs {
		if t.Focused {
			current = i
			break
		}
	}
	next := (current + dir + len(tabs)) % len(tabs)
	return herdr.TabFocus(tabs[next].TabID)
}

// SwitchWorkspace foca o workspace anterior (dir < 0) ou proximo (dir > 0).
func (s *Service) SwitchWorkspace(dir int) error {
	ws, err := herdr.Workspaces()
	if err != nil {
		return err
	}
	if len(ws) == 0 {
		return fmt.Errorf("nenhum workspace")
	}
	current := 0
	for i, w := range ws {
		if w.Focused {
			current = i
			break
		}
	}
	next := (current + dir + len(ws)) % len(ws)
	_, err = herdr.Run("workspace", "focus", ws[next].WorkspaceID)
	return err
}

// HandleEvent processa os hooks do herdr.
func (s *Service) HandleEvent(event string) error {
	debug := os.Getenv("ULTRA_UI_DEBUG") != ""
	if !debug {
		if _, err := os.Stat(filepath.Join(state.Dir(), "debug")); err == nil {
			debug = true
		}
	}
	if debug {
		if f, err := os.OpenFile(filepath.Join(state.Dir(), "debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "event=%s tab=%q ws=%q ctx=%s\n",
				event, os.Getenv("HERDR_TAB_ID"), os.Getenv("HERDR_WORKSPACE_ID"),
				os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"))
			f.Close()
		}
	}
	tabID := herdr.EnvTabID()
	workspaceID := herdr.EnvWorkspaceID()
	switch event {
	case "tab.focused":
		return s.FocusTab(tabID)
	case "workspace.focused":
		if workspaceID == "" {
			return nil
		}
		if g, ok := s.st.Get(s.st.ActiveGUI); ok && g.WorkspaceID != workspaceID {
			hypr.HideWindow(g.WindowAddress, s.cfg.Scratchpad)
			s.st.ActiveGUI = ""
			s.save()
		}
		return nil
	case "tab.closed":
		if tabID == "" {
			return nil
		}
		if g, ok := s.st.Get(tabID); ok {
			_ = hypr.CloseWindow(g.WindowAddress)
			s.st.Delete(tabID)
			if s.st.ActiveGUI == tabID {
				s.st.ActiveGUI = ""
			}
			s.save()
		}
		return nil
	case "pane.exited":
		return nil
	}
	return nil
}

// Restore reconstrói as abas GUI depois do restore do herdr.
func (s *Service) Restore() error {
	if !hypr.Available() {
		return nil
	}
	wsList, err := herdr.Workspaces()
	if err != nil {
		return err
	}
	valid := map[string]bool{}
	for _, w := range wsList {
		valid[w.WorkspaceID] = true
	}
	s.st.Prune(valid)

	for _, g := range s.st.GUITabs {
		win, ok := hypr.Find(g.WindowAddress)
		if !ok {
			app, known := s.cfg.Apps[g.App]
			if !known {
				s.st.Delete(g.TabID)
				continue
			}
			if err := hypr.Exec(strings.Join(app.Command, " ")); err != nil {
				continue
			}
			if win, ok = hypr.WaitNewWindow(nil, 5*time.Second); !ok {
				if win, ok = hypr.WaitClass(app.WindowClass, nil, 5*time.Second); !ok {
					continue
				}
			}
			g.WindowAddress = win.Address
			g.WindowClass = win.Class
			g.HyprWorkspace = win.Workspace.Name
			s.st.Put(g)
		}
		// Garante a aba e o sentinela.
		tabs, _ := herdr.Tabs(g.WorkspaceID)
		exists := false
		for _, t := range tabs {
			if t.TabID == g.TabID {
				exists = true
				break
			}
		}
		if !exists {
			tabID, paneID, err := herdr.TabCreate(g.WorkspaceID, g.Label, "", false)
			if err != nil {
				continue
			}
			g.TabID = tabID
			s.st.Put(g)
			if paneID != "" {
				_ = herdr.PaneRun(paneID, "ultra-ui", "gui", "sentinel", tabID, g.WindowAddress)
			}
		}
	}
	s.save()
	return nil
}

// Sentinel fica no painel de uma aba GUI e sai quando a janela fecha.
func Sentinel(tabID, address string) {
	fmt.Printf("ultra-ui: %s nesta aba (Alt+Setas para alternar)\n", tabID)
	if address != "" {
		for {
			time.Sleep(500 * time.Millisecond)
			if _, ok := hypr.Find(address); !ok {
				return
			}
		}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	<-ch
}

func appIDs(cfg config.Config) []string {
	ids := make([]string, 0, len(cfg.Apps))
	for id := range cfg.Apps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// findTerminalWindow descobre o terminal que roda o herdr: sobe a arvore de
// cada processo "herdr" (clientes) ate achar um PID que tenha janela no
// Hyprland. Funciona tanto no hook (server) quanto na CLI manual via SSH.
func findTerminalWindow() (hypr.Client, bool) {
	clients, err := hypr.Clients()
	if err != nil {
		return hypr.Client{}, false
	}
	byPid := map[int]hypr.Client{}
	for _, c := range clients {
		byPid[c.Pid] = c
	}
	for _, pid := range herdrProcesses() {
		for p := pid; p > 1; {
			if c, ok := byPid[p]; ok {
				return c, true
			}
			next, ok := parentPid(p)
			if !ok || next == p {
				break
			}
			p = next
		}
	}
	return hypr.Client{}, false
}

func herdrProcesses() []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		comm := string(data)
		if i := strings.IndexByte(comm, '('); i >= 0 {
			if j := strings.IndexByte(comm[i:], ')'); j >= 0 {
				comm = comm[i+1 : i+j]
			}
		}
		if comm == "herdr" {
			pids = append(pids, pid)
		}
	}
	return pids
}

func parentPid(pid int) (int, bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 4 {
		return 0, false
	}
	next, err := strconv.Atoi(fields[3])
	if err != nil {
		return 0, false
	}
	return next, true
}
