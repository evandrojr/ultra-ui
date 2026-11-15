# ultra-ui — Handout

## Resumo

Plugin do **herdr** que adiciona abas GUI do Hyprland, mantendo workspaces,
abas, painéis, sidebar, teclas, persistência e restore de sessões de IA.
Escrito em Go (stdlib). Repo: `github.com/evandrojr/ultra-ui` (privado),
diretório `/home/j/lab/ultra-ui`.

A aba GUI é uma **aba real do herdr**: Alt+←/→, Prefix+N/P e Prefix+1..9
continuam funcionando. A janela gráfica é uma janela do Hyprland ancorada na
área principal, com a sidebar visível.

## Decisões

- Ficou no herdr (não WezTerm): o híbrido é preocupação do compositor; o
  herdr já tem sidebar, persistência e restore de agentes.
- Go, sem dependências externas.
- GUI = aba do herdr + janela do Hyprland (address persistido em
  `state.json`).
- Ao focar aba de terminal, a GUI vai para `special:ultra-ui`.

## Estrutura

```
ultra-ui/
  SPEC.md                 spec completa
  README.md               uso
  HANDOUT.md              este resumo
  herdr-plugin.toml       manifest (startup + 4 hooks + 2 ações)
  cmd/ultra-ui/main.go
  internal/{config,state,hypr,herdr,apps,clipboard,gui,cli}/
  config/ultra-ui.example.json
  install.sh              instalação idempotente (com backups)
  test/{hyprctl-mock.sh,herdr-mock.sh}
```

## Instalação (já aplicada)

```sh
cd /home/j/lab/ultra-ui && ./install.sh
```

Feito: binário em `~/.local/bin/ultra-ui`, config em
`~/.config/ultra-ui/ultra-ui.json`, plugin linkado (`local-ultra-ui`),
teclas em `~/.config/herdr/config.toml`, Super+C/V no `foot.ini`.

## Teclas

Seed Omarchy (inalterado) +:

| Tecla | Ação |
|---|---|
| `prefix+shift+g` | seletor de apps GUI |
| `prefix+shift+x` | fechar a GUI da aba atual |
| `Super+C` / `Super+V` | copiar / colar |
| `Alt+←` / `Alt+→` | aba anterior / próxima |
| `Alt+Shift+←` / `→` | mover a aba |

## Estado

- `~/.local/state/ultra-ui/state.json` (ou `$HERDR_PLUGIN_STATE_DIR/state.json`)
- `~/.config/ultra-ui/ultra-ui.json` (catálogo de apps + `sidebar_width_px`)

## Verificado

- `go vet` + `go test ./...` verdes; build ok; `herdr config check` ok.
- Teste headless com mocks: cria aba real no herdr, ancora, persiste, fecha.
- Acesso ao Hyprland via SSH funciona com
  `HYPRLAND_INSTANCE_SIGNATURE=$(basename "$(ls -d /run/user/1000/hypr/*/ | head -1)")`.

## PENDÊNCIA CRÍTICA — Hyprland 0.56.2 (config Lua)

O Hyprland 0.56.2 com config Lua mudou o `hyprctl dispatch`: a forma antiga
(`dispatch exec <cmd>`, `dispatch focuswindow address:...`) **falha**. O plugin
ainda emite a forma legada, portanto abrir/focar/fechar GUI **não funciona**
nessa sessão até a correção.

Forma nova (validada, `exec_cmd` abriu o Dolphin):

```
hyprctl dispatch 'hl.dsp.exec_cmd("<cmd>")'
hyprctl dispatch 'hl.dsp.focus({ window = "address:0x..." })'
hyprctl dispatch 'hl.dsp.window.move({ workspace = "WS", window = "address:0x...", follow = false })'
hyprctl dispatch 'hl.dsp.window.close({ window = "address:0x..." })'
hyprctl dispatch 'hl.dsp.window.float({ action = "toggle", window = "address:0x..." })'
hyprctl dispatch 'hl.dsp.window.move({ window = "address:0x...", x = X, y = Y })'
hyprctl dispatch 'hl.dsp.window.resize({ window = "address:0x...", x = W, y = H })'
```

Correção prevista: em `internal/hypr`, detectar o modo (probe
`hl.dsp.exec_cmd("true")` → `ok`) e traduzir cada dispatch para a forma Lua;
manter a forma legada como fallback. Testar âncora e alternância ao vivo.

## Próximos passos

1. Camada de compatibilidade do dispatch (acima).
2. Teste ao vivo: abrir Dolphin/btop como aba, Alt+←/→, fechar/reabrir herdr.
3. Ajustar `sidebar_width_px` se a âncora não alinhar com a sidebar.
4. Restore de GUIs após restart do Hyprland.
