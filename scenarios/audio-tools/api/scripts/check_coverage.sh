#!/usr/bin/env bash
# Coverage gate for audio-tools/api.
#
# Runs `go test -cover ./...` and fails when any listed package drops
# below its floor. The script prints the actual percent and the gap so
# reviewers can immediately tell whether the regression is a covered-code
# deletion (legitimate floor lower) or a missing test.
#
# Floors are sourced from
# ~/.vrooli/plans/audio-tools-test-architecture-coverage-hardening.md
# §7 Phase 6. Update both the plan and this script in the same PR.

set -euo pipefail

cd "$(dirname "$0")/.."

# Per-package floors (whole-number percent). Use the integer floor so
# the comparison stays in shell arithmetic without bc.
declare -A FLOORS=(
  ["audio-tools/internal/byok/envelope"]=90
  ["audio-tools/internal/protomap"]=80
  ["audio-tools/internal/clock"]=100
  ["audio-tools/internal/database"]=80
  ["audio-tools/internal/middleware"]=80
  ["audio-tools/internal/modules"]=80
  ["audio-tools/internal/ai/chains"]=80
  ["audio-tools/internal/server"]=80
  ["audio-tools/internal/session"]=80
  ["audio-tools/internal/stt/strategy"]=80
  ["audio-tools/internal/capabilities"]=70
  ["audio-tools/internal/store"]=70
  ["audio-tools/internal/byokstore"]=70
  ["audio-tools/internal/byok"]=70
  ["audio-tools/internal/summarize"]=80
  ["audio-tools/internal/stt/segmenter"]=75
  ["audio-tools/internal/ai/sttchain"]=80
  ["audio-tools/internal/ai/summarizechain"]=85
  ["audio-tools/internal/ai/ttschain"]=80
  ["audio-tools/internal/diagnostics"]=90
  ["audio-tools/handlers/stt"]=80
  ["audio-tools/internal/audio"]=75
  ["audio-tools/internal/httpx"]=70
  ["audio-tools/internal/stt/pipeline"]=20
  ["audio-tools/internal/tts"]=70
  ["audio-tools/internal/usagereport"]=65
  ["audio-tools/internal/testutil/vendorws"]=80
)

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

# `go test -cover ./...` prints one line per package: "ok\tpkg\ttime\tcoverage: NN.N% of statements".
go test -cover ./... > "$tmp" 2>&1 || true

failures=0
for pkg in "${!FLOORS[@]}"; do
  floor=${FLOORS[$pkg]}
  line=$(grep -E "[[:space:]]${pkg//\//\\/}[[:space:]]" "$tmp" || true)
  if [[ -z "$line" ]]; then
    echo "WARN: $pkg not found in go test output (skipping)"
    continue
  fi
  pct=$(echo "$line" | sed -n 's/.*coverage: \([0-9]*\)\.[0-9]*%.*/\1/p')
  if [[ -z "$pct" ]]; then
    echo "WARN: $pkg coverage not parseable from: $line"
    continue
  fi
  if (( pct < floor )); then
    gap=$(( floor - pct ))
    echo "FAIL: $pkg coverage ${pct}% < floor ${floor}% (gap ${gap}pp)"
    failures=$(( failures + 1 ))
  else
    echo "OK:   $pkg coverage ${pct}% >= floor ${floor}%"
  fi
done

if (( failures > 0 )); then
  echo
  echo "$failures package(s) below coverage floor."
  exit 1
fi
echo
echo "All listed packages meet their coverage floors."
