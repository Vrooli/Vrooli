# Local attach runbook (Linux amd64)

This runbook runs the source-built sherpa-onnx adapter **outside** Vrooli's
managed lifecycle and attaches Audio Tools to it. It is the supported path
until a signed release artifact is published; `vrooli resource start
sherpa-onnx` still reports `unsupported on linux` by design.

Audio Tools accepts this mode explicitly: when the control plane does not own a
resource, the provider health probe is the serviceability truth and the engine
selection reason says `attached: provider health confirmed; lifecycle ownership
is external`.

## 1. Fetch the pinned runtime and models

```bash
mkdir -p /tmp/sherpa-root /tmp/sherpa-data/{asr,punctuation,kokoro}
cd /tmp
curl -sLO https://github.com/k2-fsa/sherpa-onnx/releases/download/v1.13.2/sherpa-onnx-v1.13.2-linux-x64-shared.tar.bz2
tar -xjf sherpa-onnx-v1.13.2-linux-x64-shared.tar.bz2 -C /tmp/sherpa-root --strip-components=1

curl -sLO https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2
tar -xjf sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2 -C /tmp/sherpa-data/asr

curl -sLO https://github.com/k2-fsa/sherpa-onnx/releases/download/punctuation-models/sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12-int8.tar.bz2
tar -xjf sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12-int8.tar.bz2 -C /tmp/sherpa-data/punctuation

curl -sLO https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-int8-multi-lang-v1_0.tar.bz2
tar -xjf kokoro-int8-multi-lang-v1_0.tar.bz2 -C /tmp/sherpa-data/kokoro
```

The 2023 Zipformer is the runtime default: ~310 MB, Apache-2.0 packaging for a
CC-BY-4.0 LibriSpeech-trained model, and the only English streaming checkpoint
in the release that the adapter serves end to end today. The Kroko 2025 model
is declared on demand (68 MB, CC-BY-SA-4.0). Its generic encoder/decoder/joiner
filenames must be configured explicitly, and the stream finalizer now feeds a
bounded 1.2 s silent tail before flushing so final words are not dropped.
Kroko remains experimental pending quality, latency, timestamp, and license
qualification; this smoke fix is not a promotion receipt.

## 2. Build the adapter bundle

```bash
cd /path/to/Vrooli/resources/sherpa-onnx
make bundle-native-linux-amd64 ROOT=/tmp/sherpa-root OUT=/tmp/sherpa-bundle
```

The bundle contains `server/sherpa-onnx-server` and the adjacent `lib/` tree.
It is a local qualification artifact, not a signed release.

## 3. Run it on the resource port

```bash
RESOURCE_LISTEN_ADDR=127.0.0.1:8881 \
SHERPA_ONNX_MODEL_DIR=/tmp/sherpa-data/kokoro/kokoro-int8-multi-lang-v1_0 \
SHERPA_ONNX_STREAMING_MODEL_DIR=/tmp/sherpa-data/asr/sherpa-onnx-streaming-zipformer-en-2023-06-26 \
SHERPA_ONNX_STREAMING_MODEL_TYPE=zipformer2 \
SHERPA_ONNX_PUNCTUATION_MODEL_DIR=/tmp/sherpa-data/punctuation/sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12-int8 \
SHERPA_ONNX_PROVIDER=cpu SHERPA_ONNX_THREADS=2 SHERPA_ONNX_FFMPEG=ffmpeg \
LD_LIBRARY_PATH=/tmp/sherpa-bundle/lib \
/tmp/sherpa-bundle/server/sherpa-onnx-server
```

Confirm readiness:

```bash
curl -s http://127.0.0.1:8881/healthz
# {"status":"ok","encoder_ready":true,"streaming_model_loaded":true,"speaker_model_loaded":false}
```

`/healthz` is the health route. `/ready` exists only when the speaker runtime
is configured.

## 4. Point Audio Tools at it

`scenarios/audio-tools/.vrooli/service.json` already sets
`AUDIO_SHERPA_URL=http://localhost:8881`. Restart the scenario, then check the
engine table:

```bash
vrooli scenario restart audio-tools
audio-tools stt engines
```

Expected: `sherpa-streaming` is `available` and selected with reason
`attached (provider health confirmed, lifecycle external)`. If a previous
operator pin selected Kyutai, clear it once:

```bash
audio-tools stt stream-config-set --engine sherpa-streaming
```

Stream a file:

```bash
audio-tools stt transcribe-stream --file /path/to/16k-or-any.wav
```

## 5. Optional: Kroko and Kale

- **Kroko**: add `--group streaming-zipformer-kroko` to a data acquisition and
  point `SHERPA_ONNX_STREAMING_MODEL_DIR` at it. Set
  `SHERPA_ONNX_STREAMING_ENCODER`, `SHERPA_ONNX_STREAMING_DECODER`, and
  `SHERPA_ONNX_STREAMING_JOINER` to the archive's `encoder.onnx`,
  `decoder.onnx`, and `joiner.onnx` paths. The native smoke test can check the
  JFK final phrase with `SHERPA_ONNX_STREAMING_EXPECT_TEXT='your country'`.
  Do not promote it until corpus quality, latency, timestamp, and licensing
  evidence are complete.
- **Kale TTS**: the adapter serves Kokoro-format ONNX models through
  `SHERPA_ONNX_MODEL_DIR`. When Kale publishes an export with the same front
  end, point this variable at the Kale model directory. Until then, Kale stays
  a candidate in the model registry with no shipped artifact.

## Why attach-only

The resource manifest deliberately claims no platform support until a signed
adapter artifact exists. Attach-only lets an operator qualify the adapter on
their own host without pretending the release boundary is complete. The
provider health probe is still required, so a dead server cannot remain
selectable.
