import { afterEach, describe, expect, it, vi } from "vitest";
import {
  _resetStreamDiagnosticTelemetryForTesting,
  publishStreamDiagnostic,
  readStreamDiagnosticTelemetry,
  STREAM_DIAGNOSTIC_GLOBAL,
  StreamDiagnosticRecorder,
  subscribeStreamDiagnosticTelemetry,
  type StreamTurnDiagnostic,
} from "./streamDiagnostic";

const diagnostic: StreamTurnDiagnostic = {
  schemaVersion: 1,
  sessionId: "session-1",
  generation: 2,
  protocolVersion: 2,
  durability: "full",
  streamSampleRateHz: 16_000,
  state: "recording",
  capturedSequence: 4,
  capturedSamples: 6400,
  captureCallbackCount: 0,
  captureCallbackMaxCostRatio: null,
  sentSequence: 3,
  processedSequence: 2,
  committedSamples: null,
  retainedBytes: 3200,
  firstPartialLatencyMs: null,
  visibleUpdateGapsMs: [],
  visibleRevisionCount: 0,
  committedTextLagMs: null,
  doneSent: false,
  statusCodes: ["stream_connected"],
  errorCodes: [],
  events: [{ atMs: 1, kind: "status", code: "stream_connected" }],
};

describe("stream diagnostic telemetry", () => {
  afterEach(() => _resetStreamDiagnosticTelemetryForTesting());

  it("publishes a bounded metadata snapshot for browser automation", () => {
    const beforePublish = Date.now();
    publishStreamDiagnostic(diagnostic);

    const telemetry = readStreamDiagnosticTelemetry();
    expect(telemetry?.latest).toEqual(diagnostic);
    expect(telemetry?.updatedAtMs).toBeGreaterThanOrEqual(beforePublish);
    expect(telemetry?.updatedAtMs).toBeLessThanOrEqual(Date.now());
    expect((globalThis as typeof globalThis & { [STREAM_DIAGNOSTIC_GLOBAL]?: unknown })[
      STREAM_DIAGNOSTIC_GLOBAL
    ]).toMatchObject({ schemaVersion: 1, latest: diagnostic });
  });

  it("notifies subscribers and does not expose mutable recorder state", () => {
    const received: string[] = [];
    const unsubscribe = subscribeStreamDiagnosticTelemetry((value) => received.push(value.latest.sessionId));
    publishStreamDiagnostic(diagnostic);
    unsubscribe();
    diagnostic.statusCodes.push("mutated-after-publish");
    diagnostic.visibleUpdateGapsMs.push(999);

    expect(received).toEqual(["session-1"]);
    expect(readStreamDiagnosticTelemetry()?.latest.statusCodes).toEqual(["stream_connected"]);
    expect(readStreamDiagnosticTelemetry()?.latest.visibleUpdateGapsMs).toEqual([]);
  });

  it("records capture, retention, and timing observations without transcript data", () => {
    const recorder = new StreamDiagnosticRecorder("session-2", 0, "reduced");
    recorder.captureStarted();
    recorder.capturedSamples(1600n);
    recorder.captureObserved();
    recorder.signalObserved();
    recorder.retained(3200);
    recorder.partial();
    recorder.committed(800);
    const value = recorder.read();

    expect(value.capturedSamples).toBe(1600);
    expect(value.retainedBytes).toBe(3200);
    expect(value.firstPartialLatencyMs).not.toBeNull();
    expect(value).toMatchObject({ committedSamples: 800, committedTextLagMs: 50 });
    expect(JSON.stringify(value)).not.toContain("transcript");
  });

  it("measures user-action to simultaneous capture-and-stream readiness", () => {
    let nowMs = 100;
    const recorder = new StreamDiagnosticRecorder("session-ready", 0, "reduced", () => nowMs);
    recorder.startRequested(80);
    nowMs = 120;
    recorder.captureStarted();
    nowMs = 450;
    recorder.captureReady();
    recorder.captureReady();

    expect(recorder.read()).toMatchObject({
      startRequestedAtMs: 80,
      captureStartedAtMs: 120,
      captureReadyAtMs: 450,
      startToCaptureReadyMs: 370,
    });
    expect(recorder.read().events.map(({ code }) => code)).toContain("capture_ready");
  });

  it("measures committed audio backlog from sample coverage", () => {
    const recorder = new StreamDiagnosticRecorder("session-backlog");
    recorder.captureRate(48_000);
    recorder.capturedSamples(3_200n);
    recorder.committed(1_600);
    expect(recorder.read()).toMatchObject({
      captureSampleRate: 48_000,
      streamSampleRateHz: 16_000,
      committedSamples: 1_600,
      committedTextLagMs: 100,
    });

    recorder.capturedSamples(6_400n);
    expect(recorder.read().committedTextLagMs).toBe(300);
  });

  it("exports bounded capture callback processing cost against source audio duration", () => {
    const recorder = new StreamDiagnosticRecorder("session-cost");
    recorder.captureCallbackCost(2, 10);
    recorder.captureCallbackCost(8, 10);
    recorder.captureCallbackCost(Number.NaN, 10);
    expect(recorder.read()).toMatchObject({ captureCallbackCount: 2, captureCallbackMaxCostRatio: 0.8 });
  });

  it("retains only an observed provider tier in the metadata snapshot", () => {
    const recorder = new StreamDiagnosticRecorder("session-route", 0, "reduced");
    recorder.providerIdentity("provider", "model", "byok");
    expect(recorder.read()).toMatchObject({ providerId: "provider", modelId: "model", providerTier: "byok" });
    recorder.providerIdentity(undefined, undefined, "unrecognized");
    expect(recorder.read().providerTier).toBe("byok");
  });

  it("anchors first-partial latency to the first voiced sample", () => {
    const recorder = new StreamDiagnosticRecorder("session-3", 0, "reduced");
    recorder.captureStarted();
    recorder.captureObserved();
    const beforeVoice = recorder.read().firstPartialLatencyMs;
    expect(beforeVoice).toBeNull();
    recorder.signalObserved();
    recorder.partial();
    expect(recorder.read().firstPartialLatencyMs).not.toBeNull();
  });

  it("separates first-partial latency from captured speech context", () => {
    let nowMs = 100;
    const recorder = new StreamDiagnosticRecorder("session-context", 0, "reduced", () => nowMs);
    recorder.captureRate(48_000);
    recorder.signalObserved(0, 800n);
    recorder.capturedSamples(16_800n);
    nowMs = 1_300;
    recorder.partial();

    expect(recorder.read()).toMatchObject({
      firstSignalSample: 800,
      firstPartialCapturedSamples: 16_800,
      firstPartialLatencyMs: 1_200,
      firstPartialAvailableAudioMs: 1_000,
      firstPartialExcessLatencyMs: 200,
    });
  });

  it("measures gaps only after transcript revisions render and exports metadata only", () => {
    let nowMs = 100;
    const recorder = new StreamDiagnosticRecorder("session-visible", 0, "reduced", () => nowMs);
    recorder.visibleRevisionRendered();
    nowMs = 420;
    recorder.visibleRevisionRendered();
    nowMs = 1_190;
    recorder.visibleRevisionRendered();
    const value = recorder.read();
    expect(value).toMatchObject({ visibleRevisionCount: 3, visibleUpdateGapsMs: [320, 770] });
    expect(JSON.stringify(value)).not.toContain("hello");
  });

  it("leaves speech latency unknown when no signal threshold was observed", () => {
    const recorder = new StreamDiagnosticRecorder("session-silent", 0, "reduced");
    recorder.captureObserved();
    recorder.partial();
    expect(recorder.read().firstPartialLatencyMs).toBeNull();
  });

  it("measures stop to a real streaming final and leaves absent finals unknown", () => {
    let nowMs = 100;
    const recorder = new StreamDiagnosticRecorder("session-final-tail", 0, "reduced", () => nowMs);
    recorder.streamingFinalReceived();
    expect(recorder.read()).toMatchObject({ stopRequestedAtMs: null, finalTailAtMs: null, stopToFinalTailMs: null });

    recorder.stopRequested();
    nowMs = 220;
    recorder.streamingFinalReceived();
    nowMs = 500;
    recorder.streamingFinalReceived();
    expect(recorder.read()).toMatchObject({ stopRequestedAtMs: 100, finalTailAtMs: 220, stopToFinalTailMs: 120 });
    expect(recorder.read().events.map(({ code }) => code)).toContain("stream_final_tail");

    const missing = new StreamDiagnosticRecorder("session-missing-final", 0, "reduced", () => nowMs);
    missing.stopRequested();
    expect(missing.read().stopToFinalTailMs).toBeNull();
  });

  it("backs speech onset into the capture block and ignores wall-clock jumps", () => {
    const times = [100, 150, 200, 300, 400, 550];
    const now = vi.fn(() => times.shift() ?? 550);
    const wallClock = vi.spyOn(Date, "now").mockReturnValue(1);
    const recorder = new StreamDiagnosticRecorder("session-4", 0, "reduced", now);

    recorder.captureStarted();
    recorder.captureObserved();
    // The threshold crossing was 50 ms before the end of its capture block.
    recorder.signalObserved(50);
    recorder.partial();

    expect(recorder.read()).toMatchObject({
      firstSignalAtMs: 350,
      firstPartialAtMs: 550,
      firstPartialLatencyMs: 200,
    });
    expect(wallClock).not.toHaveBeenCalled();
    expect(now).toHaveBeenCalledTimes(6);
  });
});
