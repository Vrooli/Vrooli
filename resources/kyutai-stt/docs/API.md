# Kyutai STT API Reference

Authoritative API documentation for the Kyutai streaming speech-to-text
service. This contract is **stable**: the audio-tools scenario depends on it.
Breaking changes require a major-version bump of the server.

## Base URL

```
http://localhost:8094
```

The host port is `8094` (container port `8000`). It is exported to scenarios as
`KYUTAI_STT_URL` / `KYUTAI_STT_BASE_URL`, and the WebSocket URL as
`KYUTAI_STT_WS_URL`.

## Model & Hardware

| Property | Value |
|---|---|
| Default model | `kyutai/stt-1b-en_fr` (~1B params) |
| Languages | English + French |
| Streaming delay | ~0.5 s |
| Native model sample rate | 24 kHz mono |
| Contract sample rate (input) | **16 kHz mono PCM s16le** (server resamples to 24 kHz internally) |
| Frame rate | 12.5 Hz (80 ms per model frame) |
| Device | CUDA GPU required for real-time streaming |

### VRAM footprint

The 1B model loads in bf16. Expected GPU memory:

- Weights (Mimi codec + LM, bf16): **~2–3 GB**
- Resident with streaming buffers / CUDA context: **~3–4 GB**

This fits the local **RTX 4070 Ti SUPER (16 GB)** alongside the ~8 GB already
used by other resources. The larger `kyutai/stt-2.6b-en` model (English only,
~2.5 s delay) needs roughly **6–8 GB** and is selectable via
`KYUTAI_STT_HF_REPO` but is **not** the default because the VRAM budget is
tighter.

## HTTP Endpoints

### Health

**GET** `/health`

```json
{ "status": "ok", "model_loaded": true, "device": "cuda" }
```

- `model_loaded` is `false` while weights are still downloading/loading on first
  start; the server answers `/health` with HTTP 200 throughout so the container
  is considered live, then flips `model_loaded` to `true` once ready.
- `device` is `cuda` or `cpu`.

### Info

**GET** `/v1/info`

```json
{
  "backend": "kyutai",
  "model": "kyutai/stt-1b-en_fr",
  "device": "cuda",
  "sample_rate": 16000,
  "version": "0.1.0"
}
```

`sample_rate` is the **input** sample rate the contract accepts (16000). The
server resamples to the model's native rate internally.

## WebSocket Streaming — `WS /v1/stream`

The streaming endpoint. One streaming session per connection; the server
serializes sessions on the shared GPU model instance.

### Client → server

1. A **TEXT** frame with a JSON start control message:

   ```json
   { "type": "start", "sample_rate": 16000, "language": "en" }
   ```

   - `sample_rate` must be `16000` (the canonical contract rate). Any other
     value yields an `error` frame and the connection closes.
   - `language` may be `"en"`, `"fr"`, or `""` (empty = auto / model default).

2. One or more **BINARY** frames, each carrying raw **little-endian 16-bit PCM,
   mono, 16 kHz** audio samples. Frames may be any length; the server buffers
   and frames internally.

3. A **TEXT** frame to finalize:

   ```json
   { "type": "end" }
   ```

### Server → client

Every server message is a **TEXT** frame containing a JSON object with a
`"type"` field:

| Frame | Shape | Meaning |
|---|---|---|
| partial | `{"type":"partial","text":"<interim>"}` | Optional interim hypothesis for the in-progress utterance. Emitted as tokens arrive. |
| segment | `{"type":"segment","text":"<committed>","start_ms":<int>,"end_ms":<int>}` | One finalized utterance/segment. `start_ms`/`end_ms` are millisecond offsets from the start of the stream. |
| done | `{"type":"done"}` | Sent after the server receives `end` and flushes all buffered audio. |
| error | `{"type":"error","message":"<reason>"}` | A protocol or runtime error. The connection then closes. |

### Frame ordering guarantees

- `partial` frames (if any) precede the `segment` they refine.
- A `segment` commits the text; subsequent `partial` frames belong to the next
  utterance.
- Exactly one `done` is sent for a well-formed session, after the final
  `segment` (if any).
- An `error` frame is terminal.

### Minimal example (Python client)

```python
import asyncio, json, wave
import websockets

async def transcribe(path):
    wf = wave.open(path, "rb")  # must be 16kHz mono s16le PCM WAV
    assert wf.getframerate() == 16000 and wf.getnchannels() == 1
    async with websockets.connect("ws://localhost:8094/v1/stream") as ws:
        await ws.send(json.dumps({"type": "start", "sample_rate": 16000, "language": "en"}))

        async def sender():
            chunk = wf.readframes(1600)  # ~100ms
            while chunk:
                await ws.send(chunk)        # BINARY frame
                chunk = wf.readframes(1600)
            await ws.send(json.dumps({"type": "end"}))

        send_task = asyncio.create_task(sender())
        async for msg in ws:
            evt = json.loads(msg)
            if evt["type"] == "segment":
                print(f"[{evt['start_ms']}-{evt['end_ms']}ms] {evt['text']}")
            elif evt["type"] == "done":
                break
            elif evt["type"] == "error":
                raise RuntimeError(evt["message"])
        await send_task

asyncio.run(transcribe("sample_16k_mono.wav"))
```

## Error semantics

| Condition | Response |
|---|---|
| Audio sent before a `start` frame | `error` frame, session continues |
| `sample_rate` != 16000 in `start` | `error` frame, connection closes |
| Invalid JSON control frame | `error` frame, session continues |
| Model not loaded yet | `error` frame on connect, connection closes |
| Internal/runtime failure | `error` frame, connection closes |
