# Authoring a Resource

This guide is the shortest path from a new `resource.json` to a resource that
can be acquired, started, and reviewed across platforms.

## Choose the driver and acquisition kind

| Question | Choose | Use it when |
|---|---|---|
| Who owns the process? | `managed-service` | Vrooli supervises a long-running local service. |
|  | `cli` | The resource exposes an installed command. |
|  | `native-external` | An operating-system tool is discovered on `PATH`. |
|  | `composed` | Several declared inputs form one resource. |
| How does the artifact arrive? | `url` | A checksum-pinned archive or file is published. |
|  | `oci-image` | A pinned container filesystem is the portable upstream artifact. |
|  | `npm` | A pinned package is the executable source. |
|  | `composed` | The resource is assembled from declared child inputs. |
|  | `none` | A reviewed host executable is intentionally adopted. |

Choose `bundling` separately from acquisition:

| Bundling | Meaning |
|---|---|
| `vendorable` | The build machine can resolve every target from `os` and `arch`. |
| `host-required` | First run must resolve host facts or discover a host tool. |
| `prohibited` | The artifact must never be placed in a bundle. |

## Write targets

Targets are evaluated in order. Put the most-specific target first, followed by
a portable fallback or an explicit `unsupported` target. A runtime fact is
available on the buyer machine, not on the build machine.

```json
{
  "kind": "oci-image",
  "targets": [
    {
      "when": {"os": "linux", "arch": "amd64", "accel.backends": "has:vulkan"},
      "image": "ghcr.io/example/tool@sha256:<digest>",
      "artifact_sha256": "<sha256 of the executable tree>"
    },
    {
      "when": {"os": "linux", "arch": "amd64"},
      "url": "https://example.com/tool-linux-amd64.tar.gz",
      "sha256": "<sha256 of downloaded bytes>",
      "archive": "tar.gz",
      "layout": "dir",
      "bin_path": "tool/tool"
    }
  ]
}
```

The acquisition target digest authenticates downloaded bytes. The
`artifact_sha256` authenticates the extracted tree that will execute. They are
different claims and must not be copied interchangeably.

Supported predicate facts are `os`, `arch`, `accel.backends`,
`accel.backend`, `accel.cuda_compute`, `accel.vram_bytes`, `accel.vendor`, and
`gpu.cuda_compute`. Values support equality, numeric `>`, `>=`, `<`, `<=`,
`has:<comma-separated-members>`, and a leading `!` for negation. Unknown fact
names fail validation.

## Composed acquisitions must be reproducible

A `composed` acquisition pins the **output** tree with `artifact_sha256`. That
pin is only as stable as the inputs that build the tree, so a composed resource
carries an obligation that a `url` or `oci-image` one does not: the compose must
produce the same bytes twice.

Two rules make that true, and `vrooli resource validate` enforces them:

| Rule | Why |
|---|---|
| The `python-wheels` lockfile must contain `--hash=sha256:` entries | The installer runs `--no-deps --require-hashes`, so the lock must be the complete resolved closure, not a list of direct requirements. An unhashed lock resolves differently over time and the pin becomes unreachable. |
| An accelerated resource must declare `index_url` | `torch==2.5.1` resolves to a CPU build or a CUDA build depending on the index. Both satisfy the version; only one uses the GPU. |

Author the input list in `requirements.in` and generate the lock:

```bash
uv pip compile resources/<name>/requirements.in \
  --python-version 3.12 --python-platform x86_64-unknown-linux-gnu \
  --generate-hashes --index-strategy unsafe-best-match \
  --index-url https://download.pytorch.org/whl/cu124 \
  --extra-index-url https://pypi.org/simple \
  --output-file resources/<name>/requirements.lock
```

Pin an accelerated build by its local version (`torch==2.5.1+cu124`) in
`requirements.in`. Without it, compile-time and install-time resolution can
disagree and `--require-hashes` fails on a wheel the lock never saw.

**Changing either file changes the composed tree**, so re-derive the digest:

```bash
vrooli resource install <name> --reacquire   # fails, printing got/want
# set artifact_sha256 to the `got` value, then re-run to confirm it verifies
```

### The digest is declared twice

`managed_service.artifact.sha256_by_platform` and each target's
`artifact_sha256` both state a digest. **Launch verification reads the target
digest**; the platform map is a fallback. Updating one and not the other leaves
a stale claim that silently does nothing, so validation requires the platform
digest to match some target that can resolve on that platform.

### When a staged artifact stops verifying

`vrooli resource status` distinguishes three causes, because they need three
different actions:

| Cause | Meaning | Action |
|---|---|---|
| `declaration_moved` | The bytes are exactly what was staged; the manifest changed. | Re-acquire. |
| `bytes_changed` | Something wrote into the artifact store after installation. | Re-acquire, then find the writer. |
| `unwitnessed` | No staging witness was recorded, so the two cannot be separated. | Re-acquire; the next install records one. |

The witness is `observed_sha256` in the `.vrooli-install-facts.json` sidecar
beside the staged artifact: the digest the bytes actually produced at staging
time, as opposed to the digest the manifest claimed. A record that only echoes
the declaration cannot witness anything.

A launched service must not write into its own pinned tree. For a Python
runtime that means `PYTHONDONTWRITEBYTECODE=1` in the launch environment — and
note that this protects only the launch path, so any other invocation of the
staged interpreter can still invalidate the tree.

## Verify before review

Run the resolver explanation and inspect every candidate:

```bash
vrooli resource acquisition explain <name>
vrooli resource validate
```

The explanation must select the intended target with the host facts you expect.
Then confirm the resource health check and artifact entry path on each claimed
platform. A platform claim without a resolvable target is incomplete.

## Storage and evidence

Keep models and user data in declared resource storage, not in the executable
tree. Record upstream provenance, artifact digests, platform evidence, and a
review date beside the manifest. A successful cross-compile is `build_verified`;
only a health-passing run on real hardware promotes a platform to `qualified`.


## Gated model metadata and download credentials

For a gated URL target, declare `auth` beside `url`/`sha256`:

```json
"auth": {"scheme": "bearer", "logical_id": "vrooli/example", "field": "huggingface-token"}
```

The same identity/field must appear in the managed resource's `credentials.descriptors`.
An `env` injection target is unnecessary for acquisition. Obtain legitimate model
access first, then use `vrooli credentials provision --identity vrooli/example
--field huggingface-token` in the secure operator terminal; values never belong in
manifest URLs, argv or logs. The installer resolves the credential only for actual
network acquisition. Verified cached bytes need no credential re-read. The bearer
is attached only to the exact HTTPS origin and stripped on cross-origin redirects.
Missing/unavailable authority refuses the download without an HTTP request.

Managed data files use checksum-required, non-executable validation, allowing
legitimate tiny JSON/tokenizer metadata without reducing the binary size floor.
When upstream exposes a pinned Git blob identity before a byte SHA-256 is known,
use the bounded metadata discovery operation (not a model-weight downloader):

```sh
vrooli resource acquisition inspect-data example --url HTTPS_METADATA_URL \
  --git-blob-sha1 EXPECTED_40_HEX --max-bytes 1048576 --out NEW_METADATA_FILE \
  --logical-id vrooli/example --field huggingface-token
```

This emits SHA-256, Git blob identity, byte count and safe origin as JSON, and
refuses existing destinations. Keep the receipt, review the small file and declare
its SHA-256 before normal install. This does not admit execution, grant model
access, select a license lane or claim the data's semantic correctness. The CLI
bounds waiting to two minutes; mandatory manifest SHA-256 pins remain unchanged.
The acquisition schema is generated by the binaryfetch owner command documented
in `packages/binaryfetch/README.md`.

## Deriving an initial composed output digest

A new composed artifact has no output digest until its pinned inputs are staged.
Use the preparation owner rather than a fake checksum or a private package install:

```sh
vrooli resource acquisition compose-candidate example --out /new/candidate-one
vrooli resource acquisition compose-candidate example --out /new/candidate-two
```

The Linux-only operation first reads the authoritative manifest through a rooted,
nonblocking regular-file guard with caller cancellation and a 1 MiB limit. It parses
those exact bytes through the shared manifest owner, then snapshots selected regular local
sources and complete SDA-owned hashed wheel lock, then calls the same managed
compose path as normal acquisition. UV stages wheels with `--require-hashes`,
`--no-deps` and `--only-binary`; no source distributions or target execution are
admitted. URL inputs remain checksum-pinned. Only absent composed output digests
are permitted in this preparation loader; malformed supplied digests and other
manifest/input defects remain errors. Waiting is bounded to 15 minutes and caller
cancellation is propagated.

Each new directory contains `artifact/`, retained `inputs/` and `receipt.json`.
The receipt is outside the hashed artifact and reports candidate-only status,
actual tree SHA-256, platform/selected target and input byte hashes. Existing or
concurrently created outputs are never replaced. Source drift and failures
publish no candidate. No install facts, model data acquisition, capacity claim,
service start or runtime verification is performed.

Compare the two actual artifact identities. Only when they agree, review/write
the real target `artifact_sha256` and matching platform digest, then run ordinary
validate/install and runtime qualification. A candidate receipt does not authorize
execution or certify runtime compatibility. Preserve differing candidates and
resolve reproducibility failures instead of choosing a convenient digest.

Port reservations use a separate typed Name/Ports projection: valid declared ports
remain reserved even when a resource's runtime pins or profile are unfinished.
Invalid port JSON/names/ranges fail allocation explicitly. The first positive port
retains the legacy bare resource key; additional reservations use `resource/port`.
This projection grants no install/start authority and never skips malformed
reservation data.
