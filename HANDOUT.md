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
| `prefix+/` | busca rápida de workspaces: digita o nome e vai direto |
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

## Status do teste ao vivo (Hyprland 0.56.2)

Verificado de verdade, via SSH com o Hyprland rodando:

- `hl.dsp.exec_cmd` abriu o Dolphin; a aba GUI foi criada no herdr e o estado
  persistiu.
- Focar a aba de terminal manda a GUI para `special:ultra-ui`; focar a aba GUI
  a traz de volta e a foca.
- `gui close` fecha a janela e limpa o estado.

Limitação conhecida: o app pode impor tamanho mínimo (o Dolphin abre com
803px de largura, um pouco acima dos 683px da área principal). A posição x
fica exatamente após a sidebar; a largura pode exceder a área principal.
Ajuste `sidebar_width_px` se precisar.

## PENDÊNCIA CRÍTICA — Hyprland 0.56.2 (config Lua) — RESOLVIDA

O plugin detecta o modo via `hyprctl dispatch 'hl.dsp.no_op()'` e usa a forma
Lua (`hl.dsp.exec_cmd`, `hl.dsp.focus`, `hl.dsp.window.*`) quando disponível,
com fallback para a forma legada. O server do herdr não tem
`HYPRLAND_INSTANCE_SIGNATURE`; o plugin descobre a instância em
`$XDG_RUNTIME_DIR/hypr` e injeta no subprocess.

## Super+setas = herdr

Vinculado **no Hyprland** (`~/.config/hypr/bindings.lua`), gerenciado pelo
`install.sh`:

| Tecla | Ação |
|---|---|
| `Super+←` / `Super+→` | aba anterior / próxima (`ultra-ui tab prev|next`) |
| `Super+↑` / `Super+↓` | workspace anterior / próximo (`ultra-ui workspace prev|next`) |

O compositor engole a tecla, então a janela gráfica focada não interpreta o
atalho. Isso substitui os bindings de foco de janela do Omarchy
(`hl.unbind("SUPER + ...")`).

## Modo Xpra (apps X11)

Apps com `"mode": "xpra"` rodam sob um servidor Xpra e são anexados numa
janela do compositor, ancorada como aba GUI. Requer `xpra` e as fontes X
(`xorg-fonts-75dpi xorg-fonts-alias-75dpi`). O Xpra **não** desenha dentro da
aba (é janela separada, como o modo nativo); o ganho é rodar apps X11 no
Wayland com a mesma âncora e alternância.

```json
"apps": { "xterm": { "label": "XTerm", "command": ["xterm"], "window_class": "XTerm", "mode": "xpra" } }
```

## Âncora

A âncora é aplicada uma vez por janela. Se você mover/redimensionar, a
geometria é preservada ao sair e voltar. Para forçar a âncora padrão:
`ultra-ui gui anchor`.

## Próximos passos

1. Camada de compatibilidade do dispatch (acima).
2. Teste ao vivo: abrir Dolphin/btop como aba, Alt+←/→, fechar/reabrir herdr.
3. Ajustar `sidebar_width_px` se a âncora não alinhar com a sidebar.
4. Restore de GUIs após restart do Hyprland.
