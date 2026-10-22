# Independent review — Planner E1 routed CLI actor propagation test harness

Reviewer only. Do not edit files or rerun tests. Exact parent run: `5d07e651-eaa6-48ec-89b4-5dc359059193`; task `790cb264-1a63-47eb-89e6-3b9ee3e16c23`.

## Bounded scope and native worker identity

Review only the Planner test-harness correction in `scenarios/personal-planner/api/handlers/calendar/connect_events_test.go`. Worker task/run: `9924b9af-32d1-45de-b0c6-c2dd8e1dce4a` / `69ef5a17-940f-4bd0-b09e-4cdfc4d8aa49`, registered delivery profile `4c19aef1-36a2-4cc5-8961-42648692883b`; native run metadata confirms this profile; exact parent above. Native task/run completed, exit 0, but run final-output selection is ambiguous. Do not rely on an unselected conversational final; inspect source bytes, native tool evidence, and retained receipts directly.

Baseline: `baseline/manifest.json`, SHA-256 `5a18e4b7557efa37bd8f1e3476ff789f6714891c575b1f83cfd61096604ff28d`. Current manifest: `current-manifest.json`, SHA-256 `ae9886a0a1778c87fc01e25232587e94480a7b403efd8235ee8a8caaaee9a0a0`. Baseline and current app/calendar CLI runtime files are byte-identical. Only the existing integration test changed: baseline 506 lines, current 514 (+8), current SHA-256 `421c22171f00b2ca39ea5abac079abd161a0c4e0ddeaeddf6628ae21903404cb`. Exact baseline patch is `worker-evidence/source-diff.patch`.

## Evidence and review questions

- Worker traced an inherited `VROOLI_AGENT_IDENTITY_TOKEN` in its environment. Shared CLI transport adds it as `X-Agent-Identity-Token`; the integration server intentionally configures only the synthetic bearer provider, so that ambient agent channel is invalid in this harness. Assess whether filtering inherited agent/API credential variables from the CLI and UI child process environments preserves the explicit synthetic bearer setup and avoids changing product auth behavior. No credential value was recorded.
- The same test deliberately sends REST/Connect requests for `synthetic-planner-owner-b`; the old observer required every actor to equal `synthetic-planner-actor`. Assess whether the new assertion allows exactly the two exercised synthetic owners and requires owner B to be observed, without weakening unknown-actor rejection.
- Focused command: `cd scenarios/personal-planner/api && go test ./internal/calendar ./handlers/calendar -count=1`. Retained stdout: `worker-evidence/focused-calendar.stdout`, SHA-256 `866a80ff667d0d3cadb4091ad32e0cadc70c22b2a4a5d297d35b68d1c98882a9`, exit 0, 2 lines / 94 bytes; it reports both packages `ok`. Stderr is empty, SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
- Worker epoch check: `worker-evidence/epoch-check.stdout`, SHA-256 `585d23c557bd30ce18ead423c97f6fb415f6c5dc465090840ed4cd084fed2789`, exit 0. It reports 44 work units, +1,731 runtime/+2,233 test lines, no step-back, a historical wall-clock review prompt, and `legacy-gates` warning (full E1 acceptance not checked). Do not infer E1/Q1 or new-event ownership acceptance.

Assess scope/hash accuracy, source behavior and whether the focused run supports accepting this bounded test-harness outcome with limits. State remaining uncertainty, especially that this is synthetic harness isolation, not real cross-surface human/agent identity qualification. Do not rerun tests or change source. Return an explicit PASS/FAIL verdict with rationale and limits.
