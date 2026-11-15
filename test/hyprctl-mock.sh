#!/usr/bin/env bash
# Mock de hyprctl para testar o ultra-ui fora de uma sessao grafica.
# Uso: MOCK_STATE=<dir> ULTRA_UI_HYPRCTL=./test/hyprctl-mock.sh ultra-ui gui open --app browser
set -u
STATE="${MOCK_STATE:?defina MOCK_STATE}"
mkdir -p "$STATE"

clients_file="$STATE/clients.json"
[[ -f $clients_file ]] || echo '[]' > "$clients_file"

key="${1:-}"
sub="${2:-}"
case "$key $sub" in
  "clients -j") cat "$clients_file" ;;
  "activeworkspace -j") cat "$STATE/activeworkspace.json" 2>/dev/null || echo '{"id":5,"name":"5"}' ;;
  "activeworkspace "*|"activeworkspace") echo "5" ;;
  "activewindow -j") cat "$STATE/activewindow.json" 2>/dev/null || echo '{"address":"0x1","class":"foot","title":"foot"}' ;;
  "workspaces -j") cat "$STATE/workspaces.json" 2>/dev/null || echo '[{"id":5,"name":"5"}]' ;;
  "monitors -j") cat "$STATE/monitors.json" 2>/dev/null || echo '[{"id":0,"name":"DP-1","width":1920,"height":1080,"focused":true}]' ;;
  "dispatch "*)
    echo "$*" >> "$STATE/dispatch.log"
    args="$*"
    if [[ $args == *'no_op'* ]]; then
      echo "ok"
      exit 0
    fi
    if [[ $args == *"exec"* ]]; then
      if [[ $args == *'exec_cmd('* ]]; then
        app=$(printf '%s' "$args" | sed -n 's/.*exec_cmd("\([^"]*\)").*/\1/p')
      else
        app="${args##*exec }"
      fi
      case "$app" in
        chromium*) cls="chromium"; addr="0xCHROMIUM" ;;
        thunar*)   cls="thunar";   addr="0xTHUNAR" ;;
        zed*)      cls="dev.zed.Zed"; addr="0xZED" ;;
        *)         cls="$app";      addr="0x${app^^}" ;;
      esac
      python3 - "$clients_file" "$cls" "$addr" <<'PY'
import json,sys
path,cls,addr=sys.argv[1],sys.argv[2],sys.argv[3]
data=json.load(open(path))
data=[c for c in data if c["address"]!=addr]
data.append({"address":addr,"class":cls,"title":cls,"pid":4242,
             "floating":False,"mapped":True,"at":[0,0],"size":[1000,700],
             "workspace":{"id":5,"name":"5"},"monitor":0})
json.dump(data,open(path,"w"))
PY
    elif [[ $args == *"closewindow"* ]]; then
      addr="${args##*address:}"
      python3 - "$clients_file" "$addr" <<'PY'
import json,sys
path,addr=sys.argv[1],sys.argv[2]
data=json.load(open(path))
data=[c for c in data if c["address"]!=addr]
json.dump(data,open(path,"w"))
PY
    fi
    ;;
  *) echo "" ;;
esac
