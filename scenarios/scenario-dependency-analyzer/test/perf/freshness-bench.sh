#!/usr/bin/env bash
set -euo pipefail

label="run"
if [[ "${1:-}" == "--label" ]]; then
  label="${2:?--label requires a value}"
  shift 2
fi
out_dir="${1:?usage: freshness-bench.sh [--label name] output-dir}"
case "$out_dir" in
  /tmp|/tmp/*) echo "refusing to write measurement state under /tmp" >&2; exit 2 ;;
esac
mkdir -p "$out_dir"
command -v scenario-dependency-analyzer >/dev/null 2>&1 || { echo "scenario-dependency-analyzer is required on PATH" >&2; exit 2; }

result="$out_dir/${label}.json"
report="$out_dir/${label}.report.json"
facts="$out_dir/${label}.facts"
{
  echo "cores=$(nproc)"
  echo "ram_bytes=$(awk '/MemTotal:/ {print $2 * 1024}' /proc/meminfo)"
  echo "goproxy=$(go env GOPROXY)"
  echo "goversion=$(go env GOVERSION)"
  echo "git_head=$(git rev-parse --short HEAD)"
} >"$facts"

timings=()
last_report=""
for run in 1 2 3; do
  timing="$out_dir/${label}.${run}.time"
  last_report="$(/usr/bin/time -p -o "$timing" scenario-dependency-analyzer freshness --touched --json)"
  timings+=("$(awk '/^real / {print $2}' "$timing")")
done
printf '%s\n' "$last_report" >"$report"

python3 - "$facts" "$result" "$label" "${timings[@]}" <<'PY'
import json, pathlib, sys
facts_path, result_path, label, *timings = sys.argv[1:]
facts = {}
for line in pathlib.Path(facts_path).read_text().splitlines():
    key, value = line.split('=', 1)
    facts[key] = value
report_path = pathlib.Path(result_path).with_suffix('.report.json')
report = json.loads(report_path.read_text())
out = {'label': label, 'host': facts, 'timings_seconds': [float(x) for x in timings], 'report_path': str(report_path), 'report': report}
pathlib.Path(result_path).write_text(json.dumps(out, indent=2) + '\n')
print(f"{report['summary']['checked']}/{report['summary']['clean']}/{report['summary']['stale']}/{report['summary']['errors']}/{report['elapsed_ms']}")
PY
