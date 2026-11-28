#!/usr/bin/env bash
set -euo pipefail
scenario_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
journey_evidence="${LAUNCHER_JOURNEY_EVIDENCE:-$HOME/.vrooli/evidence/agent-connect/t3}"
mkdir -p "$journey_evidence"
if [[ "${1:-J7}" == "J8" ]]; then
  (cd "$scenario_dir/../.." && GOWORK=off go build -o "$journey_evidence/vrooli" ./cmd/vrooli)
fi
cd "$scenario_dir/ui"
pnpm run build --outDir "$journey_evidence/ui-dist" > "$journey_evidence/ui-build.log" 2>&1
cd "$scenario_dir/api"
LAUNCHER_JOURNEY_EVIDENCE="$journey_evidence" GOWORK=off go test -tags=journey -v -count=1 -timeout 15m -run "^Test${1:-J7}_" . 2>&1 | tee "$journey_evidence/${1:-J7}-current.log"
