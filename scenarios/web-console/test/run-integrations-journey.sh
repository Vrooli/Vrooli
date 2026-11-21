#!/usr/bin/env bash
set -euo pipefail
scenario_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
journey_evidence="${INTEGRATIONS_JOURNEY_EVIDENCE:-$HOME/.vrooli/evidence/agent-connect/t5-integrations}"
mkdir -p "$journey_evidence"
cd "$scenario_dir/ui"
pnpm run build --outDir "$journey_evidence/ui-dist" > "$journey_evidence/ui-build.log" 2>&1
cd "$scenario_dir/api"
LAUNCHER_JOURNEY_EVIDENCE="$journey_evidence" GOWORK=off go test -tags=journey -v -count=1 -timeout 15m -run '^TestJ14_' . 2>&1 | tee "$journey_evidence/J14-web-console.log"
cd "$scenario_dir/../git-control-tower/ui"
pnpm run build --outDir "$journey_evidence/gct-ui-dist" > "$journey_evidence/gct-ui-build.log" 2>&1
cd "$scenario_dir/../git-control-tower/api"
INTEGRATIONS_JOURNEY_EVIDENCE="$journey_evidence" GOWORK=off go test -tags=journey -v -count=1 -timeout 10m -run '^TestJ14_' . 2>&1 | tee "$journey_evidence/J14-gct.log"
