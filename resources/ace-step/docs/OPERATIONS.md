# ACE-Step Operations

`ace-step` is a `managed-service` resource. The control plane owns the
standalone Python runtime, hash-pinned wheels, source archive, and model data;
the service owns only model loading and one-take generation.

The pinned ACE-Step model uses Transformers remote code. Executable model code
is owned by the composed source at commit
`ca1e85fe9430179831e6bc6be790c332190a3866`, bound by the source archive and
composed-tree digests. Before `trust_remote_code` loading, upstream always
checks checkpoint Python against `acestep/models/{turbo,sft}` and copies missing
or different files locally. This includes SFT's APG shim; the complete source
package on PYTHONPATH provides shared configuration, guidance, and DCW modules.
Checkpoint Python is derived from that source, not separately acquired model
data. Declaring HF copies would cause acquisition to restore older code that
startup immediately overwrites.

Weights, configuration JSON, tokenizers, and planner inputs remain separately
revision/checksum-pinned in the 32 data-artifact declarations. The historical
HF-code review and current source-authority correction are recorded in
`scenarios/music-tools/docs/internal/SECURITY.md`. Moving a source or model pin
requires a fresh review; changing composed inputs also requires re-deriving the
composed-tree digest.

The service runs with `HF_HUB_OFFLINE=1`; it must not fetch code or weights at
startup. `full` keeps the DiT resident on the GPU. `offload-dit` restarts the
managed service with `ACE_STEP_OFFLOAD_DIT=1`. Neither rung evicts another
tenant, and there is no CPU fallback.

The effective startup health budget is `lifecycle.start_timeout_seconds: 600`.
The matching orchestration estimate is informational and does not set the
managed-service driver's timeout.

Use the control plane for lifecycle:

```bash
vrooli resource validate ace-step
vrooli resource start ace-step
resource-ace-step capacity degrade --to offload-dit
resource-ace-step capacity upshift --to full
vrooli resource status ace-step --json
```

## Refreshing the composed server

Changing repository `server/server.py` does not replace an already verified
installed tree. Use the resource acquisition owner to compose a new version;
never edit the installed artifact, invoke a private Python server, or install
packages outside the declared dependency workflow. Preserve the old version for
rollback and follow `docs/resources/authoring-a-resource.md` to record the real
candidate digest and update both declared tree-digest locations.

`vrooli resource install ace-step --reacquire` refuses before discarding bytes
when the supervisor records a running service, including an older version, or
when ownership/state cannot be established. Stop only ACE through its normal
resource lifecycle before an admitted refresh. The command does not stop it
for you. Acquisition also verifies declared model data and can download missing
or altered data; a runtime `HF_HUB_OFFLINE=1` setting does not prohibit installer
downloads. When downloads are outside scope, verify the existing model closure
before admitting acquisition.

Activate the refreshed ACE wrapper together with the corresponding Music Tools
consumer. The consumer requires actual loaded model/rung headers, so restoring
an older ACE artifact may also require restoring the prior Music serving build.
Keep both runtime identities and rollback receipts; do not infer adoption from
source tests alone.


## Fixed model and planner profiles

The resource owns three named selections over already declared pinned inputs:
`fast` (turbo / 0.6B planner), `balanced` (turbo / 1.7B planner), and
`detail` (SFT / 1.7B planner). These labels describe experimental sampling
profiles; they do not certify quality. Larger selections may need different
capacity qualification. Selecting them changes no resource budget and never
evicts another tenant.

```bash
resource-ace-step profiles list --json
resource-ace-step profiles status --json
resource-ace-step profiles activate --profile detail --no-restart --json
```

Selection atomically writes only `ACE_STEP_VARIANT` and `ACE_STEP_PLANNER` to
the resource's `active.env`, consumed by the existing managed-service environment
file owner. With no file, the preserved default is `fast`. Invalid, incomplete,
or unknown pairs fail closed. `--no-restart` persists desired state only.
Without it, activation uses `vrooli resource restart ace-step` and verifies the
loaded pair afterward; an unsuccessful restart or mismatched observation leaves
desired state persisted and reports failure. Status reports unknown loaded
identity when the server is unavailable, initializing, or too old to provide
explicit loaded fields. Source and desired state alone never prove adoption.

Before adopting these commands, compose the updated server and install the
resource CLI through their normal owners. Adopt the Music consumer and ACE
wrapper together: the new consumer requires actual seed and sampling headers,
so an older server is intentionally incompatible. Keep the prior artifact and
serving identity for rollback. Compare profiles with repeated actual effective
seeds and equal prompts; audition the results before claiming an improvement.


## Planner RNG control

The wrapper's process-local generation lock covers request-boundary PyTorch
CPU/CUDA seeding and the complete upstream call, including failures. Busy calls
refuse with503; there is no hidden generation queue. This addresses pinned
upstream's ignored seeds in single-item LM generation without modifying vendor
source. It adds no model/profile/seed field or capacity budget. Metadata alone
does not certify exact kernel determinism or musical quality.

Outputs from1.1.4 can acknowledge the same resolved per-audio seed while
returning different semantic-code hashes. Preserve historical receipts; that
condition does not isolate a shift effect. After adopting a newly composed
wrapper, qualify repeated same-seed code witnesses before interpreting a
controlled shift comparison. Live model work requires its normal admission.


## Complete checkpoint readiness

A declared weight file is only one checkpoint input. Each advertised planner
requires its own model config and complete tokenizer/chat-template files. The
1.7B planner now declares these eight support files at the same pinned upstream
revision as its unchanged weight; all original24 declarations are preserved.
The complete manifest contains32 data artifacts. This repairs the earlier
weight-only1.7B declaration that prevented balanced/detail initialization.

Declaring files does not install them or establish a loaded profile. Use the
normal resource acquisition owner to verify/acquire the complete declared
closure; never copy a smaller planner's config or patch the live checkpoint.
Before activating a newly completed profile, validate its config/tokenizer and
chat template offline with local_files_only=True, without constructing models.
The larger planner uses its own2048-wide model configuration, whereas0.6B is
1024-wide. Actual model readiness and capacity still require a separately
admitted normal activation. No weight or package upgrade is implied.

## Model disk retention

The control-plane model-cache owner retains checkpoints for 30 days after a
confirmed stopped lifetime and preserves explicit pins. Preview with
`vrooli resource acquisition model-cache ace-step`. Its exact generated-files
declaration covers the source-derived checkpoint Python and small runtime
support files; these are not new HF code acquisitions. An unexpected file still
protects the bundle. Normal start/stop/install updates retention history.
The owner never removes exported Music takes or their receipts.
