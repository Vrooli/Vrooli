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

### Streaming flow control

`/v1/stream` emits an additive progress frame after accepting one or more input
batches:

```json
{ "type": "processed", "processed_batches": 42 }
```

`processed_batches` is an absolute, monotonically increasing count. Audio Tools
uses it as a bounded credit window (eight unprocessed PCM batches maximum), so a
fast deterministic replay cannot fill the transport write buffer and block
WebSocket ping/pong control traffic. It carries no audio or transcript data.

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

### Readiness

**GET** `/ready`

Returns the same metadata with HTTP `200` only after model weights are loaded.
While starting it returns HTTP `503` and `{ "status": "starting", "model_loaded": false }`.
Use this endpoint for dependency admission and orchestration; keep `/health`
for process liveness and diagnostics.

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

   The server may reply `queued` before it replies `ready`. A client MUST wait
   for `ready` before sending binary PCM; while queued it must retain/replay
   its own canonical audio rather than relying on WebSocket buffering.

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
| queued | `{"type":"queued","position":<int>}` | The FIFO admission queue accepted the session; `position` is one-based. No audio is accepted yet. |
| ready | `{"type":"ready"}` | The session owns decoder admission and may now send PCM. |
| processed | `{"type":"processed","processed_batches":<int>}` | Monotonic count of decoded binary batches. Clients use it as bounded transport credit; it carries no audio or transcript data. |
| timed_out | `{"type":"timed_out","code":"admission_timeout",...}` | The bounded admission wait expired. No PCM was accepted. |
| rejected | `{"type":"rejected","code":"admission_full",...}` | The FIFO admission queue was full. No PCM was accepted. |
| segment | `{"type":"segment","text":"<committed>","start_ms":<int>,"end_ms":<int>}` | One finalized utterance/segment. `start_ms`/`end_ms` are millisecond offsets from the start of the stream. |
| done | `{"type":"done"}` | Sent after the server receives `end` and flushes all buffered audio. |
| error | `{"type":"error","message":"<reason>"}` | A protocol or runtime error. The connection then closes. |

### Frame ordering guarantees

- `partial` frames (if any) precede the `segment` they refine.
- `queued` and `ready` are durable admission lifecycle events. `ready` is the
  sole permission to send binary audio; receipt of `start` alone is not. The
  client sends `start` first; the server validates it before allocating a queue
  position, so an idle WebSocket cannot reserve the decoder.
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
