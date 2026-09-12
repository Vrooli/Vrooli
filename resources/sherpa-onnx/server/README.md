# sherpa-onnx native audio server prototype

This directory contains the resource-local HTTP adapter required to keep
sherpa-onnx cgo out of scenarios and the control plane. It is deliberately
tagged `sherpa_onnx`; the same implementation is selected for cgo builds on
Linux, macOS, and Windows, while non-cgo builds fail closed.

The native engine is built against the official sherpa-onnx C API. A Linux
qualification build uses the extracted upstream runtime like this:

```bash
CGO_CFLAGS="-I$SHERPA_ROOT/include" \
CGO_LDFLAGS="-L$SHERPA_ROOT/lib" \
GOWORK=off go test -tags sherpa_onnx ./...
```

To produce the deployable tree on a matching native host, use the resource
bundle target instead of copying the executable by hand:

```bash
ROOT=/path/to/extracted/sherpa-onnx-v1.13.2-linux-x64-shared \
OUT=/tmp/sherpa-bundle-linux-amd64 \
make -C .. bundle-native-linux-amd64
```

The resulting `server/sherpa-onnx-server` uses the adjacent `lib/` tree via a
relative loader path. The release boundary must checksum and sign that whole
tree, including every shared runtime library.

The qualification inputs were immutable release assets: sherpa-onnx
`v1.13.2` Linux x64 shared runtime SHA-256
`1ef6741535f7af4d69e394fd440a807108036d26ed4f542660191019da5c0daa`, and
Kokoro v1.0 int8 model archive SHA-256
`75654a84864be26f345f020f4070c2c019e96dd1b7f9bf6e2ffd59efac6aa5a3`.
The same release also publishes full macOS arm64 and Windows x64 shared
runtimes (respectively
`50c5c04d93113602432a13454d6bf8e5d2624206b985fbd0dd4698454ae6c509` and
`55f31de3977e8d80178cbcf1a70c673ce46c64e7ee5725a3ff30e6df2e26e126`), but
this checkout has not yet produced or signed the corresponding Vrooli server
artifacts.

The server exposes the existing `/v1/audio/voices` and
`/v1/audio/speech` routes and preserves the legacy Kokoro v1.0 voice IDs. It
encodes the native 24 kHz float PCM as WAV directly and uses the configured
`SHERPA_ONNX_FFMPEG` executable for MP3, Opus, and FLAC. `ffmpeg` is therefore
a required host tool for the full response-format contract; the resource
fails closed for compressed output when it is unavailable. Signed per-platform
server artifacts are still required before this native resource can be
promoted on additional targets.

## Streaming STT

The same adapter exposes `/v1/stream` as a bounded WebSocket endpoint. Clients
send a text `start` control message, canonical mono 16 kHz signed-16 PCM binary
frames, then a text `end` message. The server returns `ready`, `partial`,
`segment`, `processed`, and `done` frames using the stable Vrooli stream
contract. The Zipformer model is English-only and is configured through
`SHERPA_ONNX_STREAMING_MODEL_DIR`; its checksum-pinned data artifact is declared
in the parent resource manifest. Final segments pass through the separately
declared sherpa punctuation model, which restores punctuation and casing.
