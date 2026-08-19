# Problems — Notification Hub

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-17 — Template scaffold still documented as if it were the product

**Status: RESOLVED 2026-08-18.** The reference set was rewritten against the
five-domain implementation and each registered document is now tracked at
`active` maturity.

**Symptom:** `SEAMS.md`, `TESTING.md`, `ARCHITECTURE.md`, and the
`reference/` documents still contain generic scaffold prose. `DOMAINS.md`,
`DATA.md`, `FLOWS.md`, `SECURITY.md`, and `INTEGRATIONS.md` were rewritten
against the real domain map on 2026-08-18 and no longer describe the removed
example domain.
A reader who trusts the old generic documents would have concluded this
scenario was still the generated example app.

**Root cause:** the generated documentation set was not fully rewritten when
the real notification domains first landed.

**Resolution:** Rewrote the remaining reference and internal documents
against the actual domain map and raised their manifest maturity only after
the documentation-health checks passed.

**Owner:** codex.

**Refs:** `docs/manifest.json`, `api/internal/hub/`, `cli/domains/`.

### 2026-08-17 — OT-P0-001 has a missing prerequisite: no push provider resource exists

**Status: RESOLVED 2026-08-18 — the prerequisite was removed, not met.**

OT-P0-001 now delivers through Web Push from this scenario's own installed
progressive web app. No push resource exists and none is needed. The
rejected relay design is preserved at
`.vrooli/resources/blueprints/ntfy.json` at `status: candidate`. See the
2026-08-18 decision rows in `DECISIONS.md`.

### 2026-08-17 — Event ingress blocked on an upstream gap in vrooli-events

**Status: RESOLVED 2026-08-18.** `vrooli-events` now enqueues matching
webhook subscriptions and drains them with retry/signature/health updates;
notification-hub also reconciles its optional subscription at startup and
accepts the actual nested webhook payload shape.

The former upstream gap is retained only as historical context. The active
contract is documented in `concepts/INTEGRATIONS.md` and covered by
`internal/integrations/events_test.go`.

### 2026-08-18 — Three confirmed defects in vrooli-events block event ingress

**Status: RESOLVED 2026-08-18.** The upstream queue, matcher, retry,
signature, and health-writer paths are now present and covered by that
scenario's fan-out tests.

The former fan-out, retry/signature, and health-writer defects were fixed in
`vrooli-events`. This entry remains as an audit trail; notification-hub's
current receiver and startup reconciliation are covered by
`internal/integrations/events_test.go`.

### 2026-08-18 — vrooli-bridge needs a per-scenario change to carry a new verb

**Status: RESOLVED 2026-08-18.** Bridge dispatch vocabulary is now derived
from the shared scope catalog. Notification-hub relays through its own
cataloged `notification-hub notifications relay` command and resolves a
machine's current node lineage through MachineService.

The former per-scenario bridge allowlist was replaced by catalog-derived,
run-eligible vocabulary and paired effect/verb grants. Notification-hub now
dispatches the typed `notification-hub notifications relay` command and
resolves the target machine's current node lineage before dispatch.

**Refs:** `scenarios/vrooli-bridge/api/internal/dispatch/allowlist.go`,
`packages/api-core/scopecatalog/catalog.go`,
`api/internal/hub/bridge.go`.

### 2026-08-18 — Escalation can cross a sensitivity tier

**Status: RESOLVED 2026-08-18.** Each escalation step re-evaluates channel
approval and emits a content-free pointer when the next channel is not
approved for the notification label.

**Symptom:** An unanswered critical ask escalates to the next channel in
the recipient's chain. That channel may hold a lower sensitivity approval
than the channel the first attempt used.

**Root cause:** Design gap found while writing `SECURITY.md`. The
sensitivity policy is evaluated once at routing time, and escalation
reuses the resulting decision rather than re-evaluating per step.

The escalation worker now re-evaluates approval for each step and falls back
to a content-free pointer when the next channel is not approved.

**Refs:** `docs/internal/SECURITY.md`, `api/internal/hub/escalation.go`,
OT-P0-010, OT-P1-011.

### 2026-08-18 — External push acceptance environment is not provisioned

**Status: OPEN — operator/platform action required.**

**Symptom:** The implementation and local suites pass, but the stable
`notification-hub.itsagitime.com` origin does not resolve and no operator
iPhone or paired Mac evidence is available for the required real-device
acceptance.

**Root cause:** tunnel-manager has no Cloudflare account ID, tunnel ID, or API
token configured in this host environment; the remaining acceptance steps
also require direct operator actions on iOS and macOS.

**Workaround:** Provision the three tunnel-manager credentials, reconcile the
managed core route, then install the PWA from the stable origin and record the
push body, timestamp, endpoint prefix, and paired-Mac delivery evidence.

**Real fix:** Complete the operator-owned acceptance for OT-P0-001 and
OT-P0-015 without changing the stable-origin design.

**Owner:** operator/platform.

**Refs:** `OT-P0-001`, `OT-P0-015`, `docs/concepts/INTEGRATIONS.md`,
`tunnel-manager config credentials-status --json`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
