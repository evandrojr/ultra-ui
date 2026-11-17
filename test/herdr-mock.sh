#!/usr/bin/env bash
# Mock do herdr para testes do ultra-ui fora de uma sessao real.
# Uso: ULTRA_UI_HERDR=./test/herdr-mock.sh ultra-ui ...
set -u
STATE="$(cd "$(dirname "$0")" && pwd)"
case "$*" in
  "status server --json") echo '{"status":"running","running":true}' ;;
  "workspace list") cat "$STATE/workspaces.json" ;;
  "tab list"|"tab list --workspace "*) cat "$STATE/tabs.json" ;;
  "pane current") cat "$STATE/current.json" ;;
  "tab create"*) echo '{"result":{"type":"tab_created","tab":{"tab_id":"w19:t2"},"pane":{"pane_id":"w19:p2"}}}' ;;
  "pane run"*) echo '{"result":{"type":"pane_command","ok":true}}' ;;
  "pane send-text"*) echo '{"result":{"type":"pane_command","ok":true}}' ;;
  "pane send-keys"*) echo '{"result":{"type":"pane_command","ok":true}}' ;;
  "tab close"*) echo '{"result":{"type":"tab_closed","ok":true}}' ;;
  "tab focus"*) echo '{"result":{"type":"tab_focused","ok":true}}' ;;
  "workspace focus"*) echo '{"result":{"type":"workspace_focused","ok":true}}' ;;
  *) echo '{}' ;;
esac
