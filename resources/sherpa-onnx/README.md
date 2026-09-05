# sherpa-onnx native resource

This resource owns the Vrooli HTTP adapter for sherpa-onnx Kokoro TTS,
streaming Zipformer STT, speaker embeddings, and Spleeter source separation.
cgo is confined to `server/`; scenarios and the control plane consume the
HTTP/WebSocket contracts only.

The upstream sherpa runtime and Kokoro model are checksum-pinned in
`resource.json`. The model archive is acquired as `tar.bz2` data and staged
below `RESOURCE_DATA_DIR`; it is never treated as the executable launch
artifact.

The streaming STT route is `/v1/stream` and accepts canonical 16 kHz mono
signed-16 PCM. The English Zipformer archive is separately checksum-pinned in
`resource.json`, as is the punctuation model used for final segments, so TTS,
STT, speaker verification, separation, and text post-processing data can be
refreshed independently. Speaker profiles persist the active model name and
embedding dimension; changing either returns `speaker_model_mismatch` and
requires explicit re-enrollment.

The current checkout contains a reproducible Linux amd64 adapter build path. The
macOS and Windows files sometimes found under `server/` are development/stub
outputs, not native release artifacts, and are deliberately excluded from
platform evidence. Every managed-service acquisition target and deployment
profile is explicitly unsupported until release publication supplies a real
native adapter artifact. Upstream runtime availability is not proof that the
Vrooli HTTP adapter is portable or lifecycle-owned.

Run `make check` for the resource-local gates. A native qualification requires
`ROOT=/path/to/extracted/sherpa-onnx-v1.13.2-linux-x64-shared make native-test`
and a live run with `RESOURCE_LIVE_TEST=1` after the signed server artifact and
model have been installed.

The release artifact must be built on its target platform so cgo cannot
silently produce a foreign executable. For example:

```bash
ROOT=/path/to/extracted/sherpa-onnx-v1.13.2-linux-x64-shared \
OUT=/tmp/sherpa-bundle-linux-amd64 \
make bundle-native-linux-amd64
```

The output is a self-contained managed-service tree with
`server/sherpa-onnx-server` and the matching shared libraries under `lib/`.
The tree, rather than only the executable, must be checksum-pinned and signed
before it can be enabled as an acquisition target.

The reproducible release boundary is available through the resource Makefile:

```bash
make release-stage \
  TARGET=linux-amd64 \
  ROOT=/path/to/extracted/sherpa-onnx-v1.13.2-linux-x64-shared \
  OUT=/secure/release-stage/sherpa \
  PUBLICATION_URL=https://artifacts.example.invalid/vrooli/sherpa-onnx
```

`release-stage` refuses a foreign host, rejects a non-empty stage, bundles the
server with its adjacent runtime libraries, computes the deterministic
`binaryfetch.TreeDigest`-compatible digest, writes `release-manifest.json`,
creates a target archive plus `publication-target.json`, and signs the release
manifest with the managed release authority by default. Tar and gzip metadata
are normalized, so identical target trees produce identical archive checksums.
The publication metadata carries the archive checksum, signed tree checksum,
layout, and both the canonical `entry_path` and compatibility `bin_path`.
`PUBLICATION_URL` is optional for local staging; when supplied it must be an
HTTPS base URL and is written into the handoff metadata. Use `SIGN=0` only for
a local inspection stage; an unsigned stage is not deployable and does not
justify changing the manifest's unsupported target claims. A local stage or
archive is not publication evidence.
