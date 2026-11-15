// Package hypr e um wrapper enxuto sobre o binario hyprctl.
package hypr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	luaOnce sync.Once
	luaMode bool
)

// LuaMode indica se o Hyprland avalia `hyprctl dispatch` como Lua (0.56 com
// config Lua). Nesse modo a forma legada `dispatch exec <cmd>` nao funciona.
func LuaMode() bool {
	luaOnce.Do(func() {
		out, err := run("dispatch", `hl.dsp.no_op()`)
		luaMode = err == nil && strings.Contains(strings.ToLower(out), "ok")
	})
	return luaMode
}

func luaString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func luaDispatch(expr string) error {
	_, err := run("dispatch", expr)
	return err
}

// Exec roda um comando no Hyprland.
func Exec(cmd string) error {
	if LuaMode() {
		return luaDispatch("hl.dsp.exec_cmd(" + luaString(cmd) + ")")
	}
	_, err := run("dispatch", "exec", cmd)
	return err
}

// Bin resolve o binario: ULTRA_UI_HYPRCTL (teste) ou hyprctl.
func Bin() string {
	if b := os.Getenv("ULTRA_UI_HYPRCTL"); b != "" {
		return b
	}
	return "hyprctl"
}

// Available indica se ha uma sessao Hyprland usavel.
func Available() bool {
	out, err := run("activeworkspace")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// Client e uma janela do Hyprland.
type Client struct {
	Address   string `json:"address"`
	Class     string `json:"class"`
	Title     string `json:"title"`
	Pid       int    `json:"pid"`
	Floating  bool   `json:"floating"`
	Mapped    bool   `json:"mapped"`
	At        [2]int `json:"at"`
	Size      [2]int `json:"size"`
	Workspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"workspace"`
	Monitor int `json:"monitor"`
}

// Workspace e um workspace do Hyprland.
type Workspace struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Monitor string `json:"monitor"`
	Windows int    `json:"windows"`
}

// ActiveWindowInfo e a janela atualmente focada.
type ActiveWindowInfo struct {
	Address  string `json:"address"`
	Class    string `json:"class"`
	Title    string `json:"title"`
	Pid      int    `json:"pid"`
	At       [2]int `json:"at"`
	Size     [2]int `json:"size"`
	Floating bool   `json:"floating"`
}

func run(args ...string) (string, error) {
	cmd := exec.Command(Bin(), args...)
	cmd.Env = hyprEnv()
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

// hyprEnv devolve o ambiente para o hyprctl. O server do herdr roda sem
// HYPRLAND_INSTANCE_SIGNATURE; quando ela falta, descobre a instancia ativa
// em $XDG_RUNTIME_DIR/hypr e a injeta no subprocess.
func hyprEnv() []string {
	env := os.Environ()
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return env
	}
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = "/run/user/" + strconv.Itoa(os.Getuid())
	}
	matches, err := filepath.Glob(filepath.Join(runtime, "hypr", "*"))
	if err != nil || len(matches) == 0 {
		return env
	}
	sort.Slice(matches, func(i, j int) bool {
		fi, errI := os.Stat(matches[i])
		fj, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return matches[i] > matches[j]
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return append(env, "HYPRLAND_INSTANCE_SIGNATURE="+filepath.Base(matches[0]))
}

// Dispatch executa um dispatch do Hyprland.
func Dispatch(args ...string) error {
	_, err := run(append([]string{"dispatch"}, args...)...)
	return err
}

// DispatchSilent executa um dispatch sem falhar quando a janela/workspace
// nao existe mais.
func DispatchSilent(args ...string) {
	out, err := run(append([]string{"dispatch"}, args...)...)
	if err != nil {
		debugf("dispatch %v: %v (out=%q)", args, err, out)
	}
}

// debugf registra diagnostico quando o arquivo de debug existe no state dir.
func debugf(format string, a ...any) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".local", "state", "ultra-ui")
	if _, err := os.Stat(filepath.Join(dir, "debug")); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "hypr-debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", a...)
}

// Clients lista as janelas.
func Clients() ([]Client, error) {
	out, err := run("clients", "-j")
	if err != nil {
		return nil, err
	}
	var clients []Client
	if err := json.Unmarshal([]byte(out), &clients); err != nil {
		return nil, fmt.Errorf("hyprctl clients: %w", err)
	}
	return clients, nil
}

// Workspaces lista os workspaces.
func Workspaces() ([]Workspace, error) {
	out, err := run("workspaces", "-j")
	if err != nil {
		return nil, err
	}
	var ws []Workspace
	if err := json.Unmarshal([]byte(out), &ws); err != nil {
		return nil, fmt.Errorf("hyprctl workspaces: %w", err)
	}
	return ws, nil
}

// ActiveWorkspace retorna o workspace ativo.
func ActiveWorkspace() (Workspace, error) {
	out, err := run("activeworkspace", "-j")
	if err != nil {
		return Workspace{}, err
	}
	var ws Workspace
	if err := json.Unmarshal([]byte(out), &ws); err != nil {
		return Workspace{}, fmt.Errorf("hyprctl activeworkspace: %w", err)
	}
	return ws, nil
}

// ActiveWindow retorna a janela ativa.
func ActiveWindow() (ActiveWindowInfo, error) {
	out, err := run("activewindow", "-j")
	if err != nil {
		return ActiveWindowInfo{}, err
	}
	var w ActiveWindowInfo
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		return ActiveWindowInfo{}, fmt.Errorf("hyprctl activewindow: %w", err)
	}
	return w, nil
}

// Monitor e um monitor do Hyprland.
type Monitor struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Focused bool   `json:"focused"`
}

// ActiveMonitor retorna o monitor focado.
func ActiveMonitor() (Monitor, error) {
	out, err := run("monitors", "-j")
	if err != nil {
		return Monitor{}, err
	}
	var monitors []Monitor
	if err := json.Unmarshal([]byte(out), &monitors); err != nil {
		return Monitor{}, fmt.Errorf("hyprctl monitors: %w", err)
	}
	for _, m := range monitors {
		if m.Focused {
			return m, nil
		}
	}
	if len(monitors) > 0 {
		return monitors[0], nil
	}
	return Monitor{}, errors.New("nenhum monitor")
}

// Find localiza uma janela pelo address.
func Find(address string) (Client, bool) {
	clients, err := Clients()
	if err != nil {
		return Client{}, false
	}
	for _, c := range clients {
		if c.Address == address {
			return c, true
		}
	}
	return Client{}, false
}

// FindByClass localiza a janela mais recente de uma classe.
func FindByClass(class string) (Client, bool) {
	clients, err := Clients()
	if err != nil {
		return Client{}, false
	}
	for i := len(clients) - 1; i >= 0; i-- {
		if strings.EqualFold(clients[i].Class, class) && clients[i].Mapped {
			return clients[i], true
		}
	}
	return Client{}, false
}

// FocusWindow traz uma janela para o workspace e a foca.
func FocusWindow(address, workspace string) error {
	sel := "address:" + address
	if LuaMode() {
		if workspace != "" {
			DispatchSilent("hl.dsp.window.move({ workspace = " + luaString(workspace) +
				", window = " + luaString(sel) + ", follow = false })")
		}
		return luaDispatch("hl.dsp.focus({ window = " + luaString(sel) + " })")
	}
	if workspace != "" {
		DispatchSilent("movetoworkspace", workspace+",address:"+address)
	}
	if err := Dispatch("focuswindow", "address:"+address); err != nil {
		return err
	}
	return nil
}

// HideWindow manda a janela para o scratchpad.
func HideWindow(address, scratchpad string) {
	if scratchpad == "" {
		scratchpad = "special:ultra-ui"
	}
	sel := "address:" + address
	if LuaMode() {
		DispatchSilent("hl.dsp.window.move({ workspace = " + luaString(scratchpad) +
			", window = " + luaString(sel) + ", follow = false })")
		return
	}
	DispatchSilent("movetoworkspacesilent", scratchpad+",address:"+address)
}

// CloseWindow fecha a janela.
func CloseWindow(address string) error {
	sel := "address:" + address
	if LuaMode() {
		return luaDispatch("hl.dsp.window.close({ window = " + luaString(sel) + " })")
	}
	return Dispatch("closewindow", "address:"+address)
}

// Float alterna o estado flutuante de uma janela.
func Float(address string) {
	if LuaMode() {
		DispatchSilent("hl.dsp.window.float({ action = \"toggle\", window = " +
			luaString("address:"+address) + " })")
		return
	}
	DispatchSilent("togglefloating", "address:"+address)
}

// MoveExact posiciona uma janela em coordenadas absolutas.
func MoveExact(address string, x, y int) {
	if LuaMode() {
		DispatchSilent("hl.dsp.window.move({ window = " + luaString("address:"+address) +
			fmt.Sprintf(", x = %d, y = %d })", x, y))
		return
	}
	DispatchSilent("movewindow", "exact", fmt.Sprintf("%d %d", x, y)+",address:"+address)
}

// ResizeExact redimensiona uma janela.
func ResizeExact(address string, w, h int) {
	if LuaMode() {
		DispatchSilent("hl.dsp.window.resize({ window = " + luaString("address:"+address) +
			fmt.Sprintf(", x = %d, y = %d })", w, h))
		return
	}
	DispatchSilent("resizewindow", "exact", fmt.Sprintf("%d %d", w, h)+",address:"+address)
}

// WaitNewWindow aguarda uma janela nova aparecer e a devolve.
func WaitNewWindow(before map[string]bool, timeout time.Duration) (Client, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		clients, err := Clients()
		if err == nil {
			for _, c := range clients {
				if !before[c.Address] && c.Mapped {
					return c, true
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return Client{}, false
}

// WaitClass aguarda uma janela da classe aparecer, ignorando enderecos que ja
// existiam antes.
func WaitClass(class string, ignore map[string]bool, timeout time.Duration) (Client, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		clients, err := Clients()
		if err == nil {
			for i := len(clients) - 1; i >= 0; i-- {
				c := clients[i]
				if c.Mapped && !ignore[c.Address] && strings.EqualFold(c.Class, class) {
					return c, true
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return Client{}, false
}
