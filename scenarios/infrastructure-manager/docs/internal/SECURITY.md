# Security — Infrastructure Manager

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

**No personal, customer, financial, or regulated data exists in this scenario,
and none should ever be introduced.** Everything persisted is a measurement of
local infrastructure. The sensitivities that do exist are operational rather
than personal.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Readings | low–moderate (operational) | readings | Numeric or enumerated observations of local infrastructure — uptime, restart counts, capacity claims, storage growth. Collectively they describe how the operator's host behaves, which is why they stay in local embedded SQLite and are never transmitted. |
| Trust verdicts | low | readings | Enumerated values from a closed vocabulary. No free text. |
| Findings | moderate | focus | May quote an upstream source's error text verbatim, which could incidentally contain a path or hostname. Findings carry a *stated reason*, not raw output — they are not a general log sink. |
| Efficacy records | low | focus | Finding id, sensor reference, expected and observed band result. |
| Setpoint | low | not stored | Read from the team's plan of record, which is repository content. |

**The strongest control here is architectural, not a mitigation.** Every input
is read through a typed CLI or Connect surface rather than by scraping logs or
process output, so there is no path by which a credential could enter the
reading store. Preserving that is an extension rule, not a preference — see
[`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) § Extension Rules.

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain carries placeholder data only
(removed by `template-manager detemplate`):

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None by default | n/a | no | Add entries when resources or third-party APIs require secrets. |

## Threat Model

The interesting threats here are **integrity** threats, not confidentiality
ones. This scenario's output steers where engineering effort goes, so the
question that matters is "could the board be made to lie?" rather than "could
the board leak?"

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| **Instrument grades itself** — a change lets the scenario author or cache a deadband | The board could report in band by lowering its own bar, indistinguishable from real improvement. This is the highest-impact risk in the scenario. | The setpoint is a checked-in file with **no API write path at all**, parsed at query time and never stored; `coverage` owns no tables. Prevented by construction rather than by policy — there is no endpoint to misuse and no permission to get wrong. Reinforced as an extension rule and as an experience claim (`no-edit-affordance`). | mitigated by design |
| **Cached roster drift** — a change caches the derived supervised set | Membership silently freezes; scenarios added later are never reported unsupervised. Also recreates the roster operating-model rule 6 forbids. | `supervision` owns no tables; the set is derived per read and degrades to `UNAVAILABLE` rather than falling back to a cached list. | mitigated by design |
| **Instrument fault read as plant fault** | Real engineering effort spent on an imaginary problem while the alarm channel that would show the real one stays saturated. Measured precedent: 4,624 critical events / 24h, ~92% ghost and saturated. | Trust verdicts on every reading; untrusted readings excluded from aggregates and routed to the instrument. See [`../concepts/TRUST-MODEL.md`](../concepts/TRUST-MODEL.md). | mitigated by design; two of four integrity rules blocked on roadmap Gap 10 |
| **Actuation creep** — a future change adds a "restart" or "shelve" affordance | The instrument becomes a controller; the team both sets reliability targets and performs the recovery, grading its own homework. | No mutating verb on any dependency. Enforced as extension rule 6 and as journey claim `no-action-affordance`. | mitigated by design |
| **Error text captured verbatim into findings** | An upstream error could incidentally carry a path or hostname into stored findings. | Findings carry a stated reason, not raw output; all reads go through typed surfaces rather than log scraping. | accepted, low |
| Missing auth for product data | Not applicable — the scenario stores no protected or multi-user data and exposes read-only surfaces. | Revisit only if the scenario ever gains a write surface beyond its own findings. | not-applicable |
| Unsafe file upload handling | Inherited from the worked example only; removed with `template-manager detemplate`. | BlobStore seam isolates bytes in the example. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Two of four trust-integrity rules (ghost, shelved) cannot be computed | medium — untrusted readings are conservatively marked rather than assumed valid, so the gap degrades safely, but the board's blindness is wider than it will be | Roadmap Gap 10 ships `check reconcile` and `check shelve`. |
| No mechanical enforcement that extension rules 3, 4 and 6 hold | medium — they are the mitigations for the three highest-impact risks above, and they are currently prose | Add architecture tests asserting `coverage` owns no schema, that `condition` caches no leg population, that no `setpoint` write path exists on any handler, and that no dependency client exposes a mutating verb. Should land with the first vertical slice. |
| `secrets-manager` exposes no typed read surface | low for this scenario; blocks a target row | That scenario ships a `cli/manifest.json` and a health verb. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
