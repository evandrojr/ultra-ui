#!/usr/bin/env bash
# Instala o ultra-ui: binario, config, plugin e teclas.
# Idempotente; faz backup de cada arquivo que altera.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="$HOME/.local/bin/ultra-ui"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/ultra-ui"
CONFIG="$CONFIG_DIR/ultra-ui.json"
HERDR_CONFIG="${HERDR_CONFIG_PATH:-${XDG_CONFIG_HOME:-$HOME/.config}/herdr/config.toml}"
FOOT_CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/foot/foot.ini"

log() { printf 'ultra-ui: %s\n' "$*"; }
backup() {
  local f="$1"
  if [[ -f $f ]]; then
    cp -a "$f" "$f.bak.$(date +%s)"
    log "backup: $f"
  fi
}

# 1. Binario
mkdir -p "$HOME/.local/bin"
log "compilando $BIN"
( cd "$REPO" && go build -o "$BIN" ./cmd/ultra-ui )

# 2. Config
mkdir -p "$CONFIG_DIR"
if [[ ! -f $CONFIG ]]; then
  cp "$REPO/config/ultra-ui.example.json" "$CONFIG"
  log "config: $CONFIG"
else
  log "config ja existe: $CONFIG"
fi

# 3. Plugin
if herdr plugin list 2>/dev/null | grep -q 'local-ultra-ui'; then
  log "plugin ja linkado"
else
  herdr plugin link "$REPO" >/dev/null
  log "plugin linkado"
fi

# 4. Teclas do herdr
if [[ -f $HERDR_CONFIG ]]; then
  if grep -q 'ultra-ui gui pick' "$HERDR_CONFIG"; then
    log "teclas do herdr ja presentes"
  else
    backup "$HERDR_CONFIG"
    cat >> "$HERDR_CONFIG" <<'EOF'

# --- ultra-ui ---
[[keys.command]]
key = "prefix+shift+g"
type = "shell"
command = "ultra-ui gui pick"

[[keys.command]]
key = "prefix+shift+x"
type = "shell"
command = "ultra-ui gui close"

[[keys.command]]
key = "super+c"
type = "shell"
command = "ultra-ui clipboard copy"

[[keys.command]]
key = "super+v"
type = "shell"
command = "ultra-ui clipboard paste"
EOF
    log "teclas adicionadas ao herdr"
  fi
  herdr config check || true
  herdr server reload-config >/dev/null 2>&1 || true
fi

# 5. Atalhos por app (prefix+shift+<tecla>)
if [[ -f $HERDR_CONFIG ]]; then
  if grep -q '# ultra-ui apps' "$HERDR_CONFIG"; then
    log "atalhos por app ja presentes"
  else
    backup "$HERDR_CONFIG"
    {
      echo
      echo '# ultra-ui apps'
      "$BIN" install-keys
    } >> "$HERDR_CONFIG"
    log "atalhos por app adicionados"
  fi
  herdr config check || true
  herdr server reload-config >/dev/null 2>&1 || true
fi

# 4b. Busca rapida de workspaces (prefix+/)
if [[ -f $HERDR_CONFIG ]]; then
  if grep -q 'ultra-ui workspace pick' "$HERDR_CONFIG"; then
    log "tecla de busca de workspaces ja presente"
  else
    backup "$HERDR_CONFIG"
    cat >> "$HERDR_CONFIG" <<'EOF'

# ultra-ui: busca rapida de workspaces
[[keys.command]]
key = "prefix+/"
type = "shell"
command = "ultra-ui workspace pick"
EOF
    log "tecla de busca de workspaces adicionada"
  fi
  herdr config check || true
  herdr server reload-config >/dev/null 2>&1 || true
fi

# 6. foot: Super+C / Super+V (adiciona as teclas nas linhas existentes)
if [[ -f $FOOT_CONFIG ]]; then
  if grep -q 'ultra-ui-super' "$FOOT_CONFIG"; then
    log "foot: atalhos ja presentes"
  else
    backup "$FOOT_CONFIG"
    sed -i \
      -e 's/^\(clipboard-copy=.*\)$/  # ultra-ui-super\n\1 Super+c/' \
      -e 's/^\(clipboard-paste=.*\)$/  # ultra-ui-super\n\1 Super+v/' \
      "$FOOT_CONFIG"
    log "foot: atalhos adicionados"
  fi
fi

# 7. Hyprland: Super+setas alternam abas (substitui o foco de janela)
HYPR_BINDINGS="${XDG_CONFIG_HOME:-$HOME/.config}/hypr/bindings.lua"
if [[ -f $HYPR_BINDINGS ]]; then
  if grep -q 'UltraUI: aba anterior' "$HYPR_BINDINGS"; then
    log "hypr: binds ja presentes"
  else
    backup "$HYPR_BINDINGS"
    cat >> "$HYPR_BINDINGS" <<'EOF'

-- ultra-ui: Super+setas alternam abas do herdr (substitui o foco de janela)
hl.unbind("SUPER + LEFT")
hl.unbind("SUPER + RIGHT")
o.bind("SUPER + LEFT", "UltraUI: aba anterior", "ultra-ui tab prev")
o.bind("SUPER + RIGHT", "UltraUI: aba proxima", "ultra-ui tab next")
hl.unbind("SUPER + UP")
hl.unbind("SUPER + DOWN")
o.bind("SUPER + UP", "UltraUI: workspace anterior", "ultra-ui workspace prev")
o.bind("SUPER + DOWN", "UltraUI: workspace proximo", "ultra-ui workspace next")
EOF
    log "hypr: binds adicionados"
  fi
  command -v hyprctl >/dev/null 2>&1 && hyprctl reload >/dev/null 2>&1 || true
fi

log "pronto. rode 'ultra-ui doctor' para conferir."
