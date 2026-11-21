# Kale TTS Nano API

Base URL: `http://127.0.0.1:18896`

`GET /healthz` returns `status`, `model`, `loaded`, and `ready`. It checks the
graph, tokenizer, voice catalogue, and English frontend files without warming
the ONNX session.

`GET /v1/audio/voices` returns `{ "model": "kale-tts-1-nano", "voices":
[...] }`.

`POST /v1/audio/speech` takes JSON with `model`, `input`, `voice`, and
`response_format`. `model` must be `kale-tts-1-nano`; `input` must be 1 to
5,000 non-whitespace characters; `response_format` is `wav` or `pcm`. Requests
are capped at 64 KiB. WAV is mono, 24 kHz, signed 16-bit PCM. PCM is the same
sample representation without a container header and includes
`X-Sample-Rate: 24000`. Unsupported formats and unknown voices return HTTP 400.
Synthesis runs serially to constrain CPU and working memory.
