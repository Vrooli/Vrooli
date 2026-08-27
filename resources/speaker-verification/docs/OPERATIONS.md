# Operations

`speaker-verification` is organized as a native `managed-service` resource. The
control plane composes a checksum-pinned CPython runtime, a hash-locked CPU or
CUDA wheel set, and the FastAPI server around SpeechBrain's ECAPA-TDNN model.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, acquisition, port, export, storage,
  and health metadata.
- `server/` owns the Python HTTP server and VAD source. The two lock files at the
  resource root own the CPU and CUDA wheel closures.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Speaker Verification-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. Keep native lifecycle
decisions in the manifest and shared managed-service control plane.

## Server Contract (load-bearing)

The HTTP contract in `server/server.py` is consumed byte-for-byte by the
audio-tools Go client
(`scenarios/audio-tools/api/internal/stt/pipeline/speaker_client.go`). Endpoint
paths, multipart field names (`profile_id`, `display_name`, `notes`,
`threshold`, `verify`, and the `audio` file field), and response JSON keys MUST
NOT drift. See [docs/API.md](API.md) and [docs/internal/SEAMS.md](internal/SEAMS.md).

## GPU vs CPU

GPU is selected when the NVIDIA facts satisfy the manifest predicate. The
control plane then chooses the CUDA wheel lock and sets
`SPEAKER_VERIFICATION_DEVICE=cuda`; no Docker runtime, overlay, or device
reservation is involved. ECAPA verification is small, but SepFormer
target-extraction is heavy enough that the GPU matters for interactive latency.

The CPU target uses a separate CPU wheel lock and reports its selected device by
`/v1/info` (`device`, `torch_version`, `cuda_available`). The GPU target also
falls back safely if CUDA is unavailable, but the selected artifact and its
device predicate remain explicit in the acquisition record.

## Operator Checklist

- Keep ports, storage, native acquisition, and health checks declared in `resource.json`.
- Keep enrolled profiles and the model cache in the declared resource data directory.
- The first start downloads the ECAPA-TDNN weights into the regenerable model-cache directory. Budget time for this warmup.
- Keep both wheel locks hash-pinned and update them through the dependency governance workflow.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.

## Live Validation

Native acquisition and model download require network access:

```bash
vrooli resource install speaker-verification
resource-speaker-verification status
cd cli && RESOURCE_LIVE_TEST=1 go test ./... -run TestSpeakerVerificationLive
```
