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

### 2026-08-17 — `vrooli-events` never fans out webhook subscriptions

**Symptom:** A durable webhook subscription in `vrooli-events` targeting
this scenario is stored, reports healthy, and never fires. Event-driven
notifications (OT-P1-003) cannot work today no matter what is built
here.

**Root cause:** On ingest, `vrooli-events` publishes to its SSE broker
and nothing else. `subscription.WebhookDeliverer.Deliver` is reachable
only from a manual "trigger this subscription" handler. There is no
matcher, no retry queue, and no delivery engine. The `vrooli-events`
README and its `docs/guides/creating-subscriptions.md` both describe
this scenario as the primary consumer of a path that does not run.

**Workaround:** Build the ingress receiver here and test it against a
synthetic caller so it is ready. Use direct API and CLI requests as the
primary ingress, which is why they are P0 and event ingress is P1.

**Real fix:** A subscription fan-out engine in `vrooli-events`. That
work belongs upstream and must be filed against that scenario rather
than absorbed here.

**Owner:** unassigned — needs a bug filed against `vrooli-events`.

**Refs:** `scenarios/vrooli-events/api/handlers.go` (ingest publishes to
broker only), `scenarios/vrooli-events/api/handlers_subscription.go`
(manual deliver handler), `scenarios/vrooli-events/internal/subscription/`
(no engine).

### 2026-08-17 — Generated `PROGRESS.md` inherited the template's own history

**Symptom:** The freshly generated scenario's `docs/internal/PROGRESS.md`
contained two 2026-07-07 entries describing work on the `react-vite`
template itself, directly contradicting the same file's statement that
"this file ships empty in newly generated scenarios".

**Root cause:** The template's `docs/internal/PROGRESS.md` is copied
verbatim rather than being emptied during generation.

**Workaround:** Deleted here by hand and replaced with this scenario's
real log.

**Real fix:** The generator should truncate the progress table when
copying the template. Affects every scenario generated from `react-vite`
1.6.5, not just this one.

**Owner:** unassigned — belongs to `template-manager`.

**Refs:** `templates/scenarios/react-vite/docs/internal/PROGRESS.md`.

### 2026-08-17 — `docs/manifest.json` requires headings that `detemplate` will delete

**Symptom:** The generated `docs/manifest.json` requires the heading
`"Notes (CRUD reference)"` in `docs/reference/api-endpoints.md`, which
the generated document does not contain — it uses
`## Domain endpoints — <domain>` with the notes material under a fenced
`### Example domain` subheading. The same pattern appears in the
`cli-commands.md` contract, which requires
`"Scenario commands — \`notes\` (CRUD reference)"`.

**Root cause:** The manifest's required headings were written against an
older shape of the reference docs, and they name the *example domain*.
Because `template-manager detemplate` removes every example-domain
block, any scenario that completes the `example-domain-removed`
orientation gate would then permanently fail its own documentation
contract.

**Workaround:** Changed the `api-endpoints.md` requirement here to
`"Domain endpoints"`, which is present now and survives `detemplate`.

**Real fix:** The template should not require example-domain headings in
any document contract. Affects every scenario generated from
`react-vite` 1.6.5.

**Owner:** unassigned — belongs to `template-manager`.

**Refs:** `templates/scenarios/react-vite/docs/manifest.json`,
`docs/reference/api-endpoints.md`, `docs/reference/cli-commands.md`.

### 2026-08-17 — No end-to-end proof that a notification reached a human

**Symptom:** The system can prove it handed a payload to a provider. It
cannot prove the phone showed it. Every automated signal can be green
while the owner sees nothing.

**Root cause:** Push delivery is fire-and-forget by construction. The
provider acknowledging receipt is not the device rendering a
notification.

**Workaround:** A manual real-device gate in the release checklist, and
an explicit statement in `../operations/OBSERVABILITY.md` that the
scenario measures dispatch rather than receipt. This is the single most
important thing to stay honest about, because the predecessor scenario
reported healthy for ten months while delivering nothing.

**Real fix:** Acknowledgement (OT-P2-004) closes it partially — a human
acting on a notification proves it arrived. Nothing closes it fully.

**Owner:** unassigned.

**Refs:** `docs/operations/DEPLOYMENT.md` release checklist,
`docs/operations/OBSERVABILITY.md` telemetry gaps.

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
