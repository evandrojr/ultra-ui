// Package cli implementa a interface de linha de comando do ultra-ui.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"ultra-ui/internal/apps"
	"ultra-ui/internal/clipboard"
	"ultra-ui/internal/config"
	"ultra-ui/internal/gui"
	"ultra-ui/internal/herdr"
	"ultra-ui/internal/hypr"
	"ultra-ui/internal/state"
)

const usage = `ultra-ui — abas GUI do Hyprland dentro do herdr

Uso:
  ultra-ui startup                     restaura abas GUI (hook do plugin)
  ultra-ui event <nome>                processa um evento do herdr
  ultra-ui gui open [--app id] [--label txt] [--cwd dir]
  ultra-ui gui close                   fecha a GUI da aba atual
  ultra-ui gui detach                  desacopla a GUI (a aba volta a ser terminal)
  ultra-ui gui pick                    abre o seletor de apps
  ultra-ui gui list                    lista as abas GUI
  ultra-ui gui sentinel <tab> <addr>   sentinela do painel (uso interno)
  ultra-ui status                      mostra o mapeamento atual
  ultra-ui doctor                      verifica dependencias
  ultra-ui clipboard copy|paste        copia/cola via clipboard do Wayland
`

// Run executa um comando e devolve o codigo de saida.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Print(usage)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return 0
	case "startup":
		return wrap(func() error { return mustService().Restore() })
	case "event":
		if len(rest) == 0 {
			return fail("event: nome do evento ausente")
		}
		return wrap(func() error { return mustService().HandleEvent(rest[0]) })
	case "gui":
		return runGUI(rest)
	case "tab":
		if len(rest) == 0 {
			return fail("tab: prev|next")
		}
		switch rest[0] {
		case "prev":
			return wrap(func() error { return mustService().SwitchTab(-1) })
		case "next":
			return wrap(func() error { return mustService().SwitchTab(1) })
		default:
			return fail("tab: prev|next")
		}
	case "workspace":
		if len(rest) == 0 {
			return fail("workspace: prev|next")
		}
		switch rest[0] {
		case "prev":
			return wrap(func() error { return mustService().SwitchWorkspace(-1) })
		case "next":
			return wrap(func() error { return mustService().SwitchWorkspace(1) })
		default:
			return fail("workspace: prev|next")
		}
	case "status":
		return runStatus()
	case "doctor":
		return runDoctor()
	case "clipboard":
		return runClipboard(rest)
	default:
		return fail("comando desconhecido: %s", cmd)
	}
}

func runGUI(args []string) int {
	if len(args) == 0 {
		return fail("gui: subcomando ausente")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "open":
		appID, label, cwd := parseOpenFlags(rest)
		if appID == "" {
			return fail("gui open: --app e obrigatorio")
		}
		return wrap(func() error { return mustService().OpenApp(appID, label, cwd) })
	case "close":
		return wrap(func() error { return mustService().CloseCurrent() })
	case "anchor":
		return wrap(func() error { return mustService().AnchorCurrent() })
	case "detach":
		return wrap(func() error { return mustService().DetachCurrent() })
	case "pick":
		svc := mustService()
		id, err := apps.Pick(svc.Config())
		if err != nil {
			return fail("gui pick: %v", err)
		}
		return wrap(func() error { return svc.OpenApp(id, "", "") })
	case "list":
		return runList()
	case "sentinel":
		if len(rest) < 2 {
			return fail("gui sentinel: <tab> <address>")
		}
		gui.Sentinel(rest[0], rest[1])
		return 0
	default:
		return fail("gui: subcomando desconhecido: %s", sub)
	}
}

func parseOpenFlags(args []string) (appID, label, cwd string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--app":
			if i+1 < len(args) {
				appID = args[i+1]
				i++
			}
		case "--label":
			if i+1 < len(args) {
				label = args[i+1]
				i++
			}
		case "--cwd":
			if i+1 < len(args) {
				cwd = args[i+1]
				i++
			}
		}
	}
	return
}

func runList() int {
	st, err := state.Load()
	if err != nil {
		return fail("status: %v", err)
	}
	if len(st.GUITabs) == 0 {
		fmt.Println("nenhuma aba GUI")
		return 0
	}
	ids := make([]string, 0, len(st.GUITabs))
	for id := range st.GUITabs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		g := st.GUITabs[id]
		active := ""
		if st.ActiveGUI == id {
			active = " *"
		}
		fmt.Printf("%-14s %-24s %s -> %s (%s)%s\n", g.TabID, g.Label, g.WorkspaceID, g.WindowAddress, g.App, active)
	}
	return 0
}

func runStatus() int {
	st, err := state.Load()
	if err != nil {
		return fail("status: %v", err)
	}
	cfg, err := config.Load(config.Dir())
	if err != nil {
		return fail("status: %v", err)
	}
	fmt.Printf("estado:      %s\n", stateFile())
	fmt.Printf("config:      %s\n", config.Dir())
	fmt.Printf("workspaces:  %d abas GUI\n", len(st.GUITabs))
	fmt.Printf("clipboard:   %s\n", cfg.Scratchpad)
	if hypr.Available() {
		if ws, err := hypr.ActiveWorkspace(); err == nil {
			fmt.Printf("hypr:        workspace ativo %s\n", ws.Name)
		}
	} else {
		fmt.Println("hypr:        indisponivel (fora de sessao grafica)")
	}
	return 0
}

func runDoctor() int {
	checks := []struct {
		name string
		ok   bool
		info string
	}{
		{"herdr", herdr.Available(), herdr.Bin()},
		{"hyprctl", hypr.Available(), hypr.Bin()},
		{"wl-copy", look("wl-copy") != "", ""},
		{"wl-paste", look("wl-paste") != "", ""},
		{"menu", look("omarchy-menu-select") != "", "opcional"},
	}
	ok := true
	for _, c := range checks {
		status := "ok"
		if !c.ok {
			status = "ausente"
			if c.info != "opcional" {
				ok = false
			}
		}
		fmt.Printf("%-18s %-8s %s\n", c.name, status, c.info)
	}
	if !ok {
		return 1
	}
	return 0
}

func runClipboard(args []string) int {
	if len(args) == 0 {
		return fail("clipboard: copy|paste")
	}
	paneID, _, _, err := herdr.PaneCurrent()
	if err != nil {
		return fail("clipboard: painel atual indisponivel: %v", err)
	}
	switch args[0] {
	case "copy":
		return wrap(func() error { return clipboard.Copy(paneID) })
	case "paste":
		return wrap(func() error { return clipboard.Paste(paneID) })
	default:
		return fail("clipboard: copy|paste")
	}
}

func look(bin string) string {
	path, err := exec.LookPath(bin)
	if err != nil {
		return ""
	}
	return path
}

func mustService() *gui.Service {
	svc, err := gui.NewService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ultra-ui: %v\n", err)
		os.Exit(1)
	}
	return svc
}

func wrap(fn func() error) int {
	if err := fn(); err != nil {
		return fail("%v", err)
	}
	return 0
}

func fail(format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "ultra-ui: "+format+"\n", a...)
	return 1
}

func stateFile() string {
	return state.Dir() + "/state.json"
}
