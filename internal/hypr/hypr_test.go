package hypr

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newMock(t *testing.T, clients string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clients.json"), []byte(clients), 0o644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "hyprctl")
	body := "#!/bin/sh\ncat \"" + filepath.Join(dir, "clients.json") + "\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ULTRA_UI_HYPRCTL", script)
	return dir
}

const twoClients = `[
 {"address":"0x1","class":"foot","title":"foot","pid":1,"floating":false,"mapped":true,
  "at":[0,0],"size":[1920,1080],"workspace":{"id":5,"name":"5"},"monitor":0},
 {"address":"0x2","class":"chromium","title":"Novo","pid":2,"floating":false,"mapped":true,
  "at":[0,0],"size":[800,600],"workspace":{"id":5,"name":"5"},"monitor":0}
]`

func TestClientsParse(t *testing.T) {
	newMock(t, twoClients)
	clients, err := Clients()
	if err != nil {
		t.Fatalf("Clients: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("esperava 2 clientes, veio %d", len(clients))
	}
	if clients[1].Class != "chromium" {
		t.Fatalf("classe = %q", clients[1].Class)
	}
}

func TestFindByClass(t *testing.T) {
	newMock(t, twoClients)
	c, ok := FindByClass("chromium")
	if !ok || c.Address != "0x2" {
		t.Fatalf("FindByClass = %+v, %v", c, ok)
	}
}

func TestWaitClassTimeout(t *testing.T) {
	newMock(t, `[]`)
	if _, ok := WaitClass("chromium", 200*time.Millisecond); ok {
		t.Fatal("nao deveria achar janela em estado vazio")
	}
}

func TestWaitNewWindow(t *testing.T) {
	dir := newMock(t, `[
 {"address":"0x1","class":"foot","title":"foot","pid":1,"floating":false,"mapped":true,"at":[0,0],"size":[1920,1080],"workspace":{"id":5,"name":"5"},"monitor":0},
 {"address":"0x2","class":"chromium","title":"Novo","pid":2,"floating":false,"mapped":true,"at":[0,0],"size":[800,600],"workspace":{"id":5,"name":"5"},"monitor":0}
]`)
	before := map[string]bool{"0x1": true, "0x2": true}
	if _, ok := WaitNewWindow(before, 150*time.Millisecond); ok {
		t.Fatal("nenhuma janela nova deveria aparecer")
	}
	if err := os.WriteFile(filepath.Join(dir, "clients.json"), []byte(`[
 {"address":"0x1","class":"foot","title":"foot","pid":1,"floating":false,"mapped":true,"at":[0,0],"size":[1920,1080],"workspace":{"id":5,"name":"5"},"monitor":0},
 {"address":"0x2","class":"chromium","title":"Novo","pid":2,"floating":false,"mapped":true,"at":[0,0],"size":[800,600],"workspace":{"id":5,"name":"5"},"monitor":0},
 {"address":"0x3","class":"thunar","title":"t","pid":3,"floating":false,"mapped":true,"at":[0,0],"size":[1,1],"workspace":{"id":5,"name":"5"},"monitor":0}
]`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, ok := WaitNewWindow(before, 2*time.Second)
	if !ok || c.Address != "0x3" {
		t.Fatalf("WaitNewWindow = %+v, %v", c, ok)
	}
}
