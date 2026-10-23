# pydeps — image-tools Python dependency lock

This package is the **single version source of truth** for the image-tools Python
backends. It embeds two files so they travel with the compiled binary:

| File | Role |
|------|------|
| `requirements.in` | The **governed, ranged** inputs. App-compatibility ceilings (e.g. `transformers<5`) live here, *not* in the shared Scenario Dependency Analyzer governance ranges. |
| `requirements.lock` | The **fully pinned + hashed** set, generated from `requirements.in`. The uv venv (`github.com/vrooli/pyenv-go`) is synced from exactly this. |

At boot, `pydeps.Materialize` writes the lock into the scenario data dir and
`github.com/vrooli/pyenv-go` builds/repairs a private venv from it with `uv`. The Python
backends invoke that venv's interpreter by absolute path, so their heavy deps
(torch / diffusers / transformers / onnxruntime) are isolated from any other
Python on the host.

## Regenerating the lock

Never edit `requirements.lock` by hand or run a package manager directly. Use
Scenario Dependency Analyzer's governed entry point; it follows the API's
compatibility `requirements.txt` forwarder to this canonical input/lock pair:

```bash
scenario-dependency-analyzer deps install pip/<name>@<version> \
  --scenario image-tools --surface api --json
# Inspect the preview, then repeat with --apply when the dependency is approved.
```

The owner preserves unrelated ranged inputs and lock pins, resolves wheel-only
artifacts, and stages both files before publication. Publication errors restore
the previous pair; an interrupted process still requires conformance verification.

Then run the conformance tests, which keep the lock honest against both
`requirements.in` and the SDA governance store:

```bash
go test ./internal/pydeps/
```

## Governance

Every **direct** dependency in `requirements.in` must be approved in the Scenario
Dependency Analyzer (governed, never hand-edited):

```bash
scenario-dependency-analyzer deps approved approve pip/<name> --range ">=X" --rationale "..." --apply
```

Transitive pins (the CUDA `nvidia-*` libraries, `triton`, etc.) are covered by
their direct parents' approval and are not individually governed. The
`TestLockMatchesSDAGovernance` test asserts the direct-dep ⇄ governance contract
when the store is present.

## Scope

The lock intentionally covers only the **proven, runnable** Python backends:

- **diffusers** generative/edit stack — instruct-pix2pix, qwen-image-edit,
  sd-1.5-inpainting;
- **ONNX Runtime** inference stack — the CPU-floor onnx ops.

The `basicsr`/`facexlib` RestoreFormer++ face-restore path is excluded: its
execution adapter is gated (see `internal/sidecar/py/image_tools_sidecar/restore.py`)
and its old stack would pin torch/torchvision down for the whole environment. Add
and prove it when that vertical lands.
