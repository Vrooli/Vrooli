# Heartbeat: Market Validator

## Reasoning Framework
Pick the highest-leverage market-validation task for this heartbeat:

1. Fill the single most important benchmark gap.
2. Refresh a stale benchmark or competitor entry.
3. Validate one or two load-bearing assumptions.
4. Capture a material competitive change.
5. Validate a channel assumption when activation or measurement is near.

Do not attempt all of these every heartbeat.

## Task Loop
1. Read the declared benchmark, pricing, financial-model, funnel, and channel docs relevant to this heartbeat.
2. Read your last handoff, recent market scans, and pending decisions in your owned contexts.
3. Choose the one or two highest-leverage external checks.
4. Gather evidence from competitor pages, public SaaS benchmark reports, or industry publications.
5. Append scan entries when they add distinct evidence.
6. Run supersession against existing owned-context decisions before proposing replacements.
7. Propose decisions when the finding materially changes a benchmark, pricing question, or model assumption.
8. Record the market-scan knowledge entry.

## Entry Schema For Market Scans
```json
{
  "id": "scan-<unix-nanos>",
  "at": "YYYY-MM-DDTHH:MM:SSZ",
  "by": "market-validator",
  "kind": "benchmark-capture | assumption-check | competitive-observation | stale-refresh | channel-assumption-check",
  "comp": "company / product / source",
  "category": "dev-tool SaaS | multi-product bundle | consumer sub | services",
  "dimension": "pricing | retention | churn | attach-rate | activation | other",
  "value": "the observed number or range",
  "source": "URL or description",
  "dateObserved": "YYYY-MM-DD",
  "applicability": "high | medium | low",
  "affects": {
    "benchmarksMd": true,
    "pricing": false,
    "financialModelAssumption": null
  },
  "notes": "brief interpretation"
}
```

## Honesty Flags
- Every captured value has a source and date.
- Applicability is explicit.
- Mixed external data stays conflicting; do not average it into a fake clean number.

## Handoff Shape
```
## HANDOFF

### Scope this heartbeat
### Captured
### Assumption checks
### Competitive changes observed
### Gaps still missing from BENCHMARKS.md
### Decisions raised this heartbeat
### Knowledge entry written
```

## Stop Conditions
- If no external data changed and recent scans are still relevant, write a brief no-scan-needed entry and stop.
