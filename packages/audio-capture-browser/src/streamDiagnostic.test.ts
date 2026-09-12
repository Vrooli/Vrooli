import { afterEach, describe, expect, it } from "vitest";
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
  state: "recording",
  capturedSequence: 4,
  capturedSamples: 6400,
  sentSequence: 3,
  processedSequence: 2,
  retainedBytes: 3200,
  firstPartialLatencyMs: null,
  committedTextLagMs: null,
  doneSent: false,
  statusCodes: ["stream_connected"],
  errorCodes: [],
  events: [{ atMs: 1, kind: "status", code: "stream_connected" }],
};

describe("stream diagnostic telemetry", () => {
  afterEach(() => _resetStreamDiagnosticTelemetryForTesting());

  it("publishes a bounded metadata snapshot for browser automation", () => {
    publishStreamDiagnostic(diagnostic);

    const telemetry = readStreamDiagnosticTelemetry();
    expect(telemetry?.latest).toEqual(diagnostic);
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

    expect(received).toEqual(["session-1"]);
    expect(readStreamDiagnosticTelemetry()?.latest.statusCodes).toEqual(["stream_connected"]);
  });

  it("records capture, retention, and timing observations without transcript data", () => {
    const recorder = new StreamDiagnosticRecorder("session-2", 0, "reduced");
    recorder.captureStarted();
    recorder.capturedSamples(1600n);
    recorder.captureObserved();
    recorder.retained(3200);
    recorder.partial();
    recorder.committed();
    const value = recorder.read();

    expect(value.capturedSamples).toBe(1600);
    expect(value.retainedBytes).toBe(3200);
    expect(value.firstPartialLatencyMs).not.toBeNull();
    expect(value.committedTextLagMs).not.toBeNull();
    expect(JSON.stringify(value)).not.toContain("transcript");
  });
});
