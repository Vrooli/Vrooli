"""Phase 1 resource red oracle for the streaming-STT backpressure-wedge plan.

Root cause (server.py): the decode loop emits socket frames inline —
``_step_token`` awaits ``_emit_partial`` / ``_commit_segment``, both of which
await ``ws.send_text``. When the consumer stops reading, ``send_text`` back-
pressures and the decode loop STOPS STEPPING (``frames_consumed`` freezes), so
audio spoken during the stall is never decoded — total loss, not a tail.

Desired behaviour (Phase 3 fix): decode is decoupled from send. The decode loop
enqueues events and keeps stepping regardless of consumer speed; a send worker
drains the queue (coalescing/dropping partials under pressure, never dropping
durable ``segment``/``done`` events).

This oracle drives the real decode state machine with a consumer whose
``send_text`` is gated (blocked). It asserts the decode loop advances
``frames_consumed`` WHILE the consumer is blocked. It FAILS against unmodified
code (decode freezes at the first emit) and PASSES once decode is decoupled from
send.

Run: ``python3 -m pytest resources/kyutai-stt/tests/test_server_backpressure.py``
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
import time

sys.path.insert(0, os.path.dirname(__file__))
import server  # noqa: E402

# Token convention (mirrors test_server.py): 3 = padding, 0 = word boundary,
# id > 3 = real text.
TEXT = 10
BOUNDARY = 0


class FakeTokenizer:
    def decode(self, ids):
        return " ".join(f"w{i}" for i in ids)


class BlockingWS:
    """A consumer whose send_text blocks until the test opens the gate.

    Models the browser/relay consumer that has stopped reading: every send
    back-pressures. Opening the gate lets buffered sends drain.
    """

    def __init__(self) -> None:
        self.sent: list = []
        self.gate = asyncio.Event()  # cleared = block all sends

    async def send_text(self, text: str) -> None:
        await self.gate.wait()
        self.sent.append(json.loads(text))


def _install_fake_model(max_segment_frames: int = 48, silence_commit_frames: int = 16):
    m = server.MODEL
    m.text_padding_id = 3
    m.audio_delay_seconds = 0.5
    m._model_sample_rate = 24000
    m._frame_size = 1920  # frame_rate = 12.5 Hz
    m._tokenizer = FakeTokenizer()
    server.MAX_SEGMENT_FRAMES = max_segment_frames
    server.SILENCE_COMMIT_FRAMES = silence_commit_frames


def test_decode_loop_advances_while_consumer_blocked():
    """The decode loop must keep stepping while the consumer's send back-
    pressures — a slow/stalled consumer must never freeze decode."""
    _install_fake_model(max_segment_frames=48)

    async def scenario():
        ws = BlockingWS()  # gate closed: the send worker's sends block
        session = server.StreamSession(ws)
        session.start_sender()  # worker will block on send once decode enqueues

        # 200 frames of continuous speech: force-commits fire, partials stream.
        tokens = [TEXT, BOUNDARY] * 100

        async def drive():
            for tok in tokens:
                session.frames_consumed += 1
                await session._step_token(tok)

        task = asyncio.create_task(drive())
        # Yield generously so decode runs as far as it can with the consumer
        # (the send worker) blocked.
        for _ in range(20):
            await asyncio.sleep(0.005)
        consumed_while_blocked = session.frames_consumed

        # Release the consumer and let decode + the drain-then-close finish.
        ws.gate.set()
        await asyncio.wait_for(task, timeout=5.0)
        await session.close()
        return consumed_while_blocked, session.frames_consumed, ws.sent

    consumed_while_blocked, total_consumed, sent = asyncio.run(scenario())

    assert total_consumed == 200, "sanity: all frames eventually stepped"
    assert consumed_while_blocked >= 150, (
        "decode loop froze under consumer backpressure: only "
        f"{consumed_while_blocked}/200 frames stepped while the consumer's "
        "send was blocked. Decode must be decoupled from send so a slow/stalled "
        "consumer cannot stop audio from being decoded."
    )
    # Durables must survive: the continuous-speech force-commits are delivered.
    segs = [m for m in sent if m.get("type") == "segment"]
    assert len(segs) >= 2, f"expected force-committed durable segments, got {len(segs)}"


def test_committed_segments_never_dropped_under_pressure():
    """Every committed ``segment`` the decode loop produced must be delivered,
    in commit order, with none dropped — even though partials may be. Guards the
    Phase 3 coalescing fix against ever dropping a durable event."""
    _install_fake_model(max_segment_frames=48)

    async def scenario():
        ws = BlockingWS()
        session = server.StreamSession(ws)
        session.start_sender()
        tokens = [TEXT, BOUNDARY] * 100

        # Spy on the durable-commit path to record ground truth independently
        # of what actually reaches the wire.
        committed: list = []
        orig_commit = session._commit_segment

        async def spy_commit():
            text_before = session._decode(session.seg_tokens)
            await orig_commit()
            if text_before:
                committed.append(text_before)

        session._commit_segment = spy_commit

        async def drive():
            for tok in tokens:
                session.frames_consumed += 1
                await session._step_token(tok)

        task = asyncio.create_task(drive())
        for _ in range(20):
            await asyncio.sleep(0.005)
        ws.gate.set()
        await asyncio.wait_for(task, timeout=5.0)
        await session.close()
        return committed, ws.sent

    committed, sent = asyncio.run(scenario())
    delivered = [m["text"] for m in sent if m.get("type") == "segment"]
    assert len(committed) >= 2, f"expected multiple force-commits, got {len(committed)}"
    assert delivered == committed, (
        "durable segments were dropped or reordered under consumer backpressure: "
        f"delivered={delivered!r} committed={committed!r}"
    )


def test_teardown_completes_when_consumer_permanently_blocked():
    """Drain-then-close is bounded: a consumer that never reads cannot hang
    teardown past SEND_DRAIN_TIMEOUT_S."""
    _install_fake_model()
    orig = server.SEND_DRAIN_TIMEOUT_S
    server.SEND_DRAIN_TIMEOUT_S = 0.2

    async def scenario():
        ws = BlockingWS()  # gate never opened: every send blocks forever
        session = server.StreamSession(ws)
        session.start_sender()
        session.enqueue_done()  # a durable the worker will try (and fail) to send
        # close() must return within the bounded window despite the dead consumer.
        await asyncio.wait_for(session.close(), timeout=2.0)

    try:
        asyncio.run(scenario())
    finally:
        server.SEND_DRAIN_TIMEOUT_S = orig


def test_wedged_session_is_reaped_so_next_recording_proceeds():
    """A wedged prior session holding the single-session MODEL.lock is reaped
    within LOCK_ACQUIRE_TIMEOUT_S so the next recording is not starved."""
    orig_timeout = server.LOCK_ACQUIRE_TIMEOUT_S
    server.LOCK_ACQUIRE_TIMEOUT_S = 0.1

    async def scenario():
        server.MODEL.lock = asyncio.Lock()
        server.MODEL.current_session_task = None
        holds = asyncio.Event()

        async def wedged_holder():
            await server.MODEL.lock.acquire()
            server.MODEL.current_session_task = asyncio.current_task()
            holds.set()
            try:
                await asyncio.sleep(3600)  # never releases on its own
            finally:
                server.MODEL.lock.release()

        holder = asyncio.create_task(wedged_holder())
        await holds.wait()

        # A new session acquires: must reap the wedged holder, not wait forever.
        await asyncio.wait_for(server._acquire_model_lock_or_reap(), timeout=2.0)
        acquired = server.MODEL.lock.locked()
        owns = server.MODEL.current_session_task is asyncio.current_task()
        server.MODEL.current_session_task = None
        server.MODEL.lock.release()
        return acquired, owns, holder.done()

    try:
        acquired, owns, holder_done = asyncio.run(scenario())
    finally:
        server.LOCK_ACQUIRE_TIMEOUT_S = orig_timeout

    assert acquired, "the next session must acquire the model lock after reaping"
    assert owns, "the reaping session must become the lock owner"
    assert holder_done, "the wedged prior session must be reaped (cancelled)"


def test_active_session_is_not_reaped_by_competing_connection():
    """A lock waiter must not reap a holder that is actively decoding.

    This pins the 2026-07-09 kill chain: a second bare connection waited
    LOCK_ACQUIRE_TIMEOUT_S and then cancelled the active dictation solely
    because it was waiting on the single-session lock. Desired behavior is
    oldest-wins: active holder survives; newcomer is rejected/busy.
    """
    orig_timeout = server.LOCK_ACQUIRE_TIMEOUT_S
    server.LOCK_ACQUIRE_TIMEOUT_S = 0.05

    async def scenario():
        server.MODEL.lock = asyncio.Lock()
        server.MODEL.current_session_task = None
        holds = asyncio.Event()
        keep_running = {"value": True}

        async def active_holder():
            await server.MODEL.lock.acquire()
            server.MODEL.current_session_task = asyncio.current_task()
            holds.set()
            try:
                while keep_running["value"]:
                    # Phase-2 implementation reads this activity timestamp (or
                    # its StreamSession-backed successor) before deciding
                    # whether a holder is genuinely wedged.
                    server.MODEL.current_session_last_activity = time.monotonic()
                    await asyncio.sleep(0.01)
            finally:
                if server.MODEL.lock.locked():
                    server.MODEL.lock.release()

        holder = asyncio.create_task(active_holder())
        await holds.wait()

        contender_acquired = False
        try:
            await asyncio.wait_for(server._acquire_model_lock_or_reap(), timeout=1.0)
            contender_acquired = server.MODEL.lock.locked()
            if contender_acquired:
                server.MODEL.current_session_task = None
                server.MODEL.lock.release()
        except Exception:  # noqa: BLE001 - desired implementations may reject/busy.
            pass

        holder_was_reaped = holder.done()
        keep_running["value"] = False
        holder.cancel()
        try:
            await holder
        except asyncio.CancelledError:
            pass
        return holder_was_reaped, contender_acquired

    try:
        holder_done, contender_acquired = asyncio.run(scenario())
    finally:
        server.LOCK_ACQUIRE_TIMEOUT_S = orig_timeout

    assert not holder_done, (
        "active dictation holder was reaped by a competing connection; lock "
        "timeout alone is not a wedge signal"
    )
    assert not contender_acquired, "newcomer must be rejected/busy, not steal the active holder's lock"


def test_fifo_admission_orders_waiters_and_supports_cancellation():
    async def scenario():
        admission = server.FIFOAdmission(max_depth=3, max_wait_s=1)
        first, first_position = await admission.acquire()
        queued_positions: list[int] = []
        async def record(position: int) -> None:
            queued_positions.append(position)
        second_task = asyncio.create_task(admission.acquire(record))
        third_task = asyncio.create_task(admission.acquire(record))
        await asyncio.sleep(0)
        second_task.cancel()
        try:
            await second_task
        except asyncio.CancelledError:
            pass
        await admission.release(first)
        third, third_position = await third_task
        await admission.release(third)
        return first_position, queued_positions, third_position, second_task.cancelled()

    # Do not infer FIFO from wall-clock waits: the queue's explicit ticket
    # order is the contract. The cancelled middle waiter must not block third.
    first_position, queued_positions, third_position, second_cancelled = asyncio.run(scenario())
    assert first_position == 1
    assert queued_positions == [2, 3]
    assert third_position == 3
    assert second_cancelled
