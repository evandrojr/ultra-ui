package workspaces

import (
	"testing"

	"ultra-ui/internal/herdr"
)

func sampleWorkspaces() []herdr.WorkspaceSummary {
	return []herdr.WorkspaceSummary{
		{WorkspaceID: "w5", Number: 1, Label: "torrentflix"},
		{WorkspaceID: "w17", Number: 2, Label: "centro-de-controle"},
		{WorkspaceID: "wF", Number: 6, Label: "go-proxy-serprollm"},
		{WorkspaceID: "w1D", Number: 30, Label: "go-proxy-serprollm"},
		{WorkspaceID: "w19", Number: 25, Label: "ultra-ui", Focused: true},
	}
}

func TestBuildOptionsCarregaSubtextoEstavel(t *testing.T) {
	opts := buildOptions(sampleWorkspaces())
	if len(opts) != 5 {
		t.Fatalf("opcoes = %d, esperado 5", len(opts))
	}
	want := "\tultra-ui\t#25"
	if opts[4] != want {
		t.Fatalf("opcao = %q, esperado %q", opts[4], want)
	}
}

func TestResolvePeloNumeroDesambiguaRolulosRepetidos(t *testing.T) {
	ws := sampleWorkspaces()
	if got := resolve("go-proxy-serprollm\t#30", ws); got != "w1D" {
		t.Fatalf("resolve #30 = %q, esperado w1D", got)
	}
	if got := resolve("go-proxy-serprollm\t#6", ws); got != "wF" {
		t.Fatalf("resolve #6 = %q, esperado wF", got)
	}
}

func TestResolveComRotuloPuroEFallback(t *testing.T) {
	ws := sampleWorkspaces()
	if got := resolve("ultra-ui", ws); got != "w19" {
		t.Fatalf("resolve rotulo = %q, esperado w19", got)
	}
	if got := resolve("  torrentflix  ", ws); got != "w5" {
		t.Fatalf("resolve com espacos = %q, esperado w5", got)
	}
}

func TestResolveEscolhaInvalida(t *testing.T) {
	if got := resolve("nao-existe", sampleWorkspaces()); got != "" {
		t.Fatalf("resolve invalido = %q, esperado vazio", got)
	}
}
