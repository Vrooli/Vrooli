# Device Studio requirements

This registry is the implementation-facing claim for the wizard-authored PRD.
Each operational target imports a numbered module through index.json.
Every current product requirement is **not_implemented**; every validation is
**planned**, with evidence **not_run**. None of the historical experiments or
generated example code proves Device Studio product behavior.

P0 delivery_scope is committed and includes the entire accepted first release.
The initial photographic slice does not satisfy the remaining P0 scope.
P1/P2 delivery_scope is roadmap and is explicitly outside that commitment.
The not_implemented status is a truthful initial state, not a cancellation.

[verification-protocols.md](verification-protocols.md) provides actual written
Given/When/Then procedures, counterexamples and required evidence for each ID.
They are NOT EXECUTED. A manual ref to that file is a planned verification
method, not a manual attestation. Proposed automation is stored in supported
validation metadata; non-existent test paths are deliberately not referenced.
As real tests are authored, add their actual specific refs and [REQ:ID] tags.
Do not mark code behavior complete because the manual plan is written.

Use docs/internal/ACCEPTANCE.md for the cross-cutting release/craft matrix.
Run structural validation with:
    vrooli scenario requirements validate device-studio --json
Run the owner assessment with:
    business-health validate scenario device-studio --json
Use scoped Test Genie phases per docs/TESTING.md for implementation evidence.
Auto-sync earns statuses from genuine evidence; do not edit sync snapshots,
force sync, fabricate receipts or manually check the PRD boxes.
