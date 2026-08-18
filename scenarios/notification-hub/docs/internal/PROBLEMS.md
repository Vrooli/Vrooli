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

**Symptom:** `DOMAINS.md`, `DATA.md`, `SECURITY.md`, `SEAMS.md`, and
`TESTING.md` describe a `notes` domain, and `DATA.md` still carries
`_(your data)_` placeholder rows. A reader who trusts those documents
will conclude this scenario is a CRUD notes app with no auth model.

**Root cause:** `template-manager detemplate notification-hub` has not
run, because no real domain exists to replace the worked example yet.
The PRD and the requirements registry are scenario-specific; the rest of
`docs/` is generic template text.

**Workaround:** Treat `PRD.md`, `requirements/`, `DECISIONS.md`, and
`INTEGRATIONS.md` as the authoritative charter. Everything else under
`docs/` is scaffold until the first real domain lands.

**Real fix:** Build the first real domain, run `template-manager
detemplate notification-hub`, then rewrite the concept docs against the
actual domain map. Raise each document's `maturity` in
`docs/manifest.json` only as it earns it.

**Owner:** unassigned.

**Refs:** `docs/manifest.json`, `api/internal/notes/`, `cli/domains/notes/`.

### 2026-08-17 — OT-P0-001 has a missing prerequisite: no push provider resource exists

**Symptom:** OT-P0-001 requires a real delivery to the owner's iPhone,
and the PRD names "a new `ntfy` resource" as its dependency. No such
resource exists under `resources/`, and no `ntfy` blueprint exists under
`.vrooli/resources/blueprints/`.

**Root cause:** The provider decision was made before the resource was
scaffolded. "Deliver to the iPhone" reads as one task and is two.

**Workaround:** None. The channel adapter cannot be credentialed until
the resource exists, because credential descriptors live in resource
manifests by governance rule.

**Real fix:** Scaffold `ntfy` as a `cloud-api` resource via
`template-manager resource-template generate cloud-api`, following the
`twilio` manifest shape. Alternatively promote the existing `pushover`
blueprint (`status: candidate`) instead and revise the PRD.

**Owner:** unassigned.

**Refs:** `resources/twilio/resource.json`,
`.vrooli/resources/blueprints/pushover.json`,
`path:docs/resources/resource-templates.md`.

### 2026-08-17 — Event ingress blocked on an upstream gap in vrooli-events

**Symptom:** `vrooli-events` documents this scenario as its primary
consumer and publishes a webhook subscription contract, but a
subscription never fires on its own.

**Root cause:** On ingest, `vrooli-events` publishes to the SSE broker
only. `WebhookDeliverer.Deliver` is reachable solely from a manual
"trigger this subscription" endpoint. There is no matcher, no retry
queue, and no delivery engine.

**Workaround:** Use direct Connect-RPC and CLI ingress, which is the P0
path and self-contained.

**Real fix:** Belongs to `vrooli-events`, not here. File it there.
OT-P1-003 stays P1 until it lands.

**Owner:** unassigned — upstream.

**Refs:** `scenarios/vrooli-events/api/handlers.go`,
`scenarios/vrooli-events/internal/subscription/webhook.go`.

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
