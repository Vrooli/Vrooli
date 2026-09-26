"""Unit tests for the Kyutai STT streaming server's commit/segmentation logic.

These are the first tests for server/server.py. They exercise the pure decode
and commit state machine (StreamSession._step_token / _commit_segment / flush)
WITHOUT loading torch or the real moshi model: the heavy model is stubbed so we
can drive the token stream directly and assert when durable ``segment`` events
are emitted.

Run: ``python -m pytest resources/kyutai-stt/tests/test_server.py``

Covers:
  * continuous-speech force-commit fires (a long unbroken utterance commits a
    durable segment before any pause or end flush)  -- the tail-durability fix
  * silence-commit still fires on a sustained pause
  * no premature commit for a short utterance under the force-commit window
  * flush() drains the delayed-streams tail and commits the trailing segment
  * disconnect-vs-end divergence: without flush the trailing segment is NOT
    delivered; the end path's flush is what carries it
"""

from __future__ import annotations

import asyncio
import contextlib
import json
import os
import sys

# Import the server module without its torch/moshi deps (they are imported
# lazily inside Model.load(), which we never call here).
sys.path.insert(0, os.path.dirname(__file__))
import server  # noqa: E402


def test_torch_compile_is_opt_in_before_moshi_import():
    """A live dictation turn must not trigger Inductor's unbounded cold compile.

    ``server`` sets NO_TORCH_COMPILE before Model.load imports moshi. Keep this
    assertion near the server tests so a Docker/config refactor cannot quietly
    restore the first-turn multi-minute stall.
    """
    assert server.TORCH_COMPILE_ENABLED is False
    assert os.environ["NO_TORCH_COMPILE"] == "1"


def test_readiness_distinguishes_live_process_from_loaded_model():
    old_loaded = server.MODEL.loaded
    try:
        server.MODEL.loaded = False
        starting = asyncio.run(server.ready())
        assert starting.status_code == 503
        assert json.loads(starting.body) == {
            "status": "starting", "model_loaded": False, "device": server.MODEL.device,
        }

        server.MODEL.loaded = True
        ready = asyncio.run(server.ready())
        assert ready.status_code == 200
        assert json.loads(ready.body) == {
            "status": "ok", "model_loaded": True, "device": server.MODEL.device,
        }
    finally:
        server.MODEL.loaded = old_loaded


def test_durable_event_queue_fails_closed_at_memory_budget():
    """A stalled consumer cannot turn durable transcript delivery into an
    unbounded resource allocation; already queued events remain intact for
    the drain/recovery path."""
    old_events, old_bytes = server.MAX_DURABLE_EVENTS, server.MAX_DURABLE_BYTES
    try:
        server.MAX_DURABLE_EVENTS = 2
        server.MAX_DURABLE_BYTES = 1024 * 1024
        session = server.StreamSession(FakeWS())
        session._enqueue_durable({"type": "segment", "text": "first"})
        session._enqueue_durable({"type": "segment", "text": "second"})
        queued_bytes = session._durable_bytes

        try:
            session._enqueue_durable({"type": "segment", "text": "third"})
        except server.DurableQueueOverflow:
            pass
        else:
            raise AssertionError("durable queue overflow must fail closed")

        assert len(session._durables) == 2
        assert session._durable_bytes == queued_bytes
        assert session._durable_overflow is True
    finally:
        server.MAX_DURABLE_EVENTS, server.MAX_DURABLE_BYTES = old_events, old_bytes


class FakeWS:
    """Minimal WebSocket capturing every JSON frame the session emits."""

    def __init__(self) -> None:
        self.sent: list = []

    async def send_text(self, text: str) -> None:
        import json

        self.sent.append(json.loads(text))


class HandshakeWS(FakeWS):
    """Minimal ASGI WebSocket for admission-handshake tests."""

    def __init__(self, messages):
        super().__init__()
        self.messages = list(messages)
        self.accepted = False
        self.closed = False

    async def accept(self) -> None:
        self.accepted = True

    async def receive(self):
        if self.messages:
            return self.messages.pop(0)
        await asyncio.Future()

    async def close(self) -> None:
        self.closed = True


class FakeTokenizer:
    """Detokenize by mapping each id to a token word; ids are already filtered
    to real text ids (> padding) by StreamSession._decode."""

    def decode(self, ids):
        return " ".join(f"w{i}" for i in ids)


def _install_fake_model(
    max_segment_frames: int,
    silence_commit_frames: int = 16,
    stream_reset_interval_frames: int = 0,
):
    """Stub the module-global MODEL so segmentation logic runs without torch."""
    m = server.MODEL
    m.text_padding_id = 3
    m.audio_delay_seconds = 0.5
    m._model_sample_rate = 24000
    m._frame_size = 1920  # frame_rate = 12.5 Hz
    m._tokenizer = FakeTokenizer()
    server.MAX_SEGMENT_FRAMES = max_segment_frames
    server.SILENCE_COMMIT_FRAMES = silence_commit_frames
    server.STREAM_RESET_INTERVAL_FRAMES = stream_reset_interval_frames


async def _step(session, tokens):
    """Drive tokens through the session the way _run_frames does: increment
    frames_consumed BEFORE stepping each token, then drain the decoupled
    outbound queue to the (non-blocking) FakeWS so ws.sent reflects the wire."""
    for tok in tokens:
        session.frames_consumed += 1
        await session._step_token(tok)
    await session._drain_outbound()


def _segments(ws):
    return [m for m in ws.sent if m.get("type") == "segment"]


# Token convention: id 3 = padding, id 0 = word boundary, id > 3 = real text.
TEXT = 10
PAD = 3
BOUNDARY = 0


def test_continuous_speech_force_commits_before_pause():
    """A long unbroken utterance (no >=1.3s pause) must still commit durable
    segments, not stall until the end flush. This is the tail-durability fix."""
    _install_fake_model(max_segment_frames=48)
    ws = FakeWS()
    session = server.StreamSession(ws)

    # 200 frames of continuous "word boundary" speech, no long padding run.
    tokens = [TEXT, BOUNDARY] * 100

    asyncio.run(_step(session, tokens))

    segs = _segments(ws)
    assert len(segs) >= 1, (
        "continuous speech committed no durable segment before the end flush; "
        "force-commit did not fire"
    )
    # Roughly one segment per ~max_segment_frames window (with word alignment).
    assert len(segs) >= 2, f"expected multiple force-commits over 200 frames, got {len(segs)}"


def test_long_stream_resets_decoder_only_after_durable_boundary():
    """Finite model context resets at a committed word boundary, not mid-word."""
    _install_fake_model(max_segment_frames=4, stream_reset_interval_frames=4)
    ws = FakeWS()
    session = server.StreamSession(ws)

    class Resettable:
        def __init__(self):
            self.reset_calls = 0

        def reset_streaming(self):
            self.reset_calls += 1

    mimi = Resettable()
    lm_gen = Resettable()
    old_mimi, old_lm_gen = server.MODEL._mimi, server.MODEL._lm_gen
    server.MODEL._mimi, server.MODEL._lm_gen = mimi, lm_gen

    try:
        asyncio.run(_step(session, [TEXT, BOUNDARY] * 8))
        assert len(_segments(ws)) >= 2
        assert mimi.reset_calls >= 1
        assert lm_gen.reset_calls == mimi.reset_calls
    finally:
        server.MODEL._mimi, server.MODEL._lm_gen = old_mimi, old_lm_gen


def test_decoder_reset_replays_bounded_acoustic_tail_without_emitting_text():
    """Reset warm-up must replay only recent frames and never commit replay text."""
    _install_fake_model(max_segment_frames=4, stream_reset_interval_frames=4)
    old_replay = server.STREAM_RESET_REPLAY_FRAMES
    server.STREAM_RESET_REPLAY_FRAMES = 2
    ws = FakeWS()
    session = server.StreamSession(ws)

    class FakeTorch:
        @staticmethod
        def no_grad():
            return contextlib.nullcontext()

    class Resettable:
        def __init__(self):
            self.reset_calls = 0
            self.replayed = 0

        def reset_streaming(self):
            self.reset_calls += 1

        def encode(self, _frame):
            self.replayed += 1
            return object()

        def step(self, _tokens):
            self.replayed += 1
            return None

    old_torch, old_mimi, old_lm_gen = server.MODEL._torch, server.MODEL._mimi, server.MODEL._lm_gen
    mimi = Resettable()
    lm_gen = Resettable()
    server.MODEL._torch, server.MODEL._mimi, server.MODEL._lm_gen = FakeTorch, mimi, lm_gen
    session._recent_frames.extend([object(), object(), object()])
    try:
        asyncio.run(_step(session, [TEXT, BOUNDARY] * 8))
        assert mimi.reset_calls >= 1
        assert lm_gen.reset_calls == mimi.reset_calls
        assert mimi.replayed == 2 * mimi.reset_calls
        assert lm_gen.replayed == 2 * lm_gen.reset_calls
        assert len(_segments(ws)) >= 2
    finally:
        server.STREAM_RESET_REPLAY_FRAMES = old_replay
        server.MODEL._torch, server.MODEL._mimi, server.MODEL._lm_gen = old_torch, old_mimi, old_lm_gen


def test_silence_commit_still_fires():
    """A sustained padding run after words still commits a segment (unchanged)."""
    _install_fake_model(max_segment_frames=48, silence_commit_frames=16)
    ws = FakeWS()
    session = server.StreamSession(ws)

    tokens = [TEXT, TEXT, TEXT] + [PAD] * 16

    asyncio.run(_step(session, tokens))

    assert len(_segments(ws)) == 1


def test_short_utterance_does_not_prematurely_commit():
    """Below the force-commit window and with no pause, nothing commits yet."""
    _install_fake_model(max_segment_frames=48)
    ws = FakeWS()
    session = server.StreamSession(ws)

    # 10 frames of speech, well under the 48-frame window, no pause.
    asyncio.run(_step(session, [TEXT, BOUNDARY] * 5))

    assert _segments(ws) == []
    assert any(m.get("type") == "partial" for m in ws.sent)


def test_processed_batch_signal_is_coalesced_and_monotonic():
    """The transport credit signal must be absolute and never compete with
    durable transcript delivery."""
    ws = FakeWS()
    session = server.StreamSession(ws)
    session.processed_batches = 3

    asyncio.run(session._drain_outbound())

    processed = [m for m in ws.sent if m.get("type") == "processed"]
    assert processed == [{"type": "processed", "processed_batches": 3}]
    asyncio.run(session._drain_outbound())
    assert [m for m in ws.sent if m.get("type") == "processed"] == processed


def test_processed_coverage_precedes_terminal_done():
    """The provider stops consuming after ``done``; final coverage must arrive first."""
    ws = FakeWS()
    session = server.StreamSession(ws)
    session.processed_batches = 1
    session.enqueue_done()

    asyncio.run(session._drain_outbound())

    assert [message["type"] for message in ws.sent] == ["processed", "done"]


def test_bare_websocket_cannot_reserve_decoder_admission():
    """The required start header precedes FIFO admission.

    This prevents a stale browser upgrade from occupying the one local decoder
    without ever declaring a stream or sending audio.
    """
    old_loaded = server.MODEL.loaded
    old_timeout = server.START_FRAME_TIMEOUT_S
    old_admission = server.MODEL.admission
    server.MODEL.loaded = True
    server.START_FRAME_TIMEOUT_S = 0.01
    server.MODEL.admission = server.FIFOAdmission(8, 30)
    ws = HandshakeWS([])
    try:
        asyncio.run(server.stream(ws))
    finally:
        admission = server.MODEL.admission
        server.MODEL.loaded = old_loaded
        server.START_FRAME_TIMEOUT_S = old_timeout
        server.MODEL.admission = old_admission

    assert ws.accepted and ws.closed
    assert ws.sent == [{
        "type": "error",
        "code": "start_timeout",
        "message": "start control frame was not received before admission timeout",
    }]
    assert not admission._waiters
    assert admission._active is None


def test_multi_frame_decode_cooperatively_yields_to_outbound_work():
    """A large PCM batch must not monopolize the asyncio loop long enough to
    starve WebSocket keepalive/outbound credit work."""
    previous_activity = server.MODEL.current_session_last_activity
    _install_fake_model(max_segment_frames=0)
    ws = FakeWS()
    session = server.StreamSession(ws)

    class FakeFramed:
        def __getitem__(self, _):
            return self

        def unsqueeze(self, _):
            return self

    class FakeTorch:
        @staticmethod
        def no_grad():
            return contextlib.nullcontext()

    class FakeMimi:
        @staticmethod
        def encode(_):
            return object()

    class FakeLM:
        @staticmethod
        def step(_):
            return None

    server.MODEL._torch = FakeTorch()
    server.MODEL._mimi = FakeMimi()
    server.MODEL._lm_gen = FakeLM()

    async def scenario():
        outbound_ran = False

        async def outbound():
            nonlocal outbound_ran
            await asyncio.sleep(0)
            outbound_ran = True

        task = asyncio.create_task(outbound())
        await session._run_frames(FakeFramed(), server.DECODE_YIELD_FRAMES)
        await task
        return outbound_ran

    try:
        assert asyncio.run(scenario()), "decode did not yield to outbound work"
    finally:
        server.MODEL.current_session_last_activity = previous_activity


def test_flush_commits_trailing_segment():
    """flush() must drain the delay tail and commit the pending segment."""
    _install_fake_model(max_segment_frames=0)  # disable force-commit for clarity
    ws = FakeWS()
    session = server.StreamSession(ws)

    async def scenario():
        # Simulate a pending, uncommitted segment (spoke, then stopped, no pause).
        await _step(session, [TEXT, TEXT])
        assert _segments(ws) == [], "should be pending, not yet committed"

        # Stub the model-dependent bits flush() touches so no torch is needed.
        session.remainder = None

        class _F:
            @staticmethod
            def zeros(shape, device=None):
                return ("zeros", shape)

        session_torch = _F()
        server.MODEL._torch = session_torch

        async def fake_run_frames(framed, n_frames):
            return None

        session._run_frames = fake_run_frames
        await session.flush()
        await session._drain_outbound()

    asyncio.run(scenario())

    assert len(_segments(ws)) == 1, "flush did not commit the trailing segment"


def test_disconnect_vs_end_divergence():
    """Without flush (disconnect), the trailing segment is NOT delivered; the
    end path's flush is what carries the tail. This encodes why a cold close
    loses the final words."""
    _install_fake_model(max_segment_frames=0)
    ws = FakeWS()
    session = server.StreamSession(ws)

    # Speak, then the client vanishes without sending {"type":"end"}.
    asyncio.run(_step(session, [TEXT, TEXT]))

    assert _segments(ws) == [], (
        "a trailing segment leaked without an explicit flush; the divergence "
        "between disconnect and end must be that only end/flush delivers the tail"
    )
