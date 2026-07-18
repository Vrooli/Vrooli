# macOS Field Shakeout Checklist (workspace-sandbox)

## Purpose

Everything in the cross-platform work is verified at the compile + unit
level on the Linux dev host (see
[`docs/internal/PORTABILITY_AUDIT.md`](../internal/PORTABILITY_AUDIT.md)).
This checklist is the **one thing that must be done on real Apple hardware**:
confirm the copy driver is selected, that a protected agent-manager run
executes under Seatbelt, and that Seatbelt actually denies the writes and
network it claims to.

It is written for an operator sitting at (or SSH'd into) a Mac. Run the steps
in order; each has an expected signature to eyeball. If a signature does not
match, stop and record what you saw — a mismatch here is the first evidence
that the OS seam behaves differently in the field than in the cross-compile
gate.

This checklist is one required part of the macOS setup qualification ladder.
Once it passes on a Mac reachable through Bridge, record the evidence in the
canonical [platform support matrix](../../../../docs/reference/platform-support.md)
before treating protected macOS runs as supported.

## Prerequisites

- macOS (Apple silicon or Intel) with `/usr/bin/sandbox-exec` present
  (ships with the OS). Confirm: `command -v sandbox-exec`.
- The Vrooli control plane (`vrooli`) available on PATH.
- No overlayfs/bwrap — those are Linux-only and are *expected* to be absent.
  The copy driver is the whole point of this run.

## Step 1 — Start the scenario via the lifecycle

Never run the binary directly; use the control plane so ports, process
naming, and health checks are wired.

```bash
vrooli scenario restart workspace-sandbox
vrooli scenario logs workspace-sandbox | tail -n 40
```

**Expected — boot SelectionReport.** The driver selection logs one
`candidate=` line per driver and one `selected=` summary. On a Mac, only the
copy driver is available, so the report must select it:

```
driver: candidate=overlayfs-userns state=unavailable reason="..."
driver: candidate=overlayfs-root  state=unavailable reason="..."
driver: candidate=fuse-overlayfs  state=unavailable reason="..."
driver: candidate=copy            state=selected    reason="..."
driver: selected=copy inUserNamespace=false preferenceUsed=false preferenceValue=""
driver selected | id=copy version=1.0
```

If a saved driver preference exists you will instead see
`driver: using copy (saved preference)` and `preferenceUsed=true`. Either is
fine as long as `selected=copy`.

**Fail signatures:** any `selected=` other than `copy`; a startup
`log.Fatal` about `overlayfs-userns selected but API is not running inside a
user namespace` (means selection wrongly picked an overlay driver — a real
regression on macOS).

## Step 2 — Confirm the reported containment

```bash
curl -s "http://127.0.0.1:${API_PORT:-<api-port>}/api/v1/driver/containment" | jq
```

**Expected** — macOS reports the Seatbelt backend with exactly the two
guarantees it enforces:

```json
{ "backend": "seatbelt", "available": true,
  "enforcements": ["filesystem-write-containment", "network-deny"] }
```

If `sandbox-exec` is missing, the report degrades honestly to
`{ "backend": "none", "available": false, "enforcements": [] }` — record
that and stop; Step 4 cannot pass without Seatbelt.

## Step 3 — Create a sandbox and confirm the identity layout

Create a sandbox over a scratch git repo (substitute a real path):

```bash
SB=$(curl -s -X POST "http://127.0.0.1:${API_PORT}/api/v1/sandboxes" \
  -H 'content-type: application/json' \
  -d '{"scopePath":"/Users/you/scratch","projectRoot":"/Users/you/scratch","owner":"shakeout","noLock":true}')
echo "$SB" | jq '{id, driverId, workspacePath, pathIllusion, containment}'
```

**Expected** — the copy driver's identity layout:

- `driverId == "copy"`
- `pathIllusion == false`
- `workspacePath == mergedDir` (the reported `workspacePath` equals the
  sandbox's host merged dir; **not** `/workspace`)
- `containment.backend == "none"`, `containment.level == "none"` on the
  *sandbox* response (this is the predicted layout; the *exec* below carries
  the effective Seatbelt backend).

## Step 4 — Prove Seatbelt denies out-of-workspace writes and network

This is the load-bearing step. Exec inside the sandbox and try to escape the
writable set. Use `NetworkAccess=none` (the default `full` profile is *not*
what we are testing).

**4a. Write outside the workspace must be denied.**

```bash
ID=$(echo "$SB" | jq -r .id)
curl -s -X POST "http://127.0.0.1:${API_PORT}/api/v1/sandboxes/${ID}/exec" \
  -H 'content-type: application/json' \
  -d '{"command":"sh","args":["-c","echo pwned > /Users/you/should_not_write.txt"]}' | jq
```

**Expected:** non-zero `exitCode`, and `stderr` naming an OS permission
error (`Operation not permitted` — Seatbelt denies `file-write*` outside the
sandbox writable set). `containment.backend == "seatbelt"` on the exec
response. The file `/Users/you/should_not_write.txt` must **not** exist
afterward.

A write *inside* the workspace must still succeed:

```bash
curl -s -X POST ".../sandboxes/${ID}/exec" -H 'content-type: application/json' \
  -d '{"command":"sh","args":["-c","echo ok > inside.txt && cat inside.txt"]}' | jq
```

Expected `exitCode == 0`, `stdout` = `ok`.

**4b. Network must be denied when the profile disallows it.**

Run a command that opens a socket, with a profile whose `NetworkAccess` is
`none` (do **not** pass `allowNetwork:true`):

```bash
curl -s -X POST ".../sandboxes/${ID}/exec" -H 'content-type: application/json' \
  -d '{"command":"sh","args":["-c","curl -sS -m 5 https://example.com"],"isolationLevel":"full"}' | jq
```

**Expected:** non-zero `exitCode`; `stderr` shows the connection failing
(`Operation not permitted` / `Could not resolve host` / connection refused —
Seatbelt `(deny network*)` blocks the socket). If the same command succeeds,
`network-deny` is **not** actually enforced — a critical fail.

## Step 5 — Approve a diff and confirm provenance

Read the diff, then approve it:

```bash
curl -s ".../sandboxes/${ID}/diff" | jq '{stats, files: [.files[] | {filePath, changeType}]}'
curl -s -X POST ".../sandboxes/${ID}/approve" -H 'content-type: application/json' \
  -d '{"mode":"all","actor":"shakeout","agentManagerRunId":"mac-shakeout-1","createCommit":false}' | jq
```

**Expected:** `diff.stats` reflects `inside.txt` as added; `approve` returns
`{"success":true,"applied":<n>}`; the applied file appears in the canonical
repo (`/Users/you/scratch/inside.txt`).

**Provenance fields to eyeball** — the applied change must carry the run
attribution (query the audit/provenance surface, or inspect the sandbox
`applied_changes`):

- `agentManagerRunId == "mac-shakeout-1"`
- `sandboxId == <ID>`
- `changeType`, `filePath` (project-root-joined), `provenanceState == "applied"`

## Step 6 — Run a real protected agent-manager run (integration)

Drive a protected-mode run through agent-manager targeting this sandbox
(exact invocation per your agent-manager operator flow). The point is to
confirm the SandboxLauncher path works against a live macOS sandbox:

**Expected:**

- The run executes (agent process launched via workspace-sandbox
  `/processes`), and the working dir stays the **host merged path** (identity
  layout — no `/workspace` translation).
- **No** `SANDBOX_NO_EXIT_INFO` on a clean exit.
- Because Seatbelt enforces both protected-mode guarantees
  (`filesystem-write-containment` + `network-deny`), the run timeline must
  **not** contain a containment-degradation warning. (Contrast: the copy
  driver with `sandbox-exec` absent reports `backend=none` and *would* emit
  the degradation warn naming the missing enforcements — see
  agent-manager `emitContainmentGapWarn`.)

## Recording results

Log outcomes with `swarm-manager records create` (or file defects via the
report-bug skill). Capture, for each step: the exact command, the observed
signature, and pass/fail. A green run across Steps 1–6 supplies the Sandbox
portion of real-Mac evidence; it does not by itself qualify all macOS setup or
Bridge capabilities. Update [`PORTABILITY_AUDIT.md`](../internal/PORTABILITY_AUDIT.md)
and the [platform support matrix](../../../../docs/reference/platform-support.md)
with the earned tier.

## Handoff

On a green run, complete the remaining macOS setup and Bridge lifecycle ladder
and record its evidence before registering this Mac as a standing
bridge-reachable runner.
