"""Kyutai STT streaming server.

Wraps the Kyutai delayed-streams-modeling speech-to-text model (loaded via the
`moshi` package) behind a STABLE FastAPI + WebSocket contract that the
audio-tools scenario consumes. The contract is intentionally narrow and
version-pinned; see docs/API.md for the authoritative description.

Contract summary (do not change without bumping the major version):

  GET  /health        -> {"status":"ok","model_loaded":bool,"device":"cuda|cpu"}
  GET  /v1/info        -> {"backend":"kyutai","model":...,"device":...,
                            "sample_rate":16000,"version":...}
  WS   /v1/stream      -> bidirectional streaming transcription. The server
                          ACCEPTS canonical PCM s16le 16 kHz mono and resamples
                          internally to whatever rate the model needs.

The wire protocol on /v1/stream:

  client -> server:
    1. TEXT frame: {"type":"start","sample_rate":16000,"language":"en"}
       (language may be "" for auto/model-default)
    2. BINARY frames: raw little-endian 16-bit PCM mono @ 16 kHz samples
    3. TEXT frame: {"type":"end"}

  server -> client (each a JSON TEXT frame with a "type"):
    {"type":"partial","text":"<interim hypothesis>"}        (optional)
    {"type":"segment","text":"<committed>","start_ms":int,"end_ms":int}
    {"type":"done"}
    {"type":"error","message":"<reason>"}

Decode faithfulness
-------------------
Kyutai STT is a *delayed-streams* model: each 12.5 Hz frame the language model
emits ONE text token, but the text lags the audio by `audio_delay_seconds`
(0.5 s for stt-1b-en_fr). The token stream uses two control ids:

  * `text_padding_id` (3): "no text this frame".
  * end-of-padding id (0): a word boundary.

Real words are tokens with id > padding_id and are detokenized with the
SentencePiece tokenizer (which reconstructs inter-word spacing from its
``▁`` word-start markers). This mirrors kyutai-labs/delayed-streams-modeling's
reference decode (`tokens_to_timestamped_text`); an earlier hand-rolled loop
that guessed silence ids, decoded per-token, and never flushed the delay tail
dropped words and lost spacing.
"""

from __future__ import annotations

import asyncio
import json
import logging
import math
import os
from typing import List, Optional

import numpy as np
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import JSONResponse

SERVER_VERSION = "0.2.0"

# Canonical input contract: audio-tools always sends 16 kHz mono PCM s16le.
INPUT_SAMPLE_RATE = 16000

# Configuration (env-driven; safe defaults for the bundled 1B model).
HF_REPO = os.environ.get("KYUTAI_STT_HF_REPO", "kyutai/stt-1b-en_fr")
DEVICE = os.environ.get("KYUTAI_STT_DEVICE", "cuda")

# End-of-padding token id (word boundary) is a fixed model convention.
END_OF_PADDING_ID = 0

# A run of this many consecutive padding frames (no text) after some committed
# words is treated as a speaking pause and commits the pending segment. At
# 12.5 Hz, 16 frames ~= 1.3 s. Overridable for tuning.
SILENCE_COMMIT_FRAMES = int(os.environ.get("KYUTAI_STT_SILENCE_COMMIT_FRAMES", "16"))

logging.basicConfig(
    level=os.environ.get("KYUTAI_STT_LOG_LEVEL", "INFO"),
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger("kyutai-stt")

app = FastAPI(title="Kyutai STT", version=SERVER_VERSION)


class Model:
    """Lazily-loaded Kyutai STT model wrapper.

    Holds the Mimi audio codec, the language model decoder, the text tokenizer
    and the streaming generator. All heavy imports (torch, moshi) happen inside
    load() so the process can boot and answer /health even before weights are
    resident.
    """

    def __init__(self) -> None:
        self.loaded = False
        self.device = DEVICE
        self.repo = HF_REPO
        self._torch = None
        self._julius = None
        self._mimi = None
        self._lm_gen = None
        self._tokenizer = None
        self._model_sample_rate = 24000
        self._frame_size = 1920  # 24000 / 12.5 Hz; refreshed from mimi on load
        # Decode config, refreshed from the checkpoint on load.
        self.text_padding_id = 3
        self.audio_delay_seconds = 0.5
        self.audio_silence_prefix_seconds = 0.0
        # Serialize access: a single GPU model instance is not safe to share
        # across concurrent streaming sessions.
        self.lock = asyncio.Lock()

    def load(self) -> None:
        import torch
        import julius
        from moshi.models import loaders, LMGen

        log.info("loading kyutai stt model repo=%s device=%s", self.repo, self.device)
        if self.device == "cuda" and not torch.cuda.is_available():
            log.warning("cuda requested but unavailable; falling back to cpu")
            self.device = "cpu"

        dtype = torch.bfloat16 if self.device == "cuda" else torch.float32
        info = loaders.CheckpointInfo.from_hf_repo(self.repo)
        mimi = info.get_mimi(device=self.device)
        tokenizer = info.get_text_tokenizer()
        lm = info.get_moshi(device=self.device, dtype=dtype)
        lm_gen = LMGen(lm, temp=0, temp_text=0.0)

        raw = info.raw_config or {}
        stt_cfg = raw.get("stt_config", {}) or {}

        self._torch = torch
        self._julius = julius
        self._mimi = mimi
        self._tokenizer = tokenizer
        self._lm_gen = lm_gen
        self._model_sample_rate = int(mimi.sample_rate)
        self._frame_size = int(mimi.frame_size)
        self.text_padding_id = int(raw.get("existing_text_padding_id", 3))
        self.audio_delay_seconds = float(stt_cfg.get("audio_delay_seconds", 0.5))
        self.audio_silence_prefix_seconds = float(
            stt_cfg.get("audio_silence_prefix_seconds", 0.0)
        )
        self.loaded = True
        log.info(
            "model loaded sample_rate=%d frame_size=%d frame_rate=%.3f "
            "text_padding_id=%d audio_delay_s=%.3f device=%s",
            self._model_sample_rate,
            self._frame_size,
            self.frame_rate,
            self.text_padding_id,
            self.audio_delay_seconds,
            self.device,
        )

    @property
    def model_sample_rate(self) -> int:
        return self._model_sample_rate

    @property
    def frame_size(self) -> int:
        return self._frame_size

    @property
    def frame_rate(self) -> float:
        return self._model_sample_rate / self._frame_size

    def resample_to_model(self, pcm_f32: "np.ndarray"):
        """Resample mono float32 PCM from 16 kHz to the model's native rate.

        Returns a torch tensor shaped [1, num_samples] on the model device.
        """
        torch = self._torch
        wav = torch.from_numpy(pcm_f32).to(self.device)
        if INPUT_SAMPLE_RATE != self._model_sample_rate:
            wav = self._julius.resample_frac(
                wav, INPUT_SAMPLE_RATE, self._model_sample_rate
            )
        return wav.unsqueeze(0)  # [1, T]


MODEL = Model()


@app.on_event("startup")
async def _startup() -> None:
    """Load the model in a worker thread so the event loop stays responsive."""

    async def _bg() -> None:
        try:
            await asyncio.to_thread(MODEL.load)
        except Exception:  # noqa: BLE001 - surface load failures in /health only
            log.exception("model load failed")

    asyncio.create_task(_bg())


@app.get("/health")
async def health() -> JSONResponse:
    return JSONResponse(
        {
            "status": "ok",
            "model_loaded": MODEL.loaded,
            "device": MODEL.device,
        }
    )


@app.get("/v1/info")
async def info() -> JSONResponse:
    return JSONResponse(
        {
            "backend": "kyutai",
            "model": MODEL.repo,
            "device": MODEL.device,
            "sample_rate": INPUT_SAMPLE_RATE,
            "version": SERVER_VERSION,
        }
    )


def _pcm16_to_float32(buf: bytes) -> "np.ndarray":
    """Decode little-endian s16 PCM bytes to float32 in [-1, 1]."""
    samples = np.frombuffer(buf, dtype="<i2").astype(np.float32)
    return samples / 32768.0


class StreamSession:
    """Per-connection streaming decode state.

    Resamples inbound 16 kHz PCM to the model rate, runs the mimi+LM streaming
    loop frame-by-frame, and turns the emitted text-token stream into
    partial/segment events using the model's real token conventions:
    `text_padding_id` = no text, `END_OF_PADDING_ID` (0) = word boundary,
    tokens > padding = text decoded via SentencePiece.
    """

    def __init__(self, ws: WebSocket) -> None:
        self.ws = ws
        self.pcm_tail = b""  # leftover odd byte across binary frames
        self.remainder = None  # torch tensor of un-framed model-rate samples
        self.frames_consumed = 0  # number of model frames stepped
        # Tokens accumulated for the current (uncommitted) segment, in frame
        # order. Includes control ids; decoding filters them out.
        self.seg_tokens: List[int] = []
        self.seg_start_frame: Optional[int] = None  # first text frame of segment
        self.last_text_frame = 0  # last frame that carried a real text token
        self.pad_run = 0  # consecutive padding frames since the last text token
        self.last_partial = ""  # de-dupe identical partial emissions

    # -- timing -----------------------------------------------------------

    def _ms_for_frame(self, frame_idx: int) -> int:
        # Text lags the audio by audio_delay_seconds; map the text-frame index
        # back onto the input-audio timeline.
        secs = frame_idx / MODEL.frame_rate - MODEL.audio_delay_seconds
        return max(0, int(secs * 1000.0))

    # -- decoding ---------------------------------------------------------

    def _decode(self, tokens: List[int]) -> str:
        """Detokenize real text tokens (id > padding) with SentencePiece."""
        pad = MODEL.text_padding_id
        ids = [t for t in tokens if t > pad]
        if not ids:
            return ""
        return MODEL._tokenizer.decode(ids).strip()

    async def _emit(self, obj: dict) -> None:
        await self.ws.send_text(json.dumps(obj))

    async def _emit_partial(self) -> None:
        text = self._decode(self.seg_tokens)
        if text and text != self.last_partial:
            self.last_partial = text
            await self._emit({"type": "partial", "text": text})

    async def _commit_segment(self) -> None:
        text = self._decode(self.seg_tokens)
        start_frame = self.seg_start_frame
        end_frame = self.last_text_frame
        self.seg_tokens = []
        self.seg_start_frame = None
        self.pad_run = 0
        self.last_partial = ""
        if not text:
            return
        await self._emit(
            {
                "type": "segment",
                "text": text,
                "start_ms": self._ms_for_frame(start_frame or 0),
                "end_ms": self._ms_for_frame(end_frame),
            }
        )

    # -- per-frame stepping ----------------------------------------------

    async def _step_token(self, token: int) -> None:
        self.seg_tokens.append(token)
        if token == MODEL.text_padding_id:
            self.pad_run += 1
            # A sustained pad run after committed words = speaking pause.
            if self.pad_run >= SILENCE_COMMIT_FRAMES and self.seg_start_frame is not None:
                await self._commit_segment()
            return
        # Non-padding: either a word boundary (0) or a real text token (>pad).
        self.pad_run = 0
        if token > MODEL.text_padding_id:
            if self.seg_start_frame is None:
                self.seg_start_frame = self.frames_consumed
            self.last_text_frame = self.frames_consumed
            await self._emit_partial()

    async def _run_frames(self, framed, n_frames: int) -> None:
        torch = MODEL._torch
        mimi = MODEL._mimi
        lm_gen = MODEL._lm_gen
        fs = MODEL.frame_size

        for i in range(n_frames):
            chunk = framed[:, i * fs:(i + 1) * fs].unsqueeze(0)  # [1,1,fs]
            with torch.no_grad():
                audio_tokens = mimi.encode(chunk)
                text_tokens = lm_gen.step(audio_tokens)
            self.frames_consumed += 1
            if text_tokens is None:
                continue
            token = int(text_tokens[0, 0, 0].cpu().item())
            await self._step_token(token)

    async def feed(self, pcm_bytes: bytes) -> None:
        torch = MODEL._torch
        data = self.pcm_tail + pcm_bytes
        # PCM s16 = 2 bytes/sample; stash any trailing odd byte.
        usable = len(data) - (len(data) % 2)
        self.pcm_tail = data[usable:]
        if usable == 0:
            return
        pcm_f32 = _pcm16_to_float32(data[:usable])
        wav = MODEL.resample_to_model(pcm_f32)  # [1, T] on device

        if self.remainder is not None:
            wav = torch.cat([self.remainder, wav], dim=-1)

        fs = MODEL.frame_size
        total = wav.shape[-1]
        n_frames = total // fs
        if n_frames == 0:
            self.remainder = wav
            return
        self.remainder = wav[:, n_frames * fs:]
        framed = wav[:, : n_frames * fs]

        await self._run_frames(framed, n_frames)

    async def flush(self) -> None:
        torch = MODEL._torch
        fs = MODEL.frame_size
        # Decode any buffered remainder (zero-padded to a full frame).
        if self.remainder is not None and self.remainder.shape[-1] > 0:
            rem = self.remainder.shape[-1] % fs
            if rem != 0:
                self.remainder = torch.nn.functional.pad(self.remainder, (0, fs - rem))
            n_frames = self.remainder.shape[-1] // fs
            if n_frames > 0:
                await self._run_frames(self.remainder, n_frames)
        self.remainder = None
        # Drain the delayed-streams tail: the text for the final ~audio_delay
        # seconds of speech is still in the pipeline, so feed that much silence.
        delay_frames = math.ceil(MODEL.audio_delay_seconds * MODEL.frame_rate)
        if delay_frames > 0:
            silence = torch.zeros((1, delay_frames * fs), device=MODEL.device)
            await self._run_frames(silence, delay_frames)
        if self.seg_start_frame is not None:
            await self._commit_segment()


@app.websocket("/v1/stream")
async def stream(ws: WebSocket) -> None:
    await ws.accept()
    if not MODEL.loaded:
        await ws.send_text(
            json.dumps({"type": "error", "message": "model not loaded yet"})
        )
        await ws.close()
        return

    session = StreamSession(ws)
    started = False
    # One streaming session at a time on the shared GPU model instance.
    async with MODEL.lock:
        mimi = MODEL._mimi
        lm_gen = MODEL._lm_gen
        try:
            with mimi.streaming(1), lm_gen.streaming(1):
                while True:
                    msg = await ws.receive()
                    if msg.get("type") == "websocket.disconnect":
                        break
                    if "text" in msg and msg["text"] is not None:
                        try:
                            payload = json.loads(msg["text"])
                        except json.JSONDecodeError:
                            await session._emit(
                                {"type": "error", "message": "invalid json control frame"}
                            )
                            continue
                        mtype = payload.get("type")
                        if mtype == "start":
                            started = True
                            sr = int(payload.get("sample_rate", INPUT_SAMPLE_RATE))
                            if sr != INPUT_SAMPLE_RATE:
                                await session._emit(
                                    {
                                        "type": "error",
                                        "message": (
                                            f"unsupported sample_rate {sr}; "
                                            f"contract requires {INPUT_SAMPLE_RATE}"
                                        ),
                                    }
                                )
                                break
                        elif mtype == "end":
                            await session.flush()
                            await session._emit({"type": "done"})
                            break
                        else:
                            await session._emit(
                                {"type": "error", "message": f"unknown control type {mtype}"}
                            )
                    elif "bytes" in msg and msg["bytes"] is not None:
                        if not started:
                            await session._emit(
                                {"type": "error", "message": "audio before start frame"}
                            )
                            continue
                        await session.feed(msg["bytes"])
        except WebSocketDisconnect:
            log.info("client disconnected")
        except Exception as exc:  # noqa: BLE001 - report to client, don't crash
            log.exception("stream error")
            try:
                await session._emit({"type": "error", "message": str(exc)})
            except Exception:  # noqa: BLE001
                pass
        finally:
            try:
                await ws.close()
            except Exception:  # noqa: BLE001
                pass


if __name__ == "__main__":
    import uvicorn

    port = int(os.environ.get("KYUTAI_STT_CONTAINER_PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")
