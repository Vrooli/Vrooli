#!/usr/bin/env bash
# Regenerate (or diff) the committed expected-graph.json files for the
# Phase 6 integration fixtures by spawning the real Node sidecar and
# extracting each project.
#
# Usage:
#   ./regenerate.sh             # prints unified diff between live and committed
#   UPDATE_FIXTURES=1 ./regenerate.sh   # rewrites the committed files
#
# Determinism note: we capture only .graph (not .warnings). Sidecar
# diagnostics include nondeterministic type-check noise; the .graph is
# stably sorted by id / (from,to).
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
SCENARIO_ROOT="$(cd "$HERE/../.." && pwd)"
DIST="$SCENARIO_ROOT/sidecar/dist/index.js"

if [ ! -f "$DIST" ]; then
  echo "error: sidecar dist not found at $DIST — run 'pnpm build' in sidecar/" >&2
  exit 2
fi

FIXTURES=(ts-junk-drawer ts-jsdoc-tags ts-usage-facts)

for name in "${FIXTURES[@]}"; do
  fix="$HERE/$name"
  if [ ! -d "$fix" ]; then
    echo "error: fixture dir missing: $fix" >&2
    exit 2
  fi
  out="$(mktemp)"
  printf '{"type":"handshake","request_id":"1","protocol_version":1}\n{"type":"extract","request_id":"2","project_path":"%s"}\n{"type":"shutdown"}\n' "$fix" \
    | node "$DIST" 2>/dev/null \
    | sed -n '2p' \
    | jq -S '.graph' > "$out"

  target="$fix/expected-graph.json"
  if [ "${UPDATE_FIXTURES:-0}" = "1" ]; then
    mv "$out" "$target"
    echo "updated: $target"
  else
    if [ ! -f "$target" ]; then
      echo "missing committed expected: $target" >&2
      mv "$out" "$target.live"
      echo "  wrote live capture to $target.live"
      exit 3
    fi
    if diff -u "$target" "$out" > /dev/null; then
      echo "ok:      $name"
    else
      echo "diff:    $name"
      diff -u "$target" "$out" || true
    fi
    rm -f "$out"
  fi
done
