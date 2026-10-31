# Semantic journey evidence

`semantic-journey-matrix.json` is the requirement-to-journey index for D01–D25
and the phase-2 obligations. It distinguishes a journey that has a real
driver from a deliverable that is still awaiting its target-specific receipt;
`planned` rows are not coverage claims.

The independent outcome fixture is
[`api/testdata/semantic-journeys.json`](../api/testdata/semantic-journeys.json).
It defines the operator choices and exact normalized state expected from the
real API and CLI entry points. The API test starts the production router,
invokes the generated Connect client, builds the scenario CLI, invokes its
manifest-driven command, and compares both outcomes to the fixture. It ignores
only timestamps and records the artifact and target in the fixture.

The browser half is intentionally not reimplemented in that Go test. BAS owns
the real browser journey and its recorded evidence in
[`bas/evidence/experience-journeys.json`](../bas/evidence/experience-journeys.json).
The browser cases assert semantic controls and readiness surfaces; they are
not treated as proof merely because a page loaded.

Two negative controls are part of the corpus: malformed CLI input must fail
before writing state, and a deliberately wrong expected state must fail the
parity assertion. These protect the evidence harness from becoming a
visibility-only or self-confirming check.
