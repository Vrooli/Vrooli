# Operations and qualification

Use normal control-plane resource lifecycle. Do not run uvicorn, Python service
files, a virtualenv installer or package managers directly to start/repair this
resource. Root owns dependency approval/acquisition, artifact composition,
capacity admission and native pilot qualification.

## Reproducible inputs

Source77e0398c26f7d22d801ec1bb036307e8b231f741 is MIT. The public source archive
SHA256 is2cd8dcc690766f8112a8aac8ea3c99cd35fa6690928a49588afb285718837a3f.
Medium model revisiond4d3579395a8ac0df4ab13b5b1d7138ce7290f81 uses Stability
Community and redistributed Gemma terms. The acquisition target holds only a typed
reference to `vrooli/stable-audio-3:huggingface-token`; inference receives no token.
The descriptor is optional for offline runtime; a needed gated acquisition still
fails closed when its explicit target reference cannot resolve an authorized token.
Accept the HF gate and applicable commercial obligations before acquisition.

The12-file model/support/notice set is declared in both resource data artifacts and
`server/model-lock.json`; tests require those declarations to agree. Do not pad tiny
JSON, copy a different checkpoint's configuration or use unpinned HF-cache contents.
The combined checkpoint includes its decoder; no separate SAME-L file is needed.
Startup verifies sizes/SHA256s, fixes only the copied effective config's local text
encoder path, forces offline loading and rejects incomplete/unqualified inputs.
The authenticated reviewed model config has sample_size16,777,216 at44.1kHz,
T5Gemma max_length256 with learned padding and rf_denoiser objective. It omits
sampling_distribution_shift_options, so the pinned upstream constructs a
LogSNRShift(rate=0, anchor_logsnr=-6.2, logsnr_end=2). Receipts capture the actual
loaded object rather than misreporting the separate training-distribution shift.

`requirements.in` is the authored input. SDA owns creation/update of a complete,
hash-pinned, wheel-only `requirements.lock`, with CUDA12.6 and PyPI indices declared
in resource.json. Transformers5.10.4 is the reviewed, unyanked patch above upstream's
5.8 floor; the earlier5.10.0 pin was publisher-yanked and is superseded. Native T5
compatibility remains to be qualified. FA2 uses the exact
official Dao cp312/Torch2.7/CUDA12 ABI-TRUE wheel. No source-distribution build or
post-install mutation is part of this profile.

The application artifact includes source, standalone CPython, wheels, server,
loader, model lock and the generated dependency lock. Normal repeat composition
must establish the artifact digest before root sets both manifest digest fields
and `managed_service.environment.SA3_COMPOSED_SHA256`. That last field is a declared
launch identity in receipts, not an independently recomputed tree digest. Never
copy it into a file inside the artifact and create a self-referential tree hash.
Bytecode/caches/outputs stay outside the immutable executable tree.

## Fixed initial capacity profile

The qualification profile requests **estimated9GiB VRAM** through the existing
broker and declares24GiB RAM. Capacity admission remains advisory under the current
policy; the declared receipt field is not proof of a live ledger reservation.
The first native trial exposed startup lease expiry; the shared renewal repair
still requires deployed lifecycle qualification. Before model allocation, the service applies
Torch's supported **8GiB caching-allocator ceiling**, using actual GPU total memory
to compute its fraction. The1GiB difference is estimated non-Torch/context headroom;
it is not a measured bound. CPU FP32 load→FP16 conversion→CUDA avoids a transient
FP32 main-model GPU copy. The unregistered T5Gemma encoder moves separately on first
use in its upstream precision. The qualified Medium A/B receipts observed the
conditioner as `torch.bfloat16` and the main model as `torch.float16`; the encoder
is not an FP32 GPU allocation in those runs. Decoder chunking is always enabled; Torch compile
is explicitly disabled through the supported environment before imports.

There is one service worker and one nonblocking generation lock. No CPU fallback,
automatic precision change, lower-quality rung, ceiling bump or other-tenant
mutation occurs on allocation/admission failure. Root may normally stop ACE under
the user's authority; this does not authorize stopping whisper/Ollama or prove fit.
The canonical broker profile has one estimated `medium` rung, with preferred and
floor both9GiB. `capacity degrade --to medium` uses the existing lifecycle adapter
to restart this resource with that same fixed recipe; unsupported rungs are refused.
There is no smaller-memory rung or automatic upshift. The required idle-yield opt-in
lets existing broker policy consider normal owner stop when an idle floor claim
must be reclaimed; the resource does not carry a private stop/preemption mechanism.

Observe startup separately from warmed generation. Receipts expose actual CUDA
allocated/reserved peaks and process lifetime maxrss, with whole-scope RAM and
non-Torch peak marked unknown. Root's existing owned-scope GPU/RAM observations
must accompany qualification; H200 vendor numbers are not host measurements.
The requirement disk estimate must cover retained models, runtime/wheels, download
staging and repeat-compose copies. Transient output retention is resource-owned;
export reviewed takes before normal cache/storage cleanup. No private sweeper or
unbounded background render queue is introduced.

## Source checks and native limits

From this directory, `make check` runs Go checks and no-model Python regressions.
The Python cohort needs existing FastAPI/Pydantic/httpx/numpy, or the governed test
environment; Torch/upstream/encoder are fakes. It does not load real weights or CUDA.
`PYTHONDONTWRITEBYTECODE=1` protects staged runtime invocations during qualification.

Native acceptance still requires the normal validated artifact install/start,
real ABI/FA/T5 load, two serialized48-second requests, source decode/frame/hash
checks, owned cancellation/drain evidence and bounded capacity/cleanup receipts.
Use existing `music-tools delivery arrange` for matched8..20-second excerpts.
Neither source checks nor valid audio establish operator preference or no vocals.


## Capacity liveness and stop authority

The shared control-plane start/restart admission lease renews the exact declared
claim while verification and CPU-first initialization run under the authored
operation context. It makes a final renewal on normal completion. The existing
activity-edge companion immediately renews admitted residency independently of
active/idle request reporting, then uses the canonical default TTL/3 cadence
(normally10s) and default heartbeat TTL (normally30s). This companion heartbeat
uses the default TTL rather than the resource's initial admission TTL. Renewal never creates a replacement claim or resizes the
reservation, and broker errors remain observable and fail open for requests.

A verified explicit managed-service stop releases only the same-owner claim IDs
captured under the lifecycle lock. A failed/canceled readiness wait can leave
the supervised model running and therefore does not release its reservation.
An old edge exiting during restart cannot release the replacement's reused claim.
On unexpected parent death the edge ceases renewal; canonical GPU observation and
bounded TTL expiry reconcile the claim. This is eventual expiry, not an immediate
instance-bound parent-death release. Admission is still advisory; no global policy
or capacity fit guarantee is established by this repair.
