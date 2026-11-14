# Device Studio requirements

This registry is the implementation-facing claim for the wizard-authored PRD.
Each operational target imports a numbered module through index.json.
Some requirements now point to passing E1/E2 tests and retained current-host
still-work evidence. Those references cover specific slices only; the committed
P0 release remains incomplete. In particular, physical-device qualification,
full-resolution readability, motion/video, rights, AI, desktop packaging and
other unqualified obligations remain open.

Manual verification protocols remain planned and have no manual attestations.
A linked test, screenshot, or live journey is not by itself proof of the full
requirement. Requirement and validation statuses are earned through the owner
sync from executed evidence; do not force them or edit generated snapshots.
P1/P2 delivery_scope is roadmap and is explicitly outside the P0 commitment.

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
