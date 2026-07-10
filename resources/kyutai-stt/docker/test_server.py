"""Unit tests for the Kyutai STT streaming server's commit/segmentation logic.

These are the first tests for docker/server.py. They exercise the pure decode
and commit state machine (StreamSession._step_token / _commit_segment / flush)
WITHOUT loading torch or the real moshi model: the heavy model is stubbed so we
can drive the token stream directly and assert when durable ``segment`` events
are emitted.

Run: ``python -m pytest resources/kyutai-stt/docker/test_server.py``

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
import os
import sys

# Import the server module without its torch/moshi deps (they are imported
# lazily inside Model.load(), which we never call here).
sys.path.insert(0, os.path.dirname(__file__))
import server  # noqa: E402


class FakeWS:
    """Minimal WebSocket capturing every JSON frame the session emits."""

    def __init__(self) -> None:
        self.sent: list = []

    async def send_text(self, text: str) -> None:
        import json

        self.sent.append(json.loads(text))


class FakeTokenizer:
    """Detokenize by mapping each id to a token word; ids are already filtered
    to real text ids (> padding) by StreamSession._decode."""

    def decode(self, ids):
        return " ".join(f"w{i}" for i in ids)


def _install_fake_model(max_segment_frames: int, silence_commit_frames: int = 16):
    """Stub the module-global MODEL so segmentation logic runs without torch."""
    m = server.MODEL
    m.text_padding_id = 3
    m.audio_delay_seconds = 0.5
    m._model_sample_rate = 24000
    m._frame_size = 1920  # frame_rate = 12.5 Hz
    m._tokenizer = FakeTokenizer()
    server.MAX_SEGMENT_FRAMES = max_segment_frames
    server.SILENCE_COMMIT_FRAMES = silence_commit_frames


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
