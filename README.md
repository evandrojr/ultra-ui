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
- Workspaces/abas/mapeamento sobrevivem a restart; o restore do herdr retoma as
  sessões de IA e o ultra-ui reconstrói as abas GUI.
- `Super+C` / `Super+V` como copiar/colar universal.

## Requisitos

- herdr 0.9.5+
- Hyprland + `hyprctl`
- `wl-copy` / `wl-paste`
- Go 1.27+ (só para compilar)

## Instalação

```sh
git clone SEU_REPO ultra-ui && cd ultra-ui
./install.sh
```

O script compila para `~/.local/bin/ultra-ui`, cria
`~/.config/ultra-ui/ultra-ui.json`, linka o plugin, adiciona as teclas no
`~/.config/herdr/config.toml` e os atalhos no `foot.ini` — sempre com backup.

## Teclas

Seed do Omarchy (inalterado) + ultra-ui:

| Tecla | Ação |
|---|---|
| `prefix+shift+g` | abrir seletor de apps GUI |
| `prefix+shift+x` | fechar a GUI da aba atual |
| `prefix+/` | busca rápida de workspaces: digitar o nome e ir direto |
| `Super+C` / `Super+V` | copiar / colar |
| `Alt+←` / `Alt+→` | aba anterior / próxima (GUI ou terminal) |
| `Alt+Shift+←` / `→` | mover a aba |

## Estado

- `~/.local/state/ultra-ui/state.json` — mapeamento aba → janela
  (ou `$HERDR_PLUGIN_STATE_DIR/state.json` quando o herdr injeta esse dir).
- `~/.config/ultra-ui/ultra-ui.json` — catálogo de apps e preferências.

## Comandos

```sh
ultra-ui doctor          # verifica dependencias
ultra-ui gui list        # abas GUI ativas
ultra-ui gui pick        # seletor de apps
ultra-ui gui open --app browser
ultra-ui gui close
ultra-ui workspace pick  # busca rapida de workspaces
ultra-ui status
```

## Limitações

Um terminal não renderiza janelas GUI dentro de uma aba: a GUI é uma janela do
Hyprland por cima da área principal. A sidebar continua visível, mas a GUI não
é um painel. Sem sessão gráfica (TTY/SSH) o plugin é no-op para GUI.
