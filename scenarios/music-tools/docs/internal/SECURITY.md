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

- **Implementation scope is partial.** Composition, styles, durable jobs, pool,
  model resolution and owned storage/cleanup are implemented. The wider declared
  analysis and transformation inventory is not certified by those controls.
- **Integrity and licence checks have focused regressions.** The model owner tests
  unknown-licence refusal, permissive resolution, checksums, disk floors and the
  pinned ACE registry entry. Resource acquisition verifies the executable tree
  and declared model data; the source-authority correction below identifies the
  code actually loaded. These checks are not a full supply-chain security audit.
- **Runtime qualification is separate from source tests.** Actual loaded model/rung
  identity must match the admitted execution contract before inference and remain
  attached to persisted takes. Retain failed admissions and provenance gaps;
  an available HTTP endpoint alone does not qualify generation or musical fit.
- **Host GPU accounting is currently inaccurate** — unclaimed consumers exist on the
  reference host. Admission verdicts are only as good as the ledger. See
  [`PROBLEMS.md`](PROBLEMS.md).

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — retention and privacy
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — licence lanes
- [`../reference/model-registry.md`](../reference/model-registry.md) — per-model licences
- [`PROBLEMS.md`](PROBLEMS.md) — open issues
# ACE-Step remote-code review (2026-09-19)

This is the historical review of the Hugging Face modules at the model
revision below. It does not establish that those modules are the actual
executable authority; the 2026-10-16 correction below identifies the source
code used by initialization.

The qualified source pin is `ACE-Step/ACE-Step-1.5` commit
`ca1e85fe9430179831e6bc6be790c332190a3866`. The model revision is
`ACE-Step/Ace-Step1.5@19671f406d603126926c1b7e2adc169acbcade22`; the separate
planner is `ACE-Step/acestep-5Hz-lm-0.6B@148d8ea0225bdab342ee1ae3a354275ccd60ca80`.
The source archive SHA-256 is
`b464c11f00af00e1e089a09f58a03962f803c080f116fd0517dbe5ec839706b3`.

Threat: Transformers `trust_remote_code` executes publisher-authored Python
from the pinned model tree. I read the two `auto_map` modules at the model
revision: `configuration_acestep_v15.py` and
`modeling_acestep_v15_turbo.py`. Their imports are limited to Transformers,
PyTorch, einops, vector-quantize-pytorch, and standard numerical modules. The
review found no network client, subprocess call, credential/environment read,
dynamic `eval`/`exec`, or filesystem write in the model implementation path.
The source repository contains separate API/download/training utilities that
do use network, subprocess, and filesystem operations; those entrypoints are
not launched by this resource.

Mitigation: the source and model files are commit/revision pinned, every
acquired artifact has a SHA-256 declaration, and the resource invokes the
headless PyTorch generation path only. The resource must not fetch at first
request, and outbound-network denial should be included in live qualification.
Any pin move requires a new remote-code review. A checksum protects against
post-review tampering but never replaces reading the code at the pin.

Licence evidence: the pinned source is MIT. The pinned model card reports MIT
and states that its training data includes licensed and royalty-free/no-
copyright data; that training-data statement remains a vendor assertion and
is not independently audited.

## Executable source authority correction (2026-10-16)

The same pinned source commit performs mandatory local code synchronization
before its `AutoModel.from_pretrained(..., trust_remote_code=True)` call.
`init_service_orchestrator.py` checks weights, synchronizes model Python, then
loads the model; `model_downloader.py` copies all non-`__init__.py` variant
modules from `acestep/models/{turbo,sft}` when checkpoint code is missing or
different. The resource therefore retires the four independently pinned HF
Python data artifacts. All 24 remaining data-artifact pins, source archive,
wheel lock, and composed-tree digest remain unchanged.

Actual executable model code comes from the exact composed source tree. Its
212-byte configuration shims import
`acestep.models.common.configuration_acestep_v15`; both model implementations
import common DCW code, and SFT additionally imports its APG shim and shared
guidance implementation. These dependencies are present in the composed
source package on PYTHONPATH. The model-revision modules inspected in the
historical review differ from these source modules; the historical findings
must not be treated as a complete review of the latter closure.

The current correction verifies the synchronization order and local dependency
presence. It is not a new full security audit of the source closure or a live
network-denial qualification. Source-tree integrity and offline loading remain
required, and future source/model pin changes require reviewing the executable
variant and common-module closure. See `resources/ace-step/docs/OPERATIONS.md`
for the acquisition and startup contract.
