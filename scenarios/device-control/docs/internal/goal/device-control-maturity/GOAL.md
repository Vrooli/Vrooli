# Copyable Agent Goal

Complete the Device Control and Portal maturity effort: implement and evidence TV and Android journeys across supported transports, report truthful gaps, and obtain two fresh hostile reviews with no material in-scope defect.

Use `scenarios/device-control/docs/internal/goal/device-control-maturity/` as authority; read its CHARTER, STATUS, FEEDBACK, LOG and six epochs. Follow `large-effort-orchestration`, `harness-goal-authoring`, `device-control`, `device-control-improve`, `scenario-work-ladder`, `improvement-do-and-dont` and `docs/TESTING.md`. Work in outcome epochs with one coordinator and fresh workers as useful. Continue through E6; a green test or time limit is not completion.

Implement shared lifecycle/readiness and capture; Android wireless setup and verified artifact install; screenshot/recording fidelity, performance and optional audio; semantic/AI control; Portal launcher diagnostics and owned-app start/restart/stop. Fix in-scope infra at its owner and reduce related debt. Verify device identity; do not trigger phone projection consent while the user is away; never infer Wi-Fi sleep causality from one outage. Respect artifact, control-plane and Portal owners; no raw-ADB bypass or hand-edited generated evidence.

Require regressions, terminal scoped Test Genie results, matched capture/performance evidence, measurable AI outcomes and current-identity live proof. Run `business-health validate scenario device-control`, `vrooli scenario requirements validate device-control` and `git diff --check`; use owning gates for Portal/artifact changes. **Blocked means a decision, credential, or approval you lack. Name it. Friction you can diagnose is not blocked.** Continue independent work during consent waits.

At interruptions update STATUS, LOG and active epoch with proof, risks and next action. Never mark complete due to time/budget. Final report: changed, verified commands/results and device evidence, remaining, unverified, infrastructure fixed/open.
