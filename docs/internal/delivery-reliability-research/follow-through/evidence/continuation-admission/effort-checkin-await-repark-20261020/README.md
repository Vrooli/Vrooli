# Effort check-in repark guidance (2026-10-20)

A recorded effort check-in resumes the same parked run and consumes its previous await. The run must re-read current feedback and its actual awaited condition; if the same producer is still unresolved, it must park that same run again with the same producer/key before ending the turn. The check-in remains neutral and does not imply child completion, approval, or scope change.

The native Hub parent 11175250-c50d-44e9-9af5-cd6eba79bab2 was observed terminal `complete` at 23:27:23Z while its E8 child remained live. Its bounded event range recorded a successful park, then a signal check-in, then terminal completion without a repark or child-end event. Source explains the result: `WakeRun` clears the await handle; continuation completion preserves a park only when the run is parked at turn end. This is an actual lost wait, not a false park receipt.

At the original observation, no same-run recovery route was available under the then-held authority. PM `recoverPriorFiniteOwner` resumes only nonterminal runs; when the retained run is terminal it returns to ordinary admission, which may start a new run. Agent Manager `ContinueRun` is the existing lifecycle route and requires exact `agent.manage` for `agent-manager:run/<run-id>` absent applicable lineage. The completed parent has no await handle. No heartbeat trigger or run mutation was issued because it would not resume that same wait.

## Frozen source

- Patch: `owned.patch` SHA-256 `73eab19077d386e85af7dd6f4784dfd22e12957a7e9fdb1a098750337dd32533`.
- Final source manifest: `final-source-SHA256SUMS.txt` SHA-256 `d0de85cffe8828edf3807ab09efc9220e98d4399ee466eec9afb8d34742ff18a`.
- Captured preimage manifest: `preimage/SHA256SUMS.txt` SHA-256 `11ae96d2c3bf9f58552b07656dc3f6b728190be0d4aad7744ce740612971cfde`.
- Four owned paths are listed in both manifests. Applying the patch to those captured preimages reconstructed all four current files byte-for-byte.
- The canonical skill sidecar is unchanged; it was refreshed only through `skill-sidecar-gen` and read back with `prompt-manager skill read large-effort-orchestration`.

## Validation

- `cd scenarios/agent-manager/api && GOMAXPROCS=2 go test -p 1 -v ./internal/orchestration -run '^TestWakeEffortCheckinCommitsQueuedCursorWithRetainedRunWake$' -count=1` — PASS; the named test verifies the neutral prompt explicitly instructs re-reading current state and reparking with the same producer/key, and rejects feedback/payload text leakage.
- `git diff --check` on the owned paths — PASS.
- Prompt Manager canonical readback confirms the skill text. No native projection, restart, run wake/continue, heartbeat trigger, or product write was performed.

## Limitation

This change prevents the same premature turn-end behavior on future signal check-ins; it does not restore the already-completed parent or prove that its child work was accepted. The native owner must resume through its authorized lifecycle route or use its next governed owner flow.

## Serving adoption and native consumer

Later consent a1c5b219 approved the two exact original parent continuations; they succeeded without a new owner. The reviewed AM runtime source was adopted through normal managed restart `startop-ef61b76cff98708b95900218d0f31778`, exit0 at2026-10-21T00:20:05Z. Health/readiness true, serving build fa9af9a25c7472fb08a743c2b40031433e382e5291c2d74c31766ea10a1aa99a. Full output is `adoption-restart.log`. Hub parent event2963 at00:24:41 carries the new neutral check-in text, event2997 steers the same child, event2999 parks the same parent, and current native await was registered00:25:34. This is actual prompt receiving and repark behavior, not a whole-delivery acceptance. New verified-self guidance is separately published as canonical metadata revision53; this original repark-only patch kept revision52 metadata.
