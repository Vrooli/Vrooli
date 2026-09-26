# Visited Tracker requirements

The module JSON files are the requirement registry. Their `prd_ref` values link
obligations to the operational targets in [PRD.md](../PRD.md).

- [Campaign tracking](01-campaign-tracking/module.json) owns campaign identity,
  visits, staleness, the CLI, and file persistence.
- [Web interface](02-web-interface/module.json) owns the HTTP and UI obligations,
  including export/import. Consult each requirement's title; IDs are stable and
  are not interchangeable with an older prose index.
- [Advanced features](03-advanced-features/module.json) records the extended
  capability obligations and their current validation gaps.
- [Phase integration](04-phase-integration/module.json) owns agent-facing
  coordination, revision-aware attention, claims, and program composition.

Run validation with `vrooli scenario test visited-tracker`. Test Genie owns
execution and the terminal receipt. Requirement auto-sync derives evidence
status from managed validation; do not hand-mark requirements passed or widen
assertions to match a failure. A source file reference is not execution proof.

`api/attention_test.go` exercises claim concurrency, expiry, replay, revision
changes, cancellation, and bounded exploration. `api/attention_program_test.go`
executes the shipped program sources with controlled bindings.
`api/import_test.go` includes a real HTTP export/import round-trip through the
production router. The historical `test/api/http-api.bats` path does not exist
and is not a validation source.

Use `business-health validate scenario visited-tracker` and
`vrooli scenario requirements validate visited-tracker` to check links and
obligations. These checks do not replace behavioral tests or establish a live
correctness or durability sensor.
