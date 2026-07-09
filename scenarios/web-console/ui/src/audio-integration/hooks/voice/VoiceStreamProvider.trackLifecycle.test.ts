/**
 * Mic-source lifecycle handling. A mid-capture failure of the audio SOURCE
 * (the OS/browser muting or ending the track, or the encoder erroring) used to
 * be invisible: no `ended`/`error` handler existed, so the analyser silently
 * read silence and the client RMS VAD auto-stopped mid-utterance — the mic
 * "just stopping" with no cause in the logs.
 *
 * Contract this pins:
 *  1. A track `mute` mid-capture surfaces a `mic_muted` status but does NOT end
 *     the turn (mutes are frequently transient).
 *  2. A track `ended` mid-capture is terminal: it surfaces `mic_track_ended`,
 *     closes the WS, and RECOVERS the turn's retained audio via HTTP fallback
 *     (rather than losing the tail to a silent VAD stop).
 *  3. Once the turn has already resolved (finalReceived) or was intentionally
 *     stopped, a late `ended` event is a no-op.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../../api/voice", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/voice")>();
  return { ...actual, transcribeAudioWithRetry: vi.fn(async () => "recovered tail") };
});

import { transcribeAudioWithRetry } from "../../api/voice";
import { VoiceStreamProvider } from "./VoiceStreamProvider";

class FakeWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  static CONNECTING = 0;
  static instances: FakeWebSocket[] = [];
  readyState = FakeWebSocket.OPEN;
  url: string;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  send = vi.fn<(data: unknown) => void>();
  close = vi.fn(() => {
    this.readyState = FakeWebSocket.CLOSED;
  });
  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
    queueMicrotask(() => {
      this.readyState = FakeWebSocket.OPEN;
      this.onopen?.();
    });
  }
}

class FakeMediaRecorder {
  static instances: FakeMediaRecorder[] = [];
  static isTypeSupported = vi.fn(() => true);
  state: "inactive" | "recording" = "inactive";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;
  onerror: ((e: unknown) => void) | null = null;
  start = vi.fn(() => {
    this.state = "recording";
  });
  stop = vi.fn(() => {
    this.state = "inactive";
    this.onstop?.();
  });
  constructor(_stream: MediaStream, _opts?: unknown) {
    FakeMediaRecorder.instances.push(this);
  }
}

/** A mic track that records addEventListener handlers so the test can dispatch
 *  real `mute`/`unmute`/`ended` source events. */
class FakeTrack {
  kind = "audio";
  readyState: "live" | "ended" = "live";
  muted = false;
  private listeners: Record<string, Array<() => void>> = {};
  stop = vi.fn(() => {
    this.readyState = "ended";
  });
  addEventListener(type: string, cb: () => void): void {
    (this.listeners[type] ??= []).push(cb);
  }
  dispatch(type: string): void {
    for (const cb of this.listeners[type] ?? []) cb();
  }
}

function fakeBlob(size: number): Blob {
  return {
    size,
    type: "audio/webm",
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(size)),
  } as unknown as Blob;
}

let currentTrack: FakeTrack;
function fakeStream(): MediaStream {
  currentTrack = new FakeTrack();
  return {
    active: true,
    getTracks: () => [currentTrack],
    getAudioTracks: () => [currentTrack],
  } as unknown as MediaStream;
}

describe("VoiceStreamProvider mic-source lifecycle", () => {
  let provider: VoiceStreamProvider;
  let infoSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    FakeWebSocket.instances = [];
    FakeMediaRecorder.instances = [];
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    (globalThis as unknown as { MediaRecorder: typeof FakeMediaRecorder }).MediaRecorder = FakeMediaRecorder;
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => fakeStream()) },
    });
    infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.mocked(transcribeAudioWithRetry).mockClear();
    provider = new VoiceStreamProvider();
  });

  afterEach(() => {
    provider.dispose();
    infoSpy.mockRestore();
    warnSpy.mockRestore();
  });

  async function startAndOpenWs(): Promise<{ ws: FakeWebSocket; rec: FakeMediaRecorder }> {
    await provider.start();
    await Promise.resolve();
    const ws = FakeWebSocket.instances.at(-1);
    const rec = FakeMediaRecorder.instances.at(-1);
    if (!ws || !rec) throw new Error("expected WebSocket and MediaRecorder instances to exist");
    return { ws, rec };
  }

  it("surfaces a mic_muted status on a mid-capture mute without ending the turn", async () => {
    await startAndOpenWs();
    const statuses: string[] = [];
    provider.onStatus = ({ code }) => statuses.push(code);
    const onResult = vi.fn();
    const onError = vi.fn();
    provider.onResult = onResult;
    provider.onError = onError;

    currentTrack.muted = true;
    currentTrack.dispatch("mute");

    expect(statuses).toContain("mic_muted");
    // A mute is transient — the turn must NOT be resolved or errored here.
    expect(onResult).not.toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it("recovers retained audio via HTTP fallback when the track ENDS mid-capture", async () => {
    const { ws, rec } = await startAndOpenWs();
    // Retain a chunk so the HTTP fallback has audio to transcribe.
    rec.ondataavailable?.({ data: fakeBlob(2048) });
    await Promise.resolve();

    const statuses: string[] = [];
    provider.onStatus = ({ code }) => statuses.push(code);
    const onResult = vi.fn();
    provider.onResult = onResult;

    currentTrack.readyState = "ended";
    currentTrack.dispatch("ended");
    // Let the transcribeAudioWithRetry().then microtask resolve.
    await Promise.resolve();
    await Promise.resolve();

    expect(statuses).toContain("mic_track_ended");
    expect(ws.close).toHaveBeenCalled();
    expect(transcribeAudioWithRetry).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenCalledWith("recovered tail");
  });

  it("sends audio chunks in recording order as raw Blobs (header chunk first)", async () => {
    // Regression: the WebM/EBML header lives in the FIRST MediaRecorder chunk;
    // the backend's codec sniff requires it to arrive first. The old
    // `e.data.arrayBuffer().then(send)` path used independent async promises that
    // could resolve out of order (the header chunk is largest → slowest to read),
    // putting a headerless chunk on the wire first → "could not determine audio
    // codec". Sending Blobs synchronously in ondataavailable order fixes it.
    const { ws, rec } = await startAndOpenWs();
    const header = fakeBlob(4096); // largest — would resolve last under the old race
    const c2 = fakeBlob(64);
    const c3 = fakeBlob(64);
    rec.ondataavailable?.({ data: header });
    rec.ondataavailable?.({ data: c2 });
    rec.ondataavailable?.({ data: c3 });

    // Sent immediately, in order, as the exact Blobs (not reordered ArrayBuffers).
    expect(ws.send.mock.calls.map((c) => c[0])).toEqual([header, c2, c3]);
  });

  it("treats a mid-recording final as backend loss and routes through reconnect", async () => {
    const { ws, rec } = await startAndOpenWs();
    rec.ondataavailable?.({ data: fakeBlob(2048) });
    const onResult = vi.fn();
    provider.onResult = onResult;
    vi.useFakeTimers();
    try {
      ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "" }) });
      ws.onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);

      expect(onResult).not.toHaveBeenCalled();
      expect(FakeWebSocket.instances.length).toBeGreaterThan(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("ignores a late track-ended event once the final has already arrived", async () => {
    const { ws } = await startAndOpenWs();
    // Deliver a terminal final so the turn is resolved.
    ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "all done" }) });
    const onResult = vi.fn();
    provider.onResult = onResult;
    vi.mocked(transcribeAudioWithRetry).mockClear();

    currentTrack.dispatch("ended");
    await Promise.resolve();

    // No second resolution, no HTTP fallback — the ended event is a no-op.
    expect(onResult).not.toHaveBeenCalled();
    expect(transcribeAudioWithRetry).not.toHaveBeenCalled();
  });
});
