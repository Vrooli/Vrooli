import { describe, it, expect } from "vitest";

import { decideAutoStop } from "./autoStopDecision";
import type { ServerVadStateSnapshot } from "../useServerVadStateStore";

const STALE_MS = 250;
const NOW = 10_000;

function snapshot(over: Partial<ServerVadStateSnapshot> = {}): ServerVadStateSnapshot {
  return {
    voiced: false,
    silenceElapsedMs: 0,
    silenceTimeoutMs: 0,
    receivedAt: 0,
    tickSeq: 0,
    silenceTimedOut: false,
    ...over,
  };
}

describe("decideAutoStop", () => {
  it("stops with source=server when a fresh tick reaches threshold", () => {
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - 50,
        silenceElapsedMs: 2500,
        silenceTimeoutMs: 2500,
        tickSeq: 12,
      }),
      clientVadResult: null,
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "stop", source: "server" });
  });

  it("stops with source=server even when client says continue", () => {
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - 10,
        silenceElapsedMs: 1500,
        silenceTimeoutMs: 1500,
        tickSeq: 5,
      }),
      clientVadResult: null,
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict.kind).toBe("stop");
  });

  it("stale tick + clientVad=stop + no latch → client-fallback (belt-and-suspenders)", () => {
    // The server's threshold-crossing tick is supposed to set
    // silenceTimedOut, but if it was lost in transit (network hiccup, server
    // bug) the latch never fires and the wedge appears: tick goes stale, no
    // latch, session hangs. The client RMS VAD's independent "stop" verdict
    // breaks the dead zone. Both signals must agree (clientVadResult ===
    // "stop") — that prevents the client's aggressive VAD from firing
    // mid-utterance while a FRESH server tick is keeping the session alive.
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - 1000,
        silenceElapsedMs: 3000,
        silenceTimeoutMs: 1500,
        tickSeq: 7,
        silenceTimedOut: false,
      }),
      clientVadResult: "stop",
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "stop", source: "client-fallback" });
  });

  it("stale tick + clientVad=null → continue (do not fire without client agreement)", () => {
    // Without a client "stop" verdict, a stale tick alone must NOT trigger
    // a stop — that would re-introduce the original wedge in reverse.
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - 1000,
        silenceElapsedMs: 3000,
        silenceTimeoutMs: 1500,
        tickSeq: 7,
        silenceTimedOut: false,
      }),
      clientVadResult: null,
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "continue" });
  });

  it("falls back to client when no server tick ever arrived", () => {
    const verdict = decideAutoStop({
      serverVad: snapshot(), // receivedAt = 0
      clientVadResult: "stop",
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "stop", source: "client-fallback" });
  });

  it("continues when fresh server tick is under threshold even with client stop", () => {
    // Server says "still talking" (silence < timeout) — must override client.
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - 20,
        silenceElapsedMs: 600,
        silenceTimeoutMs: 1500,
        tickSeq: 9,
      }),
      clientVadResult: "stop",
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "continue" });
  });

  it("continues when nothing has happened", () => {
    expect(
      decideAutoStop({
        serverVad: snapshot(),
        clientVadResult: null,
        nowPerf: NOW,
        staleTickMs: STALE_MS,
      }),
    ).toEqual({ kind: "continue" });
  });

  it("treats silenceTimeoutMs=0 as 'config not hydrated' and skips server-path stop", () => {
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW,
        silenceElapsedMs: 5000,
        silenceTimeoutMs: 0,
        tickSeq: 1,
      }),
      clientVadResult: null,
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "continue" });
  });

  it("treats exact-equal tick age as still fresh (boundary)", () => {
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - STALE_MS,
        silenceElapsedMs: 2500,
        silenceTimeoutMs: 2500,
        tickSeq: 3,
      }),
      clientVadResult: null,
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "stop", source: "server" });
  });

  it("stops on a latched silenceTimedOut tick EVEN WHEN the tick is stale", () => {
    // The wedge regression: the server emits the threshold tick once (then goes
    // quiet after the cut). If the RAF loop misses the 250ms freshness window,
    // the float-compare branch never fires and the session hangs forever. The
    // sticky silenceTimedOut latch must stop regardless of tick age.
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - 5000, // very stale
        silenceElapsedMs: 1500,
        silenceTimeoutMs: 1500,
        tickSeq: 8,
        silenceTimedOut: true,
      }),
      clientVadResult: null,
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "stop", source: "server" });
  });

  it("stale below-timeout tick + clientVad=null continues (no client agreement)", () => {
    // Without a client "stop" verdict, a stale tick alone must NOT stop — only
    // the latch or a fresh at-timeout tick may stop in the server-led path.
    const verdict = decideAutoStop({
      serverVad: snapshot({
        receivedAt: NOW - 5000,
        silenceElapsedMs: 600,
        silenceTimeoutMs: 1500,
        tickSeq: 4,
        silenceTimedOut: false,
      }),
      clientVadResult: null,
      nowPerf: NOW,
      staleTickMs: STALE_MS,
    });
    expect(verdict).toEqual({ kind: "continue" });
  });

  it("non-stop client actions never trigger fallback stop", () => {
    for (const action of ["segment-boundary", "no-speech"] as const) {
      const verdict = decideAutoStop({
        serverVad: snapshot(),
        clientVadResult: action,
        nowPerf: NOW,
        staleTickMs: STALE_MS,
      });
      expect(verdict).toEqual({ kind: "continue" });
    }
  });
});
