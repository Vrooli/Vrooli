# Crypto Infrastructure CI2 evaluator consumer

This is the first importing consumer for `github.com/vrooli/chain-forensics`.
It evaluates the package's typed `Case` contract, invokes its canonical
validation and private packet serializer, and reports chain coverage, flow
grades, legal-claim boundaries and residual risks. It intentionally contains
no duplicate case, occurrence, edge, source or packet models.

From this directory, `go test ./...` exercises the evaluator against the shared
package's bundled public case. `go run ./cmd/ci2-evaluate <private-packet.json>`
emits the evidence inventory. Add `--seeded` to produce the reproducible
centralized, partial, and full-profile deployment reports and their linkage
attack results. The seeded profiles are synthetic fixtures: exact transaction
and address fields, coarse asset/quantity fields, and a bounded negative
control using a supported address field absent from the case are tested against
the packet's occurrences. Each finding keeps
the adversary, layer, declared vantage, UTC interval, observed fields, matched
occurrence IDs, and source IDs. The attack interval is applied to occurrence
timestamps, and the adversary/vantage must appear in that layer's explicit
observer/control inventory. Inventory coverage is not a privacy score; exact field matches do
not establish a person, intent, control, or legal responsibility. A failed
attack only means no match for those fields and that evidence window.

Packet response input is bounded at 128 MiB so the evaluator can consume larger
retained owner exports while keeping memory use finite; each individual source
body remains subject to the shared package's 4 MiB validation limit.

The local `replace` keeps this monorepo checkout runnable;
release builds should resolve a tagged `github.com/vrooli/chain-forensics`
version and remove that development replacement.

The report is evidence coverage, not an anonymity score or a claim that a
system is decentralized. `EvaluateDeployment` requires all seven inventories
(payment, naming, storage, compute, gateway, accounts, recovery) and fails
closed on missing layers or incomplete adversary/time-window context. CI2's
full mission still requires measured deployment-specific inventories, the
censored-network reachability test, and evidence from actual centralized,
partial, and full deployment targets; seeded profiles do not close those
targets. An occurrence without a valid UTC timestamp is excluded from a bounded
time-window attack and counted in `unverifiable_occurrence_count`; it can
contribute to neither a match nor a no-match conclusion. Seeded profiles choose
a timestamped occurrence deterministically and remain synthetic regression
fixtures.
