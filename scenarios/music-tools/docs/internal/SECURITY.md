# Security — Music Tools

Sensitivity, trust boundaries, and the controls that hold them.

## Purpose Of This Document

Use this document to answer:

- What data crosses this scenario, and how sensitive is it?
- What is the supply chain, and how is it verified?
- What can go wrong that is not a bug?

## Data Sensitivity

| Data | Sensitivity | Notes |
|---|---|---|
| Caller-supplied audio | **High** | May be private recordings, unreleased work, or personal material |
| Captions and lyrics | **High** | Free text; may contain personal or identifying content |
| Generated audio | Moderate | Derived, but inherits the sensitivity of its prompt |
| Model weights | Low | Public artifacts |
| Registry and job state | Low | Operational metadata |

This scenario never sees listening behaviour, ratings, or taste — those stay in the
consumer. It sees only what it is explicitly asked to process.

**Nothing leaves the host by default.** Every model runs locally. The only outbound
paths are model acquisition from declared sources and optional BYOK caption
assistance the operator configures deliberately. If that provider is not
configured, there is no egress path for caller audio or captions at all.

## Auth And Authorization

This is a capability primitive on a single-operator host. It does not implement its
own identity; it inherits the scenario runtime's boundary and trusts its callers.

The authorization decisions it *does* own are resource decisions, not identity ones:

- **Capacity admission.** Every GPU operation claims through the control-plane
  broker. An operation that cannot claim does not run. This is the control that
  stops one caller starving every other tenant on the card.
- **Licence lane resolution.** A permissive-lane build refuses to resolve a
  restricted model. This is enforced at resolution, not at the call site, so a
  caller cannot opt into a restricted model by asking differently.
- **Disk floor.** A model install refuses to start below the declared free-disk
  floor, so no caller can fill the volume by requesting a large model.

## Secrets

This scenario holds **no secrets of its own**. Optional BYOK provider credentials
are resolved through the platform's credential authority, never read from scenario
config or written to scenario state.

Model source URLs and checksums are public and live in the registry seed. They are
integrity data, not secrets, and belong in version control.

## Threat Model

| Threat | Vector | Mitigation |
|---|---|---|
| Tampered model weights | Compromised or substituted download | Checksums declared in the registry and verified on install; an install that fails verification leaves the model unavailable rather than running unverified |
| Licence violation in shipped output | A restricted-lane model used in commercially distributed work | Lane is a registry property; permissive builds refuse to resolve outside the lane; unknown licences default to restricted |
| Copyleft obligations reaching the binary | Linking a GPL tool in-process | Invoked as a separate process, never linked |
| GPU starvation | An operation holding the card indefinitely | Exclusive leases with heartbeats and release-on-failure through the broker |
| Disk exhaustion | Unbounded derived artifacts or model installs | Declared budgets with LRU eviction; install disk floor; no library-wide materialisation path |
| Prompt-carried content risk | Captions or lyrics containing harmful or infringing material | Provenance is recorded on every artifact so downstream disclosure and takedown are possible; this scenario does not adjudicate content |
| Silent quality degradation | Contention degrades output without the caller knowing | The applied profile rung travels with every result |
| Path traversal via caller-supplied paths | Malicious input or output paths | Outputs go through the BlobStore seam with ownership metadata rather than caller-specified filesystem paths |

## Security Gaps

Known and accepted for now:

- **No implementation exists**, so none of the controls above are enforced by code
  yet. Every row in the threat table is a design commitment, not a shipped control.
- **The registry seed's checksums must actually be populated.** A seed entry with an
  empty checksum silently disables the integrity control it appears to provide.
  Treat an unverified checksum as a blocking defect, not a placeholder.
- **The lane boundary needs a test**, not just a convention. A permissive build that
  can resolve a restricted model is a compliance failure that no review will catch
  reliably.
- **Host GPU accounting is currently inaccurate** — unclaimed consumers exist on the
  reference host. Admission verdicts are only as good as the ledger. See
  [`PROBLEMS.md`](PROBLEMS.md).

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — retention and privacy
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — licence lanes
- [`../reference/model-registry.md`](../reference/model-registry.md) — per-model licences
- [`PROBLEMS.md`](PROBLEMS.md) — open issues
