# Decisions — Music Tools

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

Model-level facts that back these decisions — identifiers, disk cost, VRAM,
licences — are in [`../reference/model-registry.md`](../reference/model-registry.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-19 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-19 | **ACE-Step 1.5 is the primary composition model, not MiniMax-Music3.** | MiniMax-Music3 is the stronger model on paper (11.1B, five-minute coherence) but needs ~22–24 GB VRAM and a 57.4 GB download. The reference host has a 16 GB card with ~9.9 GB free and 274 GB disk on a volume at 85%. ACE-Step is MIT, fits the free-VRAM tier, and ships the edit operations music work actually needs. | Composition targets a model that fits. The licence is unrestricted, so no attribution or revenue ceiling reaches the product. Quality ceiling is lower than the best available open model. | Revisit on a card with ≥24 GB VRAM, or if a legitimate quantised MiniMax-Music3 build appears. |
| 2026-08-19 | **Three isolated Python runtimes, two of them managed-service resources.** | The generation, music-information-retrieval, and embedding stacks pin mutually incompatible Python versions, torch builds, and NumPy majors. The shared sidecar provisioner (`pyenv-go`) syncs exactly one virtualenv per scenario. | `ace-step` and `music-mir` own their environments and lifecycles; only the compatible embedding stack runs in the in-scenario sidecar. Cross-runtime coordination is by job queue and filesystem, never shared imports. | Revisit if the upstream stacks converge, or if `pyenv-go` gains multi-environment support. |
| 2026-08-19 | Rejected extending the shared sidecar provisioner to hold multiple environments. | `pyenv-go` is a governed shared package; the isolation needed here is exactly what a separate resource already provides. | No change to a governed package is required to ship this scenario. | Revisit if a third scenario needs the same multi-environment shape. |
| 2026-08-19 | **No container runtime.** Resources acquire lockfile-pinned wheels and checksum-verified weights natively. | Existing GPU resources use compose overlays, but the desktop delivery tier cannot assume a container runtime is present. | Isolation is at the process and environment boundary. Install is slower and more platform-sensitive than pulling an image. | Revisit if native provisioning proves unreliable across target platforms. |
| 2026-08-19 | **Every GPU operation claims through the control-plane capacity broker.** | The card is shared with other Vrooli resources; `vrooli capacity` already provides claims, admission verdicts, degradation rungs, heartbeats, and reconciliation. | No private GPU queue. Operations declare ordered profile rungs so contention degrades quality rather than failing the job. The applied rung travels with the result. | Revisit only if the broker cannot express a needed policy. |
| 2026-08-19 | **Commercial-use lane is a first-class registry property, not a scenario-level assumption.** | Several of the strongest analysis models are non-commercial or share-alike, and at least one useful tool declares no licence at all. This scenario feeds a bundle intended to be sold. | Each model records its lane; a permissive-lane build refuses to resolve anything outside it. Unknown licences default to restricted. | Revisit when a restricted-lane model relicenses, or when commercial distribution begins. |
| 2026-08-19 | **Copyleft tools are invoked as separate processes, never linked in-process.** | Reference mastering's best open implementation is GPL-3.0. | Obligations do not reach the scenario binary. Costs a process hop on an operation that is already file-to-file. | Revisit if a permissively licensed equivalent reaches parity. |
| 2026-08-19 | **No entrypoint materialises stems for an entire library.** | Four stems for a 10,000-track library is roughly 1 TB even as FLAC, against 274 GB free. | Separation is on-demand with an LRU budget. The absence of a batch path is a design constraint, not a missing feature. | Revisit only alongside a storage tier sized for it. |
| 2026-08-19 | **Embeddings persist pooled or segment-level; frame-level requires explicit single-track opt-in.** | Frame-level output for one four-minute track is roughly four orders of magnitude larger than its pooled form. | The library-wide index stays small. Frame-level analysis is an interactive operation, never a batch product. | Revisit if a use case needs frame-level data at library scale. |
| 2026-08-19 | **Personal adapter training is a P2 target, not a shipping feature.** | Published guidance for the chosen generator states 16 GB minimum and roughly 17 GB typical during training. The reference card has 16 GB total and ~9.9 GB free. | Personalisation ships as a lightweight head over frozen embeddings, owned by the consumer scenario. Adapter training is documented but gated on hardware. | Revisit on a card with ≥24 GB VRAM, or if a materially cheaper adapter method is verified. |
| 2026-08-19 | **A dedicated separator is a hard dependency, not a quality upgrade.** | The generator's own documentation is internally inconsistent about which model variants support track separation, and the variant that fits the free-VRAM tier is the one most likely to lack it. | Stem separation is owned by `music-mir` and never assumed available from the composition runtime. | Revisit when upstream resolves the contradiction; the dependency stays regardless because it is also the better separator. |
| 2026-08-19 | **Lyric transcription lives here, not in `audio-tools`.** | General speech recognition performs poorly on sung vocals; the useful models are music-specific. | A music-specific transcription operation exists here. No general speech capability is added. | Revisit if `audio-tools` gains a singing-aware engine. |
| 2026-08-19 | **Tempo and key are estimated by vote across independent estimators, and disagreement is retained.** | Three independent estimators are available at no extra cost. Disagreement correlates with rubato, ambiguous tonality, and half/double-time ambiguity. | Estimates carry a confidence signal, and disagreement itself becomes an attribute consumers can use. | Revisit if a single estimator is shown to dominate. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../reference/model-registry.md`](../reference/model-registry.md) — the evidence behind model choices
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
