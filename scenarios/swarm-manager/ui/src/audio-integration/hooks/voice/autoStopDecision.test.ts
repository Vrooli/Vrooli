import { describe, it, expect } from "vitest";

import { decideAutoStop, decideAutoStopRing } from "./autoStopDecision";
import type { ServerVadStateSnapshot } from "../useServerVadStateStore";
import type { VoiceActivitySnapshot } from "./types";

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

function activity(over: Partial<VoiceActivitySnapshot> = {}): VoiceActivitySnapshot {
  return {
    phase: "silence",
    audioLevel: 0,
    rms: 0,
    speechThreshold: 0.06,
    silenceThreshold: 0.02,
    silenceElapsedMs: 600,
    silenceTimeoutMs: 1200,
    autoStopProgress: 0.5,
    autoStopVisible: true,
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

describe("decideAutoStopRing", () => {
  it("hides the ring when not recording", () => {
    expect(decideAutoStopRing({
      isRecording: false,
      serverVad: snapshot({
        receivedAt: NOW - 10,
        silenceElapsedMs: 1200,
        silenceTimeoutMs: 1200,
      }),
      voiceActivity: activity(),
      nowPerf: NOW,
      staleTickMs: STALE_MS,
      visualGraceMs: 300,
    })).toEqual({ visible: false, progress: 0 });
  });

  it("uses fresh server silence progress before client activity", () => {
    const ring = decideAutoStopRing({
      isRecording: true,
      serverVad: snapshot({
        voiced: false,
        receivedAt: NOW - 100,
        silenceElapsedMs: 500,
        silenceTimeoutMs: 1200,
      }),
      voiceActivity: activity({ phase: "speech", autoStopProgress: 0, autoStopVisible: false }),
      nowPerf: NOW,
      staleTickMs: STALE_MS,
      visualGraceMs: 300,
    });
    expect(ring.visible).toBe(true);
    expect(ring.progress).toBeCloseTo(0.5, 5);
  });

  it("keeps a stale latched server timeout visible and complete", () => {
    expect(decideAutoStopRing({
      isRecording: true,
      serverVad: snapshot({
        receivedAt: NOW - 5000,
        silenceElapsedMs: 1200,
        silenceTimeoutMs: 1200,
        silenceTimedOut: true,
      }),
      voiceActivity: activity({ phase: "speech", autoStopProgress: 0, autoStopVisible: false }),
      nowPerf: NOW,
      staleTickMs: STALE_MS,
      visualGraceMs: 300,
    })).toEqual({ visible: true, progress: 1 });
  });

  it("falls back to client activity when the server tick is stale and not latched", () => {
    expect(decideAutoStopRing({
      isRecording: true,
      serverVad: snapshot({
        receivedAt: NOW - 1000,
        silenceElapsedMs: 1100,
        silenceTimeoutMs: 1200,
        silenceTimedOut: false,
      }),
      voiceActivity: activity({ autoStopProgress: 0.25 }),
      nowPerf: NOW,
      staleTickMs: STALE_MS,
      visualGraceMs: 300,
    })).toEqual({ visible: true, progress: 0.25 });
  });

  it("hides the ring when neither server nor client has a usable timeout", () => {
    expect(decideAutoStopRing({
      isRecording: true,
      serverVad: snapshot({
        receivedAt: NOW - 10,
        silenceElapsedMs: 500,
        silenceTimeoutMs: 0,
      }),
      voiceActivity: activity({ silenceTimeoutMs: 0, autoStopVisible: true }),
      nowPerf: NOW,
      staleTickMs: STALE_MS,
      visualGraceMs: 300,
    })).toEqual({ visible: false, progress: 0 });
  });
});
