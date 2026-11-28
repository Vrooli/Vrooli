# sherpa-onnx native resource

This resource owns the Vrooli HTTP adapter for optional sherpa-onnx Kokoro TTS,
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

The default managed profile loads the 310 MB Zipformer streaming model, its
punctuation model, and the 27 MB speaker-embedding model. The larger Spleeter
source-separation model is lazy-loaded on the first `/v1/extract` request;
until then readiness reports `speaker_model_loaded: true` and
`separation_model_loaded: false`. Kokoro TTS is an on-demand group.
`SHERPA_ONNX_TTS_ENABLED` defaults to `false`, so a streaming install does not
also load the larger Kokoro model. The resource still exposes the TTS route
when its Kokoro data is explicitly acquired and TTS is enabled.

The upstream English 20M Zipformer (2023-02-17) is runtime-compatible as an
explicit `SHERPA_ONNX_STREAMING_MODEL_DIR` override. The adapter discovers its
int8 encoder and joiner plus fp32 decoder automatically. Set
`SHERPA_ONNX_STREAMING_MODEL_TYPE=auto` for this older checkpoint; the managed
310 MB profile continues to use `zipformer2`. It is not a default, and is
available as the separately pinned, on-demand group
`streaming-zipformer-en-20m-2023-02-17`. Its 127,887,156-byte upstream archive
is pinned to SHA-256
`9c559283e8498d3fe95913c79ca1cb454bb26281ac2b102b41306c7d752765d9`. Acquire
it explicitly with `vrooli resource acquisition data sherpa-onnx --group streaming-zipformer-en-20m-2023-02-17`.
The existing 310 MB model and managed default remain unchanged; this artifact
has no quality-promotion claim.

The current checkout contains a reproducible Linux amd64 adapter build path. The
macOS and Windows files sometimes found under `server/` are development/stub
outputs, not native release artifacts, and are deliberately excluded from
platform evidence. Every managed-service acquisition target and deployment
profile is explicitly unsupported until release publication supplies a real
native adapter artifact. Upstream runtime availability is not proof that the
Vrooli HTTP adapter is portable or lifecycle-owned.

Run `make check` for the resource-local gates.

For an immediately runnable Linux amd64 setup that does not claim the managed
release boundary, see [docs/LOCAL-ATTACH.md](docs/LOCAL-ATTACH.md). It builds
the adapter from the pinned upstream runtime, runs it on port 8881, and shows
how Audio Tools attaches to it. The 2023 English Zipformer is the streaming
default; the smaller Kroko 2025 model remains on demand pending quality,
latency, timestamp, and license qualification. Its finalizer adds a bounded
silent tail so the stream decoder retains its final words.
 A native qualification requires
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
  PUBLICATION_URL="$SHERPA_RELEASE_PUBLICATION_URL" \
  UPSTREAM_PROVENANCE="$SHERPA_RELEASE_UPSTREAM_PROVENANCE"
```

Before using this signed path, set `SHERPA_RELEASE_PUBLICATION_URL` and
`SHERPA_RELEASE_UPSTREAM_PROVENANCE` to values supplied and verified by the
release owner. The publication URL must be an HTTPS base URL. Provenance must
identify the verified upstream runtime/source and Vrooli adapter source used
for this target; do not substitute a placeholder or a locally inferred claim.
The release authority status command reports whether the local signing key is
configured, but it does not provide these owner decisions.

For local build inspection before release-owner details are available, prepare
an unsigned stage explicitly:

```bash
make release-stage \
  TARGET=linux-amd64 \
  ROOT=/path/to/extracted/sherpa-onnx-v1.13.2-linux-x64-shared \
  OUT=/tmp/sherpa-stage \
  SIGN=0
make validate-stage OUT=/tmp/sherpa-stage
```

An unsigned stage is local evidence only. It cannot enable managed acquisition
or change the resource's unsupported platform claims.

`release-stage` refuses a foreign host, rejects a non-empty stage, bundles the
server with its adjacent runtime libraries, computes the deterministic
`binaryfetch.TreeDigest`-compatible digest, writes `release-manifest.json`,
creates a target archive plus `publication-target.json`, and signs the release
manifest with the managed release authority by default. Tar and gzip metadata
are normalized, so identical target trees produce identical archive checksums.
The publication metadata carries the archive checksum, signed tree checksum,
layout, and both the canonical `entry_path` and compatibility `bin_path`.
`PUBLICATION_URL` and `UPSTREAM_PROVENANCE` are required when `SIGN=1`; both
must come from the release owner. The former must be an HTTPS base URL. This
prevents a signed release from carrying placeholder publication details or an
invented upstream provenance claim. For local inspection, use `SIGN=0`; an
unsigned stage is not deployable and does not
justify changing the manifest's unsupported target claims. A local stage or
archive is not publication evidence.

Before signing, staging runs an offline validator against `resource.json`. It
checks the declared bundle name, role, target, layout, entry path, tree digest,
archive digest, and archive contents. Existing platform digests in
`sha256_by_platform` must match the staged tree. To recheck a retained stage,
run `make validate-stage OUT=/path/to/stage`; this does not publish or enable
managed acquisition.
