# Local attach runbook (Linux amd64)

This runbook runs the source-built sherpa-onnx adapter **outside** Vrooli's
managed lifecycle and attaches Audio Tools to it. It is the supported path
until a signed release artifact is published; `vrooli resource start
sherpa-onnx` still reports `unsupported on linux` by design.

Audio Tools accepts this mode explicitly: when the control plane does not own a
resource, the provider health probe is the serviceability truth and the engine
selection reason says `attached: provider health confirmed; lifecycle ownership
is external`.

## 1. Fetch the pinned runtime and selected model data

```bash
mkdir -p /tmp/sherpa-root /tmp/sherpa-data/{asr,punctuation,kokoro}
cd /tmp
curl -sLO https://github.com/k2-fsa/sherpa-onnx/releases/download/v1.13.2/sherpa-onnx-v1.13.2-linux-x64-shared.tar.bz2
tar -xjf sherpa-onnx-v1.13.2-linux-x64-shared.tar.bz2 -C /tmp/sherpa-root --strip-components=1

vrooli resource acquisition data sherpa-onnx --group streaming-zipformer-en-2023-06-26
```

The 2023 Zipformer is the runtime default: ~310 MB, Apache-2.0 packaging for a
CC-BY-4.0 LibriSpeech-trained model, and the only English streaming checkpoint
in the release that the adapter serves end to end today. The Kroko 2025 model
is declared on demand (68 MB, CC-BY-SA-4.0). Its generic encoder/decoder/joiner
filenames must be configured explicitly, and the stream finalizer now feeds a
bounded 1.2 s silent tail before flushing so final words are not dropped.
Kroko remains experimental pending quality, latency, timestamp, and license
qualification; this smoke fix is not a promotion receipt.

The 20M English Zipformer also matches the online recognizer contract. Acquire
it as a separate on-demand group with
`vrooli resource acquisition data sherpa-onnx --group streaming-zipformer-en-20m-2023-02-17`.
The archive SHA-256 is
`9c559283e8498d3fe95913c79ca1cb454bb26281ac2b102b41306c7d752765d9`. Set
`SHERPA_ONNX_STREAMING_MODEL_DIR` to its extracted directory and
`SHERPA_ONNX_STREAMING_MODEL_TYPE=auto` to use it; the adapter automatically
chooses the int8 encoder and joiner and fp32 decoder. In the native smoke test
on `quality_speech.wav`, this checkpoint emitted only `S.` (load 0.82 s,
streaming inference 30 ms for 1.96 s audio). The upstream CLI produced the same
single token on the same fixture. This confirms model load, path selection,
and end-of-stream output for this fixture; it also records a poor lexical
result, with no general quality claim. The 310 MB model remains the default.

## 2. Build the adapter bundle

```bash
cd /path/to/Vrooli/resources/sherpa-onnx
make bundle-native-linux-amd64 ROOT=/tmp/sherpa-root OUT=/tmp/sherpa-bundle
```

The bundle contains `server/sherpa-onnx-server` and the adjacent `lib/` tree.
It is a local qualification artifact, not a signed release.

## 3. Run the streaming-only server

```bash
RESOURCE_LISTEN_ADDR=127.0.0.1:8881 \
RESOURCE_DATA_DIR="$HOME/.local/share/vrooli/resources/sherpa-onnx" \
SHERPA_ONNX_TTS_ENABLED=false \
SHERPA_ONNX_SPEAKER_ENABLED=false \
SHERPA_ONNX_PROVIDER=cpu SHERPA_ONNX_THREADS=2 SHERPA_ONNX_FFMPEG=ffmpeg \
LD_LIBRARY_PATH=/tmp/sherpa-bundle/lib \
/tmp/sherpa-bundle/server/sherpa-onnx-server
```

Confirm readiness:

```bash
curl -s http://127.0.0.1:8881/healthz
# {"status":"ok","tts_model_loaded":false,"encoder_ready":true,"streaming_model_loaded":true,"speaker_model_loaded":false}
```

`/healthz/stt` checks only streaming STT. `/healthz/tts` checks the optional
Kokoro route, including its output encoder. The general `/healthz` is ready when
at least one model-backed capability is loaded.

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

## 5. Optional: Kokoro TTS, speaker verification, Kroko, and Kale

- **Kokoro TTS**: acquire `vrooli resource acquisition data sherpa-onnx --group kokoro-multi-lang-v1-0`, then run with `SHERPA_ONNX_TTS_ENABLED=true`. This model is optional; Audio Tools' normal Kale/Kokoro selection uses its separate local provider resources.
- **Speaker verification**: acquire `vrooli resource acquisition data sherpa-onnx --group speaker-verification`, then run with the profile directory configured. The resource default loads this small active capability; the streaming-only attach command above disables it to keep that temporary test path minimal.

- **Kroko**: add `--group streaming-zipformer-kroko` to a data acquisition and
  point `SHERPA_ONNX_STREAMING_MODEL_DIR` at the extracted directory. The
  adapter detects its generic `encoder.onnx`, `decoder.onnx`, and `joiner.onnx`
  files automatically; explicit `SHERPA_ONNX_STREAMING_*` file paths still
  override discovery when needed. The native smoke test can check the JFK final
  phrase with `SHERPA_ONNX_STREAMING_EXPECT_TEXT='your country'` and its
  actual final words with
  `SHERPA_ONNX_STREAMING_EXPECT_FINAL_SUFFIX='your country.'`. The final suffix
  assertion catches streams that contain the phrase earlier but truncate at
  end-of-input. Do not promote it until corpus quality, latency, timestamp, and
  licensing evidence are complete.
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
