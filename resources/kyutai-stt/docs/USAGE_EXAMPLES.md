# Kyutai STT Usage Examples

How to drive the Kyutai streaming speech-to-text service. The contract accepts
canonical **PCM s16le 16 kHz mono** audio over a WebSocket and emits JSON
partial/segment events. See [API.md](/home/matthalloran8/Vrooli/resources/kyutai-stt/docs/API.md)
for the authoritative contract.

## Lifecycle

```bash
# Install + build + start (first run downloads weights, several minutes)
vrooli resource install kyutai-stt

# Check it's up
resource-kyutai-stt status
curl -s http://localhost:8094/health | jq .
curl -s http://localhost:8094/v1/info | jq .
```

## HTTP probes

```bash
# Liveness + model-load + device
curl -s http://localhost:8094/health | jq .
# => {"status":"ok","model_loaded":true,"device":"cuda"}

# Backend / model / accepted sample rate / version
curl -s http://localhost:8094/v1/info | jq .
# => {"backend":"kyutai","model":"kyutai/stt-1b-en_fr","device":"cuda","sample_rate":16000,"version":"0.1.0"}
```

## Streaming transcription (Python)

The canonical client. Requires a 16 kHz mono s16le source.

```python
import asyncio, json, wave
import websockets

WS = "ws://localhost:8094/v1/stream"

async def main(wav_path):
    wf = wave.open(wav_path, "rb")
    assert wf.getframerate() == 16000 and wf.getnchannels() == 1 and wf.getsampwidth() == 2

    async with websockets.connect(WS) as ws:
        await ws.send(json.dumps({"type": "start", "sample_rate": 16000, "language": "en"}))

        async def pump():
            # ~100 ms chunks
            data = wf.readframes(1600)
            while data:
                await ws.send(data)  # BINARY
                data = wf.readframes(1600)
            await ws.send(json.dumps({"type": "end"}))

        pump_task = asyncio.create_task(pump())
        async for raw in ws:
            evt = json.loads(raw)
            t = evt["type"]
            if t == "partial":
                print("…", evt["text"], end="\r")
            elif t == "segment":
                print(f"\n[{evt['start_ms']}-{evt['end_ms']}ms] {evt['text']}")
            elif t == "done":
                break
            elif t == "error":
                raise RuntimeError(evt["message"])
        await pump_task

asyncio.run(main("sample_16k_mono.wav"))
```

## Streaming transcription (Node.js)

```javascript
import WebSocket from 'ws';
import fs from 'fs';

const ws = new WebSocket('ws://localhost:8094/v1/stream');

ws.on('open', () => {
  ws.send(JSON.stringify({ type: 'start', sample_rate: 16000, language: 'en' }));

  // raw 16kHz mono s16le PCM (NOT a WAV header)
  const pcm = fs.readFileSync('sample_16k_mono.pcm');
  const chunk = 3200; // ~100ms (1600 samples * 2 bytes)
  for (let i = 0; i < pcm.length; i += chunk) {
    ws.send(pcm.subarray(i, i + chunk));
  }
  ws.send(JSON.stringify({ type: 'end' }));
});

ws.on('message', (data) => {
  const evt = JSON.parse(data.toString());
  if (evt.type === 'segment') {
    console.log(`[${evt.start_ms}-${evt.end_ms}ms] ${evt.text}`);
  } else if (evt.type === 'done') {
    ws.close();
  } else if (evt.type === 'error') {
    console.error('error:', evt.message);
    ws.close();
  }
});
```

## Producing canonical PCM with ffmpeg

The server only accepts 16 kHz mono s16le. Convert any source first:

```bash
# To a raw PCM stream (no header) for BINARY frames
ffmpeg -i input.mp3 -ac 1 -ar 16000 -f s16le -acodec pcm_s16le sample_16k_mono.pcm

# To a 16kHz mono WAV (for the Python wave example)
ffmpeg -i input.mp3 -ac 1 -ar 16000 -acodec pcm_s16le sample_16k_mono.wav
```

## Notes

- One streaming session per connection. The server serializes concurrent
  connections on the shared GPU model instance.
- `partial` frames are best-effort interim hypotheses; `segment` frames are the
  committed results you should persist.
- For end-to-end voice (STT + TTS), pair `kyutai-stt` with the native
  `sherpa-onnx` resource. The Kyutai resource is an optional Linux/CUDA
  streaming accelerator, not the required TTS or streaming backend.
