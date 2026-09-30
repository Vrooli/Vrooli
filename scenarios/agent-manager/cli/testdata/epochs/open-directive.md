# E7 — fixture epoch

- Outcome: J02 passive record replays in a fresh context
- Kind: feature
- Started: 2026-09-29T08:00:00Z
- Estimate: 10 work units
- Expected size: 2000 net runtime lines
- Metric: journeys passing (journey suite)
- Workers: 11111111-1111-1111-1111-111111111111

## Exit gate
- journeys J02 pass

## Directives
- D1 2026-09-29T09:00:00Z Use the fixture site.
- D2 2026-09-29T09:30:00Z Park the flaky J23 step.

## Slice log
2026-09-29T08:00:00Z | unit 0 | exit metric=1 | net runtime lines=+50 | net test lines=+20 | ack=D1
not a slice line
2026-09-29T08:10:00Z | unit 1 | exit metric=2 | net runtime lines=+30 | net test lines=+5 | ack=-
ACCEPTED 2026-09-29T10:00:00Z J02 passes
