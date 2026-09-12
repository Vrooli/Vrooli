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

