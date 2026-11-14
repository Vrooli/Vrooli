# Archived workarounds — 2026-10-12

### DPF-WA-010 — 2026-10-08 — interactive worker prompt delivery loses its Web Console session

The F1 size-compliance worker `88b08d09-aa2c-498b-8c38-62df50025bf5`
(task `1b524404-9e1b-4a31-9cfa-254220b86ee8`) failed before receiving its
prompt. `agent-manager run get --json` reports `interactive launch failed:
deliver initial prompt to codex: web-console paste prompt: not_found` for the
new session. A prior F1 worker (`3a13d5cf-c5af-4456-9abd-a1fea0d13361`) failed
at the same session boundary while waiting for the Web Console shell. Between
the attempts, `vrooli scenario status web-console` reported `running` and
`healthy`, but that did not verify session creation or prompt delivery.

Fallback: preserve the worker-card acceptance rule and do not edit product code
as orchestrator. The original no-retry condition is superseded by operator
feedback DPF-FB-013 below; it permits two consecutive fresh attempts before
re-parking for external repair.

Operator feedback DPF-FB-013 updates the retry condition: dispatch is intermittent,
and a fresh D8 retry is authorized after later interactive runs completed. Attempt
`8569df86-5f74-457e-b1a0-409df6d6277c` (task
`23b47e09-2328-43a6-87fe-60e2a4512ca9`, workload
`desktop-packaging/F1-size-retry-1`) was admitted at 2026-10-08T04:55Z and failed
at 04:55:28Z before prompt delivery: `web-console paste prompt: not_found` for
session `77daaea4-c4ae-4391-9b64-3d0a2680c851`. This is the first consecutive
fresh attempt failure under DPF-FB-013. Attempt 2,
`3c852ed9-63fc-4745-9899-eaf269675fd0` (task
`383be0b0-6144-4282-b2db-983cf77883fa`, workload
`desktop-packaging/F1-size-retry-2`), received its prompt, ran from
04:56:18Z–05:03:44Z, and completed blocked: D8 reduced F1's runtime delta to
+47, but the remaining code is outside the admitted deletion list. The two
fresh attempts were not consecutive pre-prompt failures, so DPF-FB-013's
external-repair condition did not fire. Related bug report:
`knw-1791435299428858340`; friction report:
`knw-1791429534048076654`.

### DPF-WA-011 — 2026-10-08 — Staticcheck binary is older than the Go module

During D8 audit, `staticcheck ./cmd/vrooli-dist` could not run because the
installed analyzer supports Go 1.24 while the module requires Go 1.25. The
worker used passing focused Go tests and
`golangci-lint run --enable-only=unused ./cmd/vrooli-dist` as available checks.

Fallback: retain those owner checks and do not retry this binary. Revisit only
when a Go 1.25-compatible Staticcheck is available; this tool gap does not block
the remaining epoch decision.

### DPF-WA-012 — 2026-10-12 — F1 Git Control Tower proposal cannot recover late-anchored changes

F1's retained proposal anchors `gcta-62dc30c5bad8` and
`gcta-f0ee057b8a3d` were captured after the source work; the strict proposal
builder refuses paths unchanged since the latest anchor. Per operator feedback
DPF-FB-015, proposal creation is best-effort bookkeeping and does not gate
acceptance or E1 admission. F1 was accepted from its mechanical gates without a
proposal; do not retry these anchors. Future epochs are anchored at admission.
