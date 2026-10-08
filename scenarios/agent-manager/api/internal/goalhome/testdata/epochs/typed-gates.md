# E26 — One ReplaySpec renderer for standalone HTML replay (typed gates)

- Outcome: RD-EXP first tranche: standalone HTML replay archives use the existing ReplaySpec-driven `ReplayPlayer` renderer.
- Kind: refactor
- Started: 2026-10-07T01:32:16Z
- Estimate: 24 work units
- Yield estimate: −600 runtime lines
- Metric: one ReplaySpec-driven renderer serves the composer page and the standalone archive
- Workers: e8c322ca-72e7-434c-9d37-1d2cf853961f
- Status: in-progress

## Exit gate

The admitted gate items; acceptance checks exactly these.

- G1 test: cd api && go test ./services/export/... ./handlers/ -run Export
- G2 test: cd ui && pnpm type-check && npx vitest run src/export
- G3 journey: J01,J02,J03,J04,J07,J08,J13,J15,J17,J18,J23,J25 @ bas-goal
- G4 inventory: runtime_lines <= 253452 from 253452 (python3 docs/internal/refactor_inventory.py --no-git)
- G5 review: deletion-list
- G6 custom: offline archive playback in a browser with network disabled — reason: no automated offline check exists
- G7 custom: generated ui/dist identity matches the pre-write baseline

## Gate amendments

- A1 2026-10-07T14:30:00Z G7 drop — reason: BAS-FB-064, generated output identity is not a gate
- A2 2026-10-07T14:31:00Z G6 unverified — reason: offline browser tool unavailable after one authorized attempt; logged in WORKAROUNDS.md
- A3 2026-10-07 G5 drop

## Directives

- D1 2026-10-07T02:18:00Z Spend trigger fired. Stop repeated evidence searches and record the gate results.

## Slice log

2026-10-07T01:50:00Z | Replaced the Go presentation template with ReplayExportPage packaging and an embedded ReplaySpec bootstrap. | exit metric=one renderer | net runtime lines=-801 | net test lines=+21 | gate G1=pass | gate G2=pass | ack=-
2026-10-07T02:09:17Z | Journeys 12/12 on bas-goal; no-Git inventory runtime=252651; pre-write baseline comparison=gap. | exit metric=12/12 | net runtime lines=-801 | net test lines=+0 | gate G3=pass 12/12 | gate G4=pass 252651 | gate G5=pass | gate G6=unverified | ack=D1
