# Security — Music Library

Sensitivity, trust boundaries, and the controls that hold them.

## Purpose Of This Document

Use this document to answer:

- How sensitive is the data this scenario accumulates?
- What protects a listener's own files?
- What is the integrity boundary, and why is it a security concern?

## Data Sensitivity

**Listening history is among the most revealing behavioural data a person
produces.** It exposes daily routine, sleep pattern, mood, emotional state, and
personal association far beyond musical preference. Volume and timing alone
reconstruct a life.

| Data | Sensitivity | Notes |
|---|---|---|
| Interaction events (play, skip, replay, position, time) | **Very high** | Behavioural; reconstructs routine and mood |
| Preference model and profiles | **Very high** | A compact model of the listener |
| Comparison history | High | Explicit judgments |
| Listener-authored constraints | High | Stated preference, often personal |
| Track index and attributes | Moderate | Reveals library contents |
| Source audio | **High, and not ours** | Read-only; belongs to the listener |
| Generated audio | Moderate | Inherits the profile that conditioned it |

The governing decision: **behavioural data never leaves the host.** There is no
telemetry path for it — not anonymised, not aggregated, not opt-in. The absence of
an egress path is the control; a policy that could be reconfigured would not be.

## Auth And Authorization

Single-listener application on an operator-controlled host. Identity is inherited
from the scenario runtime rather than reimplemented.

Where a shared or bundled deployment introduces multiple listeners, profiles,
comparison history, and events are per-listener and must not be readable across
listeners. That is a hard requirement whenever more than one profile exists —
cross-listener leakage here is not a preference bug, it is disclosure of exactly
the data named above.

## Secrets

This scenario holds no credentials of its own. If a deployment adds remote access,
it uses the platform's consumer identity and credential authority rather than a
scenario-local mechanism.

The preference model itself deserves handling closer to a secret than to
application state: it is small, portable, and highly revealing. Exports are a
listener-initiated action, never a background one.

## Threat Model

| Threat | Vector | Mitigation |
|---|---|---|
| Damage to a listener's music | Any write to source files | **Source audio is opened read-only.** The scenario has no code path that writes to a source root |
| Behavioural data exfiltration | Telemetry, crash reports, analytics | No egress path exists for events, profiles, or comparisons |
| Cross-listener disclosure | Shared deployment | Per-listener scoping of profiles, events, and comparisons |
| Ranking corrupted by commercial interest | `ranking` or `generation` observing what is sold | **Blindness boundary enforced at package level**, not by convention; `offers` sits strictly downstream of a final ranking and cannot reorder or filter it |
| Undisclosed synthetic content | Generated audio presented as owned | Provenance recorded on every candidate; disclosure in the interface and in exported metadata |
| Silent history loss | Files moved or reorganised | Content-derived identity; a missing track is marked, never deleted |
| Profile corruption by bad signal | Noisy implicit feedback, position bias | Section attribution is reliability-gated and requires repetition; raw history is retained so any refit is reversible |
| Coordinate-space mismatch | Upstream embedding model changes | Profiles record the embedding model they were fitted against; a change forces explicit refit rather than silent misreading |

The blindness boundary is listed here rather than only in the monetisation docs
because it is an **integrity control**. The thing being protected is the listener's
justified belief that a recommendation reflects their taste and nothing else. That
belief is the product.

## Security Gaps

Known and accepted for now:

- **No implementation exists**, so every control above is a design commitment rather
  than a shipped control.
- **The blindness boundary has no enforcing test.** Until a test fails when
  `ranking` can import `offers`, the guarantee rests on reviewer memory. This is the
  single most important test in the scenario.
- **Read-only source access needs enforcement**, not just intent — a single
  incorrect open flag would violate it silently.
- **Multi-listener scoping is unspecified** because the first deployment is
  single-listener. It must be specified before any bundled or shared deployment.

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — retention, deletion, privacy notes
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — the blindness boundary
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — why blindness is required
- [`PROBLEMS.md`](PROBLEMS.md) — open issues
