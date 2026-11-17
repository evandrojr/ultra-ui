# ultra-ui — Especificação

## 1. Objetivo

Estender o **herdr** com abas que comportam aplicações gráficas do Hyprland,
sem abrir mão de nada que o herdr já faz: workspaces, abas, painéis, barra
lateral de workspaces, teclas, persistência e restore de sessões de IA.

O ganho é um: uma aba pode ser um terminal **ou** uma janela gráfica, e a
alternância continua sendo a mesma de sempre (Alt+←/→, Prefix+N/P, Prefix+1..9).

## 2. Decisões de arquitetura

| Decisão | Escolha | Por quê |
|---|---|---|
| Base | herdr (plugin) + Hyprland (`hyprctl`) | Um emulador de terminal não renderiza janelas GUI dentro de uma aba; o híbrido é uma preocupação do compositor. O herdr já tem sidebar, persistência e restore de agentes. |
| Linguagem | Go (stdlib apenas) | Orquestração de processos (herdr CLI, hyprctl, wl-clipboard). Binário estático, compilação rápida, sem runtime. Rust só pagaria como plugin nativo do herdr-core; o plugin v1 é argv puro. |
| Aba GUI | Uma *tab real* do herdr + uma janela do Hyprland | Alt+←/→, Prefix+N/P e Prefix+1..9 continuam funcionando sem tratamento especial. A sidebar do herdr permanece visível. |
| Posicionamento | Janela GUI flutuante, ancorada à direita da sidebar | A GUI ocupa a área principal dos painéis sem cobrir a barra lateral de workspaces. |
| Persistência | `state.json` no `HERDR_PLUGIN_STATE_DIR` | Estado extra (mapeamento aba→janela) fora do plugin root; o herdr já persiste workspaces/abas/panes/agentes. |

## 3. Requisitos

- **RF-01 — Aba GUI first-class.** Criar, focar, renomear, mover e fechar uma
  aba GUI com as mesmas teclas e cliques de uma aba de terminal. Alt+←/→ e
  Alt+Shift+←/→ funcionam sem exceção.
- **RF-02 — Sidebar preservada.** Ao focar uma aba GUI, a janela gráfica é
  movida para o workspace do Hyprland ativo, recebe foco e é ancorada na área
  principal (sidebar visível à esquerda).
- **RF-03 — Alternância.** Ao focar uma aba de terminal, a GUI do workspace
  vai para o scratchpad (`special:ultra-ui`) e o terminal volta a aparecer.
- **RF-04 — Persistência.** Workspaces, abas e o mapeamento aba→janela
  sobrevivem a restart do herdr e do Hyprland. Autosave em cada mudança.
- **RF-05 — Restore de IA.** O restore nativo do herdr
  (`resume_agents_on_restore` + integrações claude/codex/opencode/kiro/hermes)
  continua responsável por retomar as sessões de IA; o plugin só reconstrói as
  abas GUI depois dele.
- **RF-06 — Teclas.** Manter o seed do Omarchy; adicionar atalhos de GUI.
- **RF-07 — Super+C / Super+V.** Copiar e colar universais no terminal/Omarchy.
- **RF-08 — Degradação.** Sem Hyprland (TTY, SSH), o plugin não falha: abas GUI
  não são criadas e os comandos avisam.

## 4. Modelo de dados

`$HERDR_PLUGIN_STATE_DIR/ultra-ui/state.json`:

```json
{
  "version": 1,
  "workspace_prefix": "ultra:",
  "apps": {
    "browser": { "label": "Chromium", "command": ["chromium"], "window_class": "chromium" }
  },
  "gui_tabs": {
    "w5:t2": {
      "workspace_id": "w5",
      "app": "browser",
      "label": "Chromium",
      "hypr_workspace": "ultra:w5",
      "window_address": "0x5612abcd",
      "window_class": "chromium",
      "anchored": true
    }
  }
}
```

Chaves: `gui_tabs` é indexado pelo ID completo da tab do herdr (`workspace:tab`).
O mapeamento de workspace é implícito: a GUI vive no workspace do Hyprland ativo
quando é focada; o campo `hypr_workspace` registra onde foi criada.

## 5. Fluxos

### 5.1 Foco de aba GUI (`tab.focused`)

1. Resolve o ID da tab (`HERDR_TAB_ID` ou payload do evento).
2. Se a tab está em `gui_tabs`:
   - `hyprctl dispatch movetoworkspace <ws-ativo>,address:<addr>`
   - `hyprctl dispatch focuswindow address:<addr>`
   - se `anchored != true`: `togglefloating`, `movewindow`, `resizewindow`
     (x = terminal.x + `sidebar_width_px`, y = terminal.y,
     w = terminal.w − sidebar, h = terminal.h) e marca `anchored`.
3. Se a tab **não** é GUI: se existe GUI ativa no workspace, movê-la para
   `special:ultra-ui` (`movetoworkspacesilent`), e o terminal volta ao foco.

### 5.2 Abertura (`ultra-ui gui open`)

1. Lê os clients do Hyprland antes.
2. Cria a tab no herdr (`herdr tab create --label <label> --focus`).
3. Executa o app (`hyprctl dispatch exec` / spawn direto com `uwsm-app`).
4. Aguarda até 5 s por uma nova janela; associa por endereço.
5. Roda um sentinela no pane (`ultra-ui gui sentinel <tab-id>`) para a aba ter
   um placeholder claro quando a GUI não está visível.
6. Persiste o estado.

### 5.3 Fechamento (`ultra-ui gui close` ou fechar a aba)

- Fechar a aba GUI fecha a janela do Hyprland (`hyprctl dispatch closewindow`).
- O sentinela morre junto; o estado é limpo.
- Alternativa: `ultra-ui gui detach` mantém a janela e devolve a aba ao shell.

### 5.4 Restore (`[[startup]]`)

1. Herdr restaura workspaces/abas/panes e sessões de IA.
2. O plugin lê `state.json` e, para cada `gui_tabs`:
   - Janela ainda existe → garante a tab (cria se o herdr a perdeu) e associa.
   - Janela não existe → reabre o app e cria a tab.
3. Abas que o herdr restaurou como shell (porque o sentinela morreu) são
   convertidas de volta para GUI pelo plugin.

## 6. Teclas

Seed do Omarchy (inalterado, já em `~/.config/herdr/config.toml`):

```
prefix            ctrl+space
prefix+h / alt+enter        split horizontal
prefix+v / alt+shift+enter  split vertical
prefix+x / alt+esc          fecha painel
prefix+z                    zoom
prefix+[                    copy mode
ctrl+alt+setas              foco de painel
ctrl+alt+shift+setas        redimensiona
prefix+c / r / k            nova / renomear / fechar aba
prefix+1..9 / alt+1..9      troca de aba
prefix+p / n / alt+← / →    aba anterior / próxima
alt+shift+← / →             move aba
prefix+shift+c / r / k      novo / renomear / fechar workspace
prefix+shift+p / n / alt+↑/↓  workspace anterior / próximo
```

Adicionais do ultra-ui:

```
prefix+/          ultra-ui workspace pick  busca rapida de workspaces
prefix+shift+g   ultra-ui gui pick    abre o seletor de apps GUI
prefix+shift+x   ultra-ui gui close   fecha a GUI da aba atual
super+c          copiar (clipboard universal)
super+v          colar  (clipboard universal)
```

As teclas de GUI entram em `~/.config/herdr/config.toml` como
`[[keys.command]]` (`type = "shell"`); o manifest do plugin não pode declarar
keybindings.

## 7. Super+C / Super+V

Duas camadas, para cobrir terminal e TUI:

1. **foot** (terminal do Omarchy): bindings `Super+c` → `copy`, `Super+v` →
   `paste` no `foot.ini`.
2. **herdr** (opcional): `[[keys.command]]` com `ultra-ui clipboard copy|paste`,
   que usa `wl-copy`/`wl-paste` e `herdr pane send-text`.
3. `copy_on_select = true` permanece ativo no herdr.

## 8. CLI

```
ultra-ui startup                 hook [[startup]]
ultra-ui event <nome>            hook [[events]] (lê HERDR_PLUGIN_CONTEXT_JSON)
ultra-ui gui open [--app id] [--label txt] [--cwd dir]
ultra-ui gui close               fecha a GUI da aba atual
ultra-ui gui detach              desacopla a GUI (aba vira shell)
ultra-ui gui pick                seletor de apps
ultra-ui gui list                lista abas GUI
ultra-ui status                  mapeamento atual
ultra-ui workspace pick          busca rapida de workspaces (prefix+/)
ultra-ui doctor                  verifica herdr/hyprctl/wl-clipboard/foot
ultra-ui clipboard copy|paste
```

## 9. Estrutura

```
ultra-ui/
  SPEC.md
  README.md
  herdr-plugin.toml
  go.mod
  cmd/ultra-ui/main.go
  internal/{config,state,hypr,herdr,apps,clipboard,cli}/
  config/ultra-ui.example.json
  install.sh
```

## 10. Instalação

`install.sh` compila para `~/.local/bin/ultra-ui`, copia a config de exemplo
para `~/.config/ultra-ui/`, liga o plugin (`herdr plugin link`), aplica as
chaves no `config.toml` do herdr (com backup) e recarrega
(`herdr server reload-config`).

## 11. Verificação

- `go build ./...` e `go test ./...` (estado, parsing de `hyprctl`, âncora).
- Testes com mocks: `ULTRA_UI_HYPRCTL` e `ULTRA_UI_HERDR` apontam para scripts
  de teste; cobre foco, fechamento e restore.
- `herdr config check` após aplicar as teclas.
- Em sessão gráfica: abrir uma GUI, Alt+←/→ para terminal e de volta, fechar e
  reabrir o herdr, conferir o restore.

## 12. Riscos e limites

- **Sem embed real.** A GUI não é desenhada dentro da aba; é uma janela do
  Hyprland por cima da área principal. A sidebar permanece visível, mas a GUI
  não é um pane.
- **Posicionamento no Wayland.** `movewindow`/`resizewindow` via hyprctl
  funcionam no Hyprland, mas o usuário pode reposicionar depois; o plugin só
  ancora uma vez por janela.
- **Apps que não abrem janela própria** (daemon, tray) não são detectados;
  o plugin avisa e não cria a aba.
- **SSH/headless:** sem Hyprland, o plugin é no-op para GUI.

## 13. Faseamento

1. **MVP** — abas GUI, foco/alternância, persistência, teclas, Super+C/V.
2. **Restore de GUIs** — reabrir janelas após restart do Hyprland.
3. **Seletor** integrado e ancoragem por app.
4. **Polimento** — detach, preferências por app, metadados na sidebar.
