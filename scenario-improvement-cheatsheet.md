# Scenario Improvement Cheatsheet

Events ledger path: `scenario-improvement-events.ndjson`
Summary files: `scenario-summary.json`, `scenario-summary.txt`

- Success/timeout/fail totals and success rate (last 2000 events):
```bash
tail -n 2000 scenario-improvement-events.ndjson | jq -s '
  def finishes: map(select(.type=="finish"));
  (. // []) as $e | ($e|finishes) as $fin |
  {runs: ($fin|length),
   ok: ($fin|map(select(.exit_code==0))|length),
   timeout: ($fin|map(select(.exit_code==124))|length),
   fail: ($fin|map(select(.exit_code!=0 and .exit_code!=124))|length)}
  | . + {success_rate: (if .runs==0 then 0 else (.ok/.runs) end)}'
```

- Recent finishes (last 10):
```bash
tail -n 2000 scenario-improvement-events.ndjson | jq -c 'select(.type=="finish") | {ts,iteration,exit_code,duration_sec}' | tail -n 10
```

- In-flight runs (started but not finished):
```bash
tail -n 2000 scenario-improvement-events.ndjson | jq -s '[ group_by(.pid)
  | map({pid: .[0].pid,
         started:(map(select(.type=="start"))|length),
         finished:(map(select(.type=="finish"))|length)})
  | map(select(.started > .finished)) ]'
```

- Duration stats (avg, p50, p90, p95, p99):
```bash
tail -n 2000 scenario-improvement-events.ndjson | jq -s '
  def finishes: map(select(.type=="finish"));
  (. // []) as $e | ($e|finishes|map(.duration_sec)|map(select(type=="number"))) as $d |
  {avg:(if ($d|length)==0 then null else ($d|add/($d|length)) end),
   p50:($d|sort|.[(length*0.5|floor)]),
   p90:($d|sort|.[(length*0.9|floor)]),
   p95:($d|sort|.[(length*0.95|floor)]),
   p99:($d|sort|.[(length*0.99|floor)]),
   min:($d|min), max:($d|max)}'
```

- Last 5 errors/timeouts:
```bash
tail -n 2000 scenario-improvement-events.ndjson | jq -c 'select(.type=="finish" and .exit_code!=0) | {ts,iteration,exit_code,duration_sec}' | tail -n 5
```

- Hourly aggregation (counts and avg duration):
```bash
tail -n 2000 scenario-improvement-events.ndjson | jq -s '
  [ .[] | select(.type=="finish")
    | {h: (.ts|split(":")[0]), exit_code, duration_sec} ]
  | group_by(.h)
  | map({hour: .[0].h,
         count: length,
         ok: (map(select(.exit_code==0))|length),
         timeout: (map(select(.exit_code==124))|length),
         avg_duration: (map(.duration_sec)|add/length)})'
```

Tip: Set `SCENARIO_EVENTS_JSONL` to the absolute path and use in scripts. Replace `tail -n 2000 scenario-improvement-events.ndjson` with `tail -n 2000 "$SCENARIO_EVENTS_JSONL"`. 