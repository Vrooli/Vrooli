#!/usr/bin/env bash
# recurring-scan-cpu-sampler.sh — Before/after CPU + exec-sighting sampler.
#
# Objective evidence source for the "Recurring-Scan CPU Efficiency" plan
# (security-health / vrooli-autoheal / system-monitor). Mirrors the original
# manual investigation: it snapshots `ps` at a fixed interval over a window,
# accumulates per-command CPU, and tallies sightings of the recurring-scan
# subprocesses each scenario spawns (osv-scanner, journalctl, nvidia-smi, plus
# the three scenario API binaries themselves).
#
# It does NOT need root: it samples /proc via `ps`. Short, bursty subprocesses
# are caught probabilistically by the sampling cadence — the same limitation
# the original 7-sample/2-minute investigation had, so the method is comparable
# before vs after. For a tighter exec count, set --interval low (e.g. 2s).
#
# Usage:
#   scripts/perf/recurring-scan-cpu-sampler.sh [--window SECS] [--interval SECS] \
#       [--label before|after] [--out DIR]
#
# Output: writes <out>/<label>-<timestamp>.json and a human summary to stdout.
# Default out dir: /tmp/recurring-scan-perf

set -u

WINDOW=120
INTERVAL=6
LABEL="sample"
OUT_DIR="/tmp/recurring-scan-perf"

while [ $# -gt 0 ]; do
  case "$1" in
    --window)   WINDOW="$2"; shift 2 ;;
    --interval) INTERVAL="$2"; shift 2 ;;
    --label)    LABEL="$2"; shift 2 ;;
    --out)      OUT_DIR="$2"; shift 2 ;;
    -h|--help)
      grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

mkdir -p "$OUT_DIR"
TS="$(date +%Y%m%d-%H%M%S)"
RAW="$(mktemp)"
trap 'rm -f "$RAW"' EXIT

# Targets we tally sightings of (recurring-scan subprocesses + scenario APIs).
TARGETS="osv-scanner journalctl nvidia-smi security-health-api vrooli-autoheal-api system-monitor-api"

SAMPLES=$(( WINDOW / INTERVAL ))
[ "$SAMPLES" -lt 1 ] && SAMPLES=1

echo "Sampling for ${WINDOW}s (every ${INTERVAL}s, ${SAMPLES} samples), label=${LABEL}..." >&2

i=0
while [ "$i" -lt "$SAMPLES" ]; do
  # pid ppid %cpu comm args — one block per sample, marked with sample index.
  ps -eo pid=,ppid=,pcpu=,comm=,args= --sort=-pcpu 2>/dev/null \
    | awk -v s="$i" '{print s"\t"$0}' >> "$RAW"
  i=$(( i + 1 ))
  [ "$i" -lt "$SAMPLES" ] && sleep "$INTERVAL"
done

JSON_OUT="${OUT_DIR}/${LABEL}-${TS}.json"

# Aggregate with awk:
#  - per-target: number of samples it appeared in (sighting count), summed %cpu,
#    peak %cpu, and process-instance count (distinct pids seen).
#  - top-10 commands by mean %cpu across samples.
awk -F'\t' -v samples="$SAMPLES" -v targets="$TARGETS" '
function basename_comm(c,   n, arr) { return c }
BEGIN {
  ntar = split(targets, T, " ")
}
{
  # fields after split: $1=sample idx, $2 = "pid ppid pcpu comm args..."
  line = $2
  # parse leading numeric columns from the ps line
  n = split(line, f, /[ \t]+/)
  # f may have leading empty due to spacing; find first non-empty
  idx = 1
  while (f[idx] == "") idx++
  pid = f[idx]; ppid = f[idx+1]; cpu = f[idx+2]; comm = f[idx+3]
  # reconstruct args (everything from idx+4 on)
  args = ""
  for (j = idx+4; j <= n; j++) args = args (args==""?"":" ") f[j]

  # per-comm cpu accumulation (for top list)
  commCpu[comm] += cpu
  if (cpu > commPeak[comm]) commPeak[comm] = cpu
  commSeen[comm] = 1

  # target matching: match on comm OR within args (osv-scanner etc. may show as full path)
  for (t = 1; t <= ntar; t++) {
    tg = T[t]
    if (comm == tg || index(args, tg) > 0) {
      key = tg
      tgtCpu[key] += cpu
      if (cpu > tgtPeak[key]) tgtPeak[key] = cpu
      seenSample[key SUBSEP $1] = 1
      seenPid[key SUBSEP pid] = 1
    }
  }
}
END {
  # count sample-sightings and pid-instances per target
  for (k in seenSample) { split(k, a, SUBSEP); sight[a[1]]++ }
  for (k in seenPid)    { split(k, a, SUBSEP); pids[a[1]]++ }

  printf "{\n"
  printf "  \"window_samples\": %d,\n", samples
  printf "  \"targets\": {\n"
  first = 1
  for (t = 1; t <= ntar; t++) {
    tg = T[t]
    if (!first) printf ",\n"; first = 0
    printf "    \"%s\": {\"sample_sightings\": %d, \"distinct_pids\": %d, \"sum_cpu\": %.1f, \"peak_cpu\": %.1f}",
      tg, sight[tg]+0, pids[tg]+0, tgtCpu[tg]+0, tgtPeak[tg]+0
  }
  printf "\n  },\n"

  # top 10 commands by mean cpu
  ncmd = 0
  for (c in commCpu) { mean[c] = commCpu[c] / samples; ncmd++ }
  # simple selection sort for top 10
  printf "  \"top_commands_by_mean_cpu\": [\n"
  printed = 0
  for (rank = 0; rank < 10 && rank < ncmd; rank++) {
    best = ""; bestv = -1
    for (c in mean) {
      if (done[c]) continue
      if (mean[c] > bestv) { bestv = mean[c]; best = c }
    }
    if (best == "") break
    done[best] = 1
    if (printed) printf ",\n"; printed = 1
    printf "    {\"comm\": \"%s\", \"mean_cpu\": %.2f, \"peak_cpu\": %.1f}", best, mean[best], commPeak[best]
  }
  printf "\n  ]\n"
  printf "}\n"
}
' "$RAW" > "$JSON_OUT"

echo "Wrote $JSON_OUT" >&2
echo "--- summary (${LABEL}) ---"
cat "$JSON_OUT"
