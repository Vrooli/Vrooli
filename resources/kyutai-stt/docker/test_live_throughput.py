"""Opt-in real-resource regression probe for Kyutai streaming throughput.

Run only against the managed local resource:

    KYUTAI_STT_LIVE_PROBE=1 python3 -m pytest docker/test_live_throughput.py

The test sends thirty seconds of canonical PCM in browser-sized 100 ms frames.
It has a hard wall-clock ceiling so a deferred compiler, CPU fallback, or decoder
stall fails quickly instead of consuming an unbounded experiment budget.
"""

from __future__ import annotations

import asyncio
import json
import os
import time

import pytest


pytestmark = pytest.mark.skipif(
    os.environ.get("KYUTAI_STT_LIVE_PROBE") != "1",
    reason="set KYUTAI_STT_LIVE_PROBE=1 to probe the managed CUDA resource",
)


SAMPLE_RATE = 16_000
CHUNK_MS = 100
DURATION_SECONDS = 30
WALL_CLOCK_LIMIT_SECONDS = 90


async def _run_probe() -> tuple[float, int]:
    # Import only for the opt-in integration path; unit tests do not need a
    # host-level websocket dependency or a running GPU resource.
    import websockets

    endpoint = os.environ.get("KYUTAI_STT_WS_URL", "ws://localhost:8094/v1/stream")
    expected_batches = DURATION_SECONDS * 1000 // CHUNK_MS
    chunk = b"\0" * (SAMPLE_RATE * 2 * CHUNK_MS // 1000)
    started_at = time.monotonic()

    async with websockets.connect(endpoint, max_size=2**20, open_timeout=10) as ws:
        # The resource validates intent before allocating its one decoder slot;
        # ready is the permission to begin sending PCM, not permission to send
        # this control frame.
        await ws.send(json.dumps({"type": "start", "sample_rate": SAMPLE_RATE, "language": "en"}))
        while True:
            event = json.loads(await asyncio.wait_for(ws.recv(), timeout=10))
            if event["type"] == "ready":
                break
            if event["type"] in {"error", "rejected", "timed_out"}:
                raise AssertionError(f"admission failed: {event}")

        for _ in range(expected_batches):
            await ws.send(chunk)
        await ws.send(json.dumps({"type": "end"}))

        processed_batches = 0
        while True:
            remaining = WALL_CLOCK_LIMIT_SECONDS - (time.monotonic() - started_at)
            assert remaining > 0, "Kyutai probe exceeded its 90-second wall-clock budget"
            event = json.loads(await asyncio.wait_for(ws.recv(), timeout=remaining))
            if event["type"] == "processed":
                processed_batches = event["processed_batches"]
            if event["type"] == "error":
                raise AssertionError(f"Kyutai stream errored: {event}")
            if event["type"] == "done":
                return time.monotonic() - started_at, processed_batches


def test_thirty_second_100ms_stream_is_complete_within_budget() -> None:
    elapsed, processed_batches = asyncio.run(_run_probe())
    expected_batches = DURATION_SECONDS * 1000 // CHUNK_MS
    assert processed_batches == expected_batches
    assert elapsed <= WALL_CLOCK_LIMIT_SECONDS
