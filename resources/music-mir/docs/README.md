# Music MIR local API

`music-mir` is a private local adapter called by Music Tools. It never fetches
audio. Music Tools retains the source and passes its canonical path, SHA-256,
the owning blob root, and a request-scoped staging directory. The resource
rejects paths outside that root, symlinks, source digest mismatches, nonempty
staging directories, and inputs larger than 1 GiB.

## Endpoints

- `GET /health` reports service/model status and a one-second, loopback-only Whisper readiness probe.
- `GET /v1/capabilities` reports available analysis/separation features and whether optional Whisper transcription is configured and reachable.
- `POST /v1/analyze` returns a partial acoustic analysis and source-separation
  energy curves.
- `POST /v1/separate` writes four 24-bit WAV stems (`vocals`, `drums`, `bass`,
  `other`) in the request staging directory and returns each file's digest.
- `POST /v1/cancel` requests cancellation and acknowledges only after active
  inference and cleanup have drained.

Analysis and separation accept a JSON object with `request_id`, `allowed_root`,
`input: {path, sha256}`, and `output_dir`. The output directory must be
`<allowed_root>/analysis/.staging/<request_id>`. Music Tools validates and
promotes completed outputs into durable owned blobs; this resource does not
publish, relocate, or delete source audio.

## Runtime guarantees and limits

The selected SubRoformer adapter requires CUDA and refuses CPU fallback. The
Music Tools job owns the capacity lease; the resource unloads the model and
releases cached VRAM before its response returns. Cancellation does not claim
that an in-flight model call was interrupted: Music Tools must retain its job
and capacity claim until `/v1/cancel` returns `drained: true`.

Inputs are decoded to local 48 kHz stereo PCM in staging, limited to seven
minutes; decoding is hard-capped at one second beyond that limit before the
size check rejects longer files. This bounds temporary growth even for highly
compressed long audio. The temporary decode is removed after the operation.
Each emitted stem is limited to 128 MiB. Analysis returns tempo and beat
estimates, key candidates with a margin, energy curves, low-confidence acoustic
cut candidates, and best-effort timestamped transcript segments through the
configurable local Whisper service. Only loopback HTTP URLs are accepted so
private reference audio cannot egress to remote services. Whisper segments are hypotheses with no
word-level alignment or confidence value; service absence is recorded without
failing the other analysis. The resource does not identify semantic
sections/hooks or produce plain-language captions. Those gaps prevent this
increment from qualifying as the MQ1 showcase.

The health and capability reports distinguish Whisper configuration from
reachability. An unavailable optional companion appears as `configured: true`,
`ready: false`, `status: "unavailable"`; it does not make MIR analysis fail.

Model weights are acquired on demand from the pinned manifest in
`server/model-lock.json`. The selected checkpoint's license is not stated by
its model card; it is restricted here to local personal analysis. No commercial
use is enabled by this resource configuration.
