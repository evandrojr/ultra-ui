// Package xpra executa apps graficos sob um servidor Xpra e os anexa em uma
// janela do compositor, permitindo usar apps X11 no Wayland e manter a janela
// ancorada como aba GUI.
package xpra

import (
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"ultra-ui/internal/state"
)

// SocketDir e onde ficam os sockets/display do Xpra.
func SocketDir() string { return filepath.Join(state.Dir(), "xpra") }

// DisplayFor deriva um numero de display estavel a partir do tab id.
func DisplayFor(tabID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(tabID))
	return 200 + int(h.Sum32()%200)
}

// Env devolve o ambiente necessario ao cliente attach. O servidor do herdr nao
// tem DISPLAY/WAYLAND_DISPLAY; usamos o Xwayland do Hyprland.
func Env() []string {
	home, _ := os.UserHomeDir()
	env := os.Environ()
	env = append(env,
		"DISPLAY=:0",
		"XAUTHORITY="+filepath.Join(home, ".Xauthority"),
		"GDK_BACKEND=x11",
	)
	return env
}

func displayName(display int) string { return fmt.Sprintf(":%d", display) }

// Available indica se o xpra esta instalado.
func Available() bool {
	_, err := exec.LookPath("xpra")
	return err == nil
}

// Start sobe um servidor Xpra que roda o comando e morre junto com ele.
func Start(display int, command []string) error {
	if err := os.MkdirAll(SocketDir(), 0o755); err != nil {
		return err
	}
	args := []string{
		"start", displayName(display),
		"--start-child=" + strings.Join(command, " "),
		"--exit-with-children",
		"--daemon=yes",
		"--socket-dir=" + SocketDir(),
	}
	cmd := exec.Command("xpra", args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xpra start: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Attach abre a janela do servidor em background, destacado do plugin.
func Attach(display int) error {
	logPath := filepath.Join(SocketDir(), fmt.Sprintf("attach-%d.log", display))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd := exec.Command("xpra", "attach", displayName(display), "--socket-dir="+SocketDir())
	cmd.Env = Env()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("xpra attach: %w", err)
	}
	return nil
}

// Stop encerra o servidor Xpra.
func Stop(display int) {
	cmd := exec.Command("xpra", "stop", displayName(display), "--socket-dir="+SocketDir())
	cmd.Env = os.Environ()
	_ = cmd.Run()
}
