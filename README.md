# ultra-ui

Abas GUI do Hyprland dentro do **herdr**. Cada aba continua sendo uma aba normal
do herdr — Alt+←/→, Prefix+N/P e Prefix+1..9 funcionam — mas pode comportar uma
janela gráfica (Chromium, Zed, thunar, …) ancorada na área principal, com a
barra lateral de workspaces preservada.

## O que faz

- Aba GUI é tab real do herdr: criar, focar, renomear, mover e fechar com as
  mesmas teclas e cliques.
- Ao focar a aba GUI, a janela do Hyprland vai para o workspace ativo, recebe
  foco e é ancorada à direita da sidebar (largura configurável).
- Ao focar uma aba de terminal, a GUI vai para o scratchpad e o terminal volta.
- Apps X11 podem rodar sob Xpra e ser ancorados da mesma forma (modo `xpra`).
- Workspaces/abas/mapeamento sobrevivem a restart; o restore do herdr retoma as
  sessões de IA e o ultra-ui reconstrói as abas GUI.
- `Super+C` / `Super+V` como copiar/colar universal; `Super+setas` alterna
  abas e workspaces direto do compositor.
- **Go to space com `/`**: `prefix+/` abre um seletor, digite o nome do
  workspace e vá direto para ele.

## Requisitos

- herdr 0.9.5+
- Hyprland + `hyprctl`
- `wl-copy` / `wl-paste`
- Go 1.27+ (só para compilar)
- `omarchy-menu-select` (opcional; fallback para seletor em terminal)
- `xpra` (opcional; só para apps com `"mode": "xpra"`)

## Instalação

```sh
git clone SEU_REPO ultra-ui && cd ultra-ui
./install.sh
```

O script compila para `~/.local/bin/ultra-ui`, cria
`~/.config/ultra-ui/ultra-ui.json`, linka o plugin e instala as teclas: no
`~/.config/herdr/config.toml` (incluindo `prefix+/` e os atalhos por app), no
`foot.ini` (`Super+C`/`Super+V`) e no `bindings.lua` do Hyprland
(`Super+setas`) — sempre com backup e recarregando a config do herdr e do
Hyprland.

## Teclas

Seed do Omarchy (inalterado) + ultra-ui, no herdr (`prefix`):

| Tecla | Ação |
|---|---|
| `prefix+shift+g` | abrir seletor de apps GUI |
| `prefix+shift+x` | fechar a GUI da aba atual |
| `prefix+/` | go to space: digitar o nome do workspace e ir direto |
| `Super+C` / `Super+V` | copiar / colar |
| `Alt+←` / `Alt+→` | aba anterior / próxima (GUI ou terminal) |
| `Alt+Shift+←` / `→` | mover a aba |

Atalhos por app (`prefix+shift+<tecla>`), definidos pelo campo `key` de cada
app no JSON e gerados no config do herdr pelo `install-keys`. Padrões:

| Tecla | App |
|---|---|
| `prefix+shift+f` | Dolphin |
| `prefix+shift+v` | VS Code |
| `prefix+shift+m` | Omawrite |
| `prefix+shift+b` | btop |
| `prefix+shift+i` | imv |
| `prefix+shift+t` | XTerm (xpra) |
| `prefix+shift+l` | lazygit |

No Hyprland (`bindings.lua`, gerenciado pelo `install.sh`), `Super+setas`
alterna abas e workspaces. O compositor engole a tecla, então a janela gráfica
focada não interpreta o atalho:

| Tecla | Ação |
|---|---|
| `Super+←` / `Super+→` | aba anterior / próxima |
| `Super+↑` / `Super+↓` | workspace anterior / próximo |

## Configuração

`~/.config/ultra-ui/ultra-ui.json`:

```json
{
  "workspace_prefix": "ultra:",
  "scratchpad": "special:ultra-ui",
  "sidebar_width_px": 240,
  "anchor": true,
  "apps": {
    "browser": { "label": "Chromium", "command": ["chromium"], "window_class": "chromium", "key": "c" }
  }
}
```

- `sidebar_width_px` — largura da âncora; ajuste se a janela exceder a área.
- `anchor` — reaplica a âncora ao reabrir uma GUI.
- App: `label`, `command`, `window_class`, `mode` (`native` padrão, `xpra`
  para rodar sob Xpra) e `key` (atalho `prefix+shift+<key>`).

### Modo Xpra

Apps X11 com `"mode": "xpra"` rodam sob um servidor Xpra e são anexados numa
janela do compositor, ancorada como aba GUI. Requer `xpra` e as fontes X
(`xorg-fonts-75dpi xorg-fonts-alias-75dpi`). O Xpra não desenha dentro da aba
(é janela separada, como o modo nativo); o ganho é rodar apps X11 no Wayland
com a mesma âncora e alternância:

```json
"xterm": { "label": "XTerm", "command": ["xterm"], "window_class": "XTerm", "mode": "xpra" }
```

## Estado

- `~/.local/state/ultra-ui/state.json` — mapeamento aba → janela
  (ou `$HERDR_PLUGIN_STATE_DIR/state.json` quando o herdr injeta esse dir).
- `~/.config/ultra-ui/ultra-ui.json` — catálogo de apps e preferências.

## Comandos

```sh
ultra-ui doctor                  # verifica dependencias
ultra-ui status                  # mostra o mapeamento atual
ultra-ui gui list                # abas GUI ativas
ultra-ui gui pick                # seletor de apps
ultra-ui gui open --app browser  # abre app na aba atual
ultra-ui gui close               # fecha a GUI da aba atual
ultra-ui gui anchor              # reaplica a ancora na janela ativa
ultra-ui gui detach              # desacopla a GUI (aba volta a ser terminal)
ultra-ui tab prev|next           # aba anterior / proxima
ultra-ui workspace pick          # go to space (prefix+/)
ultra-ui workspace prev|next     # workspace anterior / proximo (Super+setas)
ultra-ui clipboard copy|paste    # copia/cola do terminal (Super+C/V)
ultra-ui install-keys            # gera os atalhos por app (prefix+shift+<tecla>)
```

## Limitações

Um terminal não renderiza janelas GUI dentro de uma aba: a GUI é uma janela do
Hyprland por cima da área principal. A sidebar continua visível, mas a GUI não
é um painel. Sem sessão gráfica (TTY/SSH) o plugin é no-op para GUI. A âncora é
aplicada uma vez por janela; se mover/redimensionar, a geometria é preservada
ao sair e voltar (`ultra-ui gui anchor` força a âncora padrão).
