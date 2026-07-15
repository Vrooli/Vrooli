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
    {"type":"processed","processed_batches":int}
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
import time
from collections import deque
from typing import Awaitable, Callable, Deque, List, Optional

import numpy as np
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import JSONResponse

SERVER_VERSION = "0.2.0"

# Canonical input contract: audio-tools always sends 16 kHz mono PCM s16le.
INPUT_SAMPLE_RATE = 16000

# Configuration (env-driven; safe defaults for the bundled 1B model).
HF_REPO = os.environ.get("KYUTAI_STT_HF_REPO", "kyutai/stt-1b-en_fr")
DEVICE = os.environ.get("KYUTAI_STT_DEVICE", "cuda")


def _enabled(value: str) -> bool:
    """Parse an operator boolean without accepting accidental truthiness."""
    return value.strip().lower() in {"1", "true", "yes", "on"}


# moshi lazily enables torch.compile on its first model step unless this flag is
# set *before* importing moshi.  On the local RTX 4070 Ti SUPER that deferred
# compile creates a 32-worker Inductor pool and can occupy the first dictation
# session for many minutes, while /health has already advertised ready.  CUDA
# graphing remains enabled in moshi, so the normal streaming path is still
# accelerated without making a user turn pay an unbounded compilation cost.
# Operators can opt into the experimental compiler only after benchmarking it
# on their own hardware.
TORCH_COMPILE_ENABLED = _enabled(os.environ.get("KYUTAI_STT_TORCH_COMPILE", "0"))
if TORCH_COMPILE_ENABLED:
    os.environ.pop("NO_TORCH_COMPILE", None)
else:
    os.environ["NO_TORCH_COMPILE"] = "1"

# End-of-padding token id (word boundary) is a fixed model convention.
END_OF_PADDING_ID = 0

# A run of this many consecutive padding frames (no text) after some committed
# words is treated as a speaking pause and commits the pending segment. At
# 12.5 Hz, 16 frames ~= 1.3 s. Overridable for tuning.
SILENCE_COMMIT_FRAMES = int(os.environ.get("KYUTAI_STT_SILENCE_COMMIT_FRAMES", "16"))

# During CONTINUOUS speech (no pause long enough to trip SILENCE_COMMIT_FRAMES),
# force-commit the pending segment once it has spanned this many frames, at the
# next word boundary, so a long unbroken utterance still produces durable
# segments instead of stalling as a volatile partial until the end flush. At
# 12.5 Hz, 48 frames ~= 3.8 s. Set to 0 to disable force-commit (legacy
# pause-or-flush-only behaviour). Sibling knob to SILENCE_COMMIT_FRAMES.
MAX_SEGMENT_FRAMES = int(os.environ.get("KYUTAI_STT_MAX_SEGMENT_FRAMES", "48"))

# Bounded wait (seconds) to acquire the single-session MODEL.lock before a new
# connection reaps an abandoned/wedged prior session. A stuck stream (e.g. a
# half-open consumer that stopped reading) can no longer starve the next
# recording indefinitely. See docs/OPERATIONS.md.
LOCK_ACQUIRE_TIMEOUT_S = float(os.environ.get("KYUTAI_STT_LOCK_TIMEOUT_S", "10"))

# FIFO admission is distinct from the model lock: contenders wait visibly
# outside the decoder so a healthy active holder is never cancelled merely
# because another user arrived. Audio Tools retains the queued turn for later
# replay/recovery; this resource accepts audio only after emitting ready.
ADMISSION_MAX_DEPTH = int(os.environ.get("KYUTAI_STT_ADMISSION_MAX_DEPTH", "8"))
ADMISSION_MAX_WAIT_S = float(os.environ.get("KYUTAI_STT_ADMISSION_MAX_WAIT_S", "30"))

# A WebSocket transport alone must not reserve the single decoder. The client
# sends its start control frame before it can receive ``ready``; this short
# handshake limit keeps abandoned browser upgrades from becoming an invisible
# admission holder.
START_FRAME_TIMEOUT_S = float(os.environ.get("KYUTAI_STT_START_FRAME_TIMEOUT_S", "5"))

# A lock holder with decode activity newer than this is active, not wedged.
# Contenders are rejected with a typed busy error instead of cancelling the
# holder. Holders with no recent decode activity are still reapable, preserving
# the original wedge protection.
ACTIVITY_WEDGE_TIMEOUT_S = float(os.environ.get("KYUTAI_STT_ACTIVITY_WEDGE_S", "5"))

# Bounded wait (seconds) for the send worker to drain queued durable events to a
# slow consumer during teardown before the socket is force-closed. Committed
# text is flushed within this window; a dead consumer cannot hang teardown past
# it. Sibling to the relay's drain deadline on the audio-tools side.
SEND_DRAIN_TIMEOUT_S = float(os.environ.get("KYUTAI_STT_SEND_DRAIN_TIMEOUT_S", "5"))

# Model stepping is compute-heavy and normally synchronous. Yield periodically
# during a multi-frame input batch so the WebSocket sender can emit processed
# credits and the ASGI runtime can service control traffic. This is scheduling,
# not a decode shortcut: every frame still reaches mimi + lm_gen in order.
DECODE_YIELD_FRAMES = int(os.environ.get("KYUTAI_STT_DECODE_YIELD_FRAMES", "4"))

logging.basicConfig(
    level=os.environ.get("KYUTAI_STT_LOG_LEVEL", "INFO"),
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger("kyutai-stt")

app = FastAPI(title="Kyutai STT", version=SERVER_VERSION)


class AdmissionRejected(Exception):
    """The bounded FIFO queue cannot accept another session."""


class AdmissionTimedOut(Exception):
    """A queued session exceeded its explicit operator wait bound."""


class FIFOAdmission:
    """Cancellation-safe FIFO admission for one local decoder.

    The admission queue deliberately does not own model health/reaping; it
    serializes contenders before the model lock so healthy work survives and
    callers receive queued/ready/cancelled/timed-out lifecycle states.
    """

    def __init__(self, max_depth: int, max_wait_s: float) -> None:
        self.max_depth = max_depth
        self.max_wait_s = max_wait_s
        self._guard = asyncio.Lock()
        self._waiters: Deque[asyncio.Future] = deque()
        self._active: Optional[asyncio.Future] = None

    async def acquire(
        self, on_queued: Optional[Callable[[int], Awaitable[None]]] = None
    ) -> tuple[asyncio.Future, int]:
        future = asyncio.get_running_loop().create_future()
        async with self._guard:
            if len(self._waiters) >= self.max_depth:
                raise AdmissionRejected("kyutai admission queue is full")
            self._waiters.append(future)
            position = len(self._waiters)
            self._promote_locked()
        if position > 1 and on_queued is not None:
            await on_queued(position)
        try:
            await asyncio.wait_for(asyncio.shield(future), timeout=self.max_wait_s)
        except asyncio.TimeoutError as exc:
            await self.cancel(future)
            raise AdmissionTimedOut("kyutai admission wait timed out") from exc
        except asyncio.CancelledError:
            await self.cancel(future)
            raise
        return future, position

    async def cancel(self, future: asyncio.Future) -> None:
        async with self._guard:
            if future in self._waiters:
                self._waiters.remove(future)
            if self._active is future:
                self._active = None
            if not future.done():
                future.cancel()
            self._promote_locked()

    async def release(self, future: asyncio.Future) -> None:
        async with self._guard:
            if self._active is future:
                self._active = None
            if future in self._waiters:
                self._waiters.remove(future)
            self._promote_locked()

    def _promote_locked(self) -> None:
        if self._active is not None or not self._waiters:
            return
        self._active = self._waiters[0]
        if not self._active.done():
            self._active.set_result(None)


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
        # The task currently holding `lock`, so a new connection can reap an
        # abandoned/wedged holder instead of starving forever behind it.
        self.current_session_task: "Optional[asyncio.Task]" = None
        self.current_session_last_activity: Optional[float] = None
        self.current_session_started_at: Optional[float] = None
        self.admission = FIFOAdmission(ADMISSION_MAX_DEPTH, ADMISSION_MAX_WAIT_S)

    def load(self) -> None:
        import torch
        import julius
        from moshi.models import loaders, LMGen

        log.info(
            "loading kyutai stt model repo=%s device=%s torch_compile=%s",
            self.repo,
            self.device,
            TORCH_COMPILE_ENABLED,
        )
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
    """Liveness: the process can answer even while the model is loading."""
    return JSONResponse(
        {
            "status": "ok",
            "model_loaded": MODEL.loaded,
            "device": MODEL.device,
        }
    )


@app.get("/ready")
async def ready() -> JSONResponse:
    """Readiness: callers may admit audio only after model weights are live."""
    if not MODEL.loaded:
        return JSONResponse(
            {"status": "starting", "model_loaded": False, "device": MODEL.device},
            status_code=503,
        )
    return JSONResponse(
        {"status": "ok", "model_loaded": True, "device": MODEL.device}
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
        self.started_at = time.monotonic()
        self.segments_emitted = 0

        # ── Decode/send decoupling (event-durability contract) ──
        # The decode loop ENQUEUES events and keeps stepping regardless of
        # consumer speed; a send worker drains these to the socket. Durable
        # events (segment/done/error) are ordered and lossless; the partial is
        # a single coalesced-to-latest slot that may be dropped under pressure
        # and MUST NEVER back-pressure decode. See
        # scenarios/audio-tools/docs/domains/stt/streaming-pipeline.md#event-durability-contract.
        self._durables: Deque[dict] = deque()  # ordered, never dropped
        self._latest_partial: Optional[str] = None  # coalesced-to-latest
        self._last_sent_partial = ""  # de-dupe on the wire
        # Absolute count of binary batches accepted by this session. This is a
        # coalesced credit signal for the client-side bounded in-flight window,
        # including a codec/model-frame remainder, not a transcript/audit
        # payload.
        self.processed_batches = 0
        self._last_sent_processed_batches = 0
        self._wake = asyncio.Event()  # signals the send worker
        self._closed = False  # no more events will be enqueued
        self._sender_task: "Optional[asyncio.Task]" = None

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

    # -- outbound: decode enqueues, a send worker drains -------------------

    def _enqueue_durable(self, obj: dict) -> None:
        """Queue a durable event (segment/done/error): ordered and lossless.
        Non-blocking — decode never waits on the consumer."""
        self._durables.append(obj)
        self._wake.set()

    def _enqueue_partial(self, text: str) -> None:
        """Coalesce the latest partial into a single slot. Disposable: a newer
        partial overwrites an unsent one; never back-pressures decode."""
        if text and text != self.last_partial:
            self.last_partial = text
            self._latest_partial = text
            self._wake.set()

    async def _emit_partial(self) -> None:
        self._enqueue_partial(self._decode(self.seg_tokens))

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
        # This segment commits the text the partial was tracking, so the pending
        # partial is now durable — drop the coalesced slot to avoid re-sending
        # already-committed words after the segment.
        self._latest_partial = None
        self.segments_emitted += 1
        self._enqueue_durable(
            {
                "type": "segment",
                "text": text,
                "start_ms": self._ms_for_frame(start_frame or 0),
                "end_ms": self._ms_for_frame(end_frame),
            }
        )

    def enqueue_error(self, message: str, code: Optional[str] = None) -> None:
        obj = {"type": "error", "message": message}
        if code:
            obj["code"] = code
        self._enqueue_durable(obj)

    def enqueue_done(self) -> None:
        self._enqueue_durable({"type": "done"})

    async def _drain_outbound(self) -> None:
        """Flush all pending durables in order, then the coalesced partial.
        Durable sends may block on a slow consumer — that is fine, they are
        lossless; only the worker waits, never the decode loop."""
        while self._durables:
            obj = self._durables.popleft()
            await self.ws.send_text(json.dumps(obj))
        if self.processed_batches > self._last_sent_processed_batches:
            self._last_sent_processed_batches = self.processed_batches
            await self.ws.send_text(json.dumps({
                "type": "processed",
                "processed_batches": self.processed_batches,
            }))
        if self._latest_partial is not None and self._latest_partial != self._last_sent_partial:
            self._last_sent_partial = self._latest_partial
            partial = self._latest_partial
            self._latest_partial = None
            await self.ws.send_text(json.dumps({"type": "partial", "text": partial}))

    async def _sender_loop(self) -> None:
        while True:
            await self._wake.wait()
            self._wake.clear()
            await self._drain_outbound()
            if self._closed and not self._durables and self._latest_partial is None:
                return

    def start_sender(self) -> None:
        """Start the background send worker. Idempotent."""
        if self._sender_task is None:
            self._sender_task = asyncio.create_task(self._sender_loop())

    async def close(self) -> None:
        """Drain-then-close: signal end-of-events, give the worker a bounded
        window to flush queued durables to the consumer, then stop it. A dead
        or wedged consumer cannot hang teardown past SEND_DRAIN_TIMEOUT_S."""
        self._closed = True
        self._wake.set()
        if self._sender_task is not None:
            try:
                await asyncio.wait_for(self._sender_task, timeout=SEND_DRAIN_TIMEOUT_S)
            except (asyncio.TimeoutError, asyncio.CancelledError, Exception):  # noqa: BLE001
                self._sender_task.cancel()
                try:
                    await self._sender_task
                except (asyncio.CancelledError, Exception):  # noqa: BLE001
                    pass
            self._sender_task = None

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
        if token == END_OF_PADDING_ID:
            # Word boundary. If the pending segment has run past the force-commit
            # window, commit it HERE so the split lands between words (never
            # mid-word) — continuous speech commits durable text without a pause.
            if (
                MAX_SEGMENT_FRAMES > 0
                and self.seg_start_frame is not None
                and self.frames_consumed - self.seg_start_frame >= MAX_SEGMENT_FRAMES
            ):
                await self._commit_segment()
            return
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
            MODEL.current_session_last_activity = time.monotonic()
            if text_tokens is None:
                if DECODE_YIELD_FRAMES > 0 and (i + 1) % DECODE_YIELD_FRAMES == 0:
                    await asyncio.sleep(0)
                continue
            token = int(text_tokens[0, 0, 0].cpu().item())
            await self._step_token(token)
            if DECODE_YIELD_FRAMES > 0 and (i + 1) % DECODE_YIELD_FRAMES == 0:
                await asyncio.sleep(0)

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


class ModelBusyError(Exception):
    """Raised when a contender finds an actively decoding holder."""


async def _acquire_model_lock_or_reap() -> None:
    """Acquire the single-session MODEL.lock, reaping an abandoned/wedged prior
    session if it does not release within LOCK_ACQUIRE_TIMEOUT_S. A stuck stream
    (e.g. a half-open consumer whose send worker is blocked forever) can no
    longer starve the next recording."""
    try:
        await asyncio.wait_for(MODEL.lock.acquire(), timeout=LOCK_ACQUIRE_TIMEOUT_S)
    except asyncio.TimeoutError:
        holder = MODEL.current_session_task
        now = time.monotonic()
        last = MODEL.current_session_last_activity
        started = MODEL.current_session_started_at
        idle_s = None if last is None else now - last
        age_s = None if started is None else now - started
        if holder is not None and not holder.done():
            if idle_s is not None and idle_s <= ACTIVITY_WEDGE_TIMEOUT_S:
                log.warning(
                    "rejecting streaming session because active holder owns model lock "
                    "holder_age_s=%.3f holder_idle_s=%.3f activity_wedge_s=%.3f",
                    age_s if age_s is not None else -1.0,
                    idle_s,
                    ACTIVITY_WEDGE_TIMEOUT_S,
                )
                raise ModelBusyError("kyutai model busy: active streaming session")
            log.warning(
                "reaping wedged prior streaming session to free the model lock "
                "holder_age_s=%.3f holder_idle_s=%.3f activity_wedge_s=%.3f",
                age_s if age_s is not None else -1.0,
                idle_s if idle_s is not None else -1.0,
                ACTIVITY_WEDGE_TIMEOUT_S,
            )
            holder.cancel()
            try:
                await holder
            except (asyncio.CancelledError, Exception):  # noqa: BLE001
                pass
        # The reaped holder released the lock in its finally; if it somehow did
        # not, acquire normally (bounded only by a genuinely healthy successor).
        await MODEL.lock.acquire()
    MODEL.current_session_task = asyncio.current_task()
    MODEL.current_session_started_at = time.monotonic()
    MODEL.current_session_last_activity = None


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
    # Validate the protocol intent before joining FIFO admission. In
    # particular, a bare WS connection cannot consume the one local decoder
    # slot and make real dictation appear stuck in queue.
    try:
        initial = await asyncio.wait_for(ws.receive(), timeout=START_FRAME_TIMEOUT_S)
    except asyncio.TimeoutError:
        await ws.send_text(json.dumps({
            "type": "error", "code": "start_timeout",
            "message": "start control frame was not received before admission timeout",
        }))
        await ws.close()
        return
    if initial.get("type") == "websocket.disconnect":
        return
    if "text" not in initial or initial["text"] is None:
        await ws.send_text(json.dumps({
            "type": "error", "code": "start_required",
            "message": "start control frame is required before admission",
        }))
        await ws.close()
        return
    try:
        initial_payload = json.loads(initial["text"])
    except json.JSONDecodeError:
        await ws.send_text(json.dumps({
            "type": "error", "code": "invalid_start",
            "message": "start control frame must be valid JSON",
        }))
        await ws.close()
        return
    if initial_payload.get("type") != "start":
        await ws.send_text(json.dumps({
            "type": "error", "code": "start_required",
            "message": "start control frame is required before admission",
        }))
        await ws.close()
        return
    try:
        sample_rate = int(initial_payload.get("sample_rate", INPUT_SAMPLE_RATE))
    except (TypeError, ValueError):
        await ws.send_text(json.dumps({
            "type": "error", "code": "unsupported_sample_rate",
            "message": f"sample_rate must be {INPUT_SAMPLE_RATE}",
        }))
        await ws.close()
        return
    if sample_rate != INPUT_SAMPLE_RATE:
        await ws.send_text(json.dumps({
            "type": "error", "code": "unsupported_sample_rate",
            "message": f"unsupported sample_rate {sample_rate}; contract requires {INPUT_SAMPLE_RATE}",
        }))
        await ws.close()
        return
    started = True
    admission_ticket: Optional[asyncio.Future] = None
    try:
        admission_ticket, _ = await MODEL.admission.acquire(
            lambda position: ws.send_text(json.dumps({
                "type": "queued", "position": position,
                "message": "Waiting for the local Kyutai decoder.",
            }))
        )
        await ws.send_text(json.dumps({"type": "ready"}))
    except AdmissionRejected as exc:
        await ws.send_text(json.dumps({"type": "rejected", "code": "admission_full", "message": str(exc)}))
        await ws.close()
        return
    except AdmissionTimedOut as exc:
        await ws.send_text(json.dumps({"type": "timed_out", "code": "admission_timeout", "message": str(exc)}))
        await ws.close()
        return
    # One streaming session at a time on the shared GPU model instance. Bound
    # the acquire so a wedged prior session is reaped, not waited on forever.
    try:
        await _acquire_model_lock_or_reap()
    except ModelBusyError as exc:
        if admission_ticket is not None:
            await MODEL.admission.release(admission_ticket)
        await ws.send_text(
            json.dumps({"type": "error", "code": "stt_busy", "message": str(exc)})
        )
        log.info(
            "stream close summary reason=busy-rejected frames=%d segments=%d duration_s=%.3f",
            session.frames_consumed,
            session.segments_emitted,
            time.monotonic() - session.started_at,
        )
        await ws.close()
        return
    # Start the send worker so decode can enqueue events without ever blocking
    # on the consumer (event-durability contract).
    session.start_sender()
    mimi = MODEL._mimi
    lm_gen = MODEL._lm_gen
    try:
        close_reason = "disconnect"
        with mimi.streaming(1), lm_gen.streaming(1):
            while True:
                msg = await ws.receive()
                if msg.get("type") == "websocket.disconnect":
                    break
                if "text" in msg and msg["text"] is not None:
                    try:
                        payload = json.loads(msg["text"])
                    except json.JSONDecodeError:
                        session.enqueue_error("invalid json control frame")
                        continue
                    mtype = payload.get("type")
                    if mtype == "start":
                        # The validated start frame was consumed before
                        # admission. A second start has no useful meaning and
                        # is rejected rather than silently resetting a turn.
                        session.enqueue_error("duplicate start control frame", "duplicate_start")
                        break
                    elif mtype == "end":
                        await session.flush()
                        session.enqueue_done()
                        close_reason = "graceful-end"
                        break
                    else:
                        session.enqueue_error(f"unknown control type {mtype}")
                elif "bytes" in msg and msg["bytes"] is not None:
                    if not started:
                        session.enqueue_error("audio before start frame")
                        continue
                    await session.feed(msg["bytes"])
                    # Credit every accepted binary batch, including a short
                    # chunk retained as codec/model-frame remainder. The
                    # client window is transport flow control, not a count of
                    # fully stepped model frames.
                    session.processed_batches += 1
                    session._wake.set()
    except WebSocketDisconnect:
        close_reason = "disconnect"
        log.info("client disconnected")
    except asyncio.CancelledError:
        # Reaped by a successor session — unwind and release the lock so the
        # next recording proceeds.
        close_reason = "reaped"
        log.info("streaming session cancelled (reaped)")
    except Exception as exc:  # noqa: BLE001 - report to client, don't crash
        close_reason = "error"
        log.exception("stream error")
        session.enqueue_error(str(exc))
    finally:
        # Drain-then-close: flush queued durables to the consumer within the
        # bounded window, then stop the worker and close the socket.
        try:
            await session.close()
        except Exception:  # noqa: BLE001
            pass
        if MODEL.current_session_task is asyncio.current_task():
            MODEL.current_session_task = None
            MODEL.current_session_last_activity = None
            MODEL.current_session_started_at = None
        if MODEL.lock.locked():
            MODEL.lock.release()
        if admission_ticket is not None:
            await MODEL.admission.release(admission_ticket)
        log.info(
            "stream close summary reason=%s frames=%d segments=%d duration_s=%.3f",
            close_reason,
            session.frames_consumed,
            session.segments_emitted,
            time.monotonic() - session.started_at,
        )
        try:
            await ws.close()
        except Exception:  # noqa: BLE001
            pass


if __name__ == "__main__":
    import uvicorn

    port = int(os.environ.get("KYUTAI_STT_CONTAINER_PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")
