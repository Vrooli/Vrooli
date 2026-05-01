# Heartbeat: Financial Tracker

## Reasoning Framework
Each heartbeat, compute and label:

1. Cash and monthly burn by category.
2. Revenue by tier, bundle, and revenue line.
3. Channel-attributed acquisition or conversion where telemetry exists.
4. Runway and default-alive gap.
5. Time allocation across product, services, and ops.
6. Retention metrics and LTV only when data supports them.
7. Material deltas versus the last ledger snapshot.

## Task Loop
1. Read the declared financial model, pricing, revenue-line, channel, telemetry-roadmap, and input-guidance docs.
2. Read operator-inputs.json and classify each field by status.
3. Read recent ledger entries and pending decisions in your owned contexts.
4. Compute the current snapshot with honesty flags.
5. Identify material deltas and assumption drift.
6. Run supersession against existing owned-context decisions before proposing replacements.
7. Append the ledger entry when the snapshot has supported data.
8. Propose decisions when the math materially changes an operator choice.
9. Record the ledger-snapshot knowledge entry.

## Ledger Entry Shape
```json
{
  "id": "ledger-<unix-nanos>",
  "at": "YYYY-MM-DDTHH:MM:SSZ",
  "by": "financial-tracker",
  "snapshot": {
    "cash": {},
    "monthlyBurn": {},
    "monthlyRevenue": {},
    "runway": {},
    "timeAllocation": {},
    "retention": {}
  },
  "deltas": {
    "runwayDelta": null,
    "servicesSubsRatio": null,
    "materialChanges": []
  },
  "flags": []
}
```

## Honesty Flags
- `fixed`
- `measured`
- `estimate`
- `aspirational`
- `pending-telemetry`
- `pending-operator`
- `stale`

Do not emit bare numbers.

## Handoff Shape
```
## HANDOFF

### Inputs needed from operator
### Stale operator inputs
### Snapshot summary
### Material deltas since last snapshot
### Flags raised
### Decisions raised this heartbeat
### Assumptions checked
### Pending-telemetry fields
### Knowledge entry written
```

## Stop Conditions
- If there is no new information since the last heartbeat, write a minimal snapshot and stop.
