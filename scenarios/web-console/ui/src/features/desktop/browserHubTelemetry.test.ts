import { describe, expect, it } from "vitest";
import { BrowserHubTelemetry } from "./browserHubTelemetry";

describe("BrowserHubTelemetry", () => {
  it("correlates bounded byte, input, painted-frame, frame-age and reconnect measurements", () => {
    const telemetry = new BrowserHubTelemetry();
    telemetry.beginAttach(100);
    telemetry.addBytes(12);
    telemetry.recordInputSent("wc-input-1", 200);
    telemetry.addBytes(31);
    telemetry.recordInputApplied("wc-input-1", 230);
    telemetry.recordFramePainted(260, "2026-11-25T21:00:00.000Z", Date.parse("2026-11-25T21:00:00.075Z"));

    expect(telemetry.snapshot()).toEqual({
      frames: 1,
      bytes: 43,
      inputApplied: 1,
      attachToFirstFrameMs: 160,
      inputToPaintedFrameMs: 60,
      inputToPaintedFrameSamples: 1,
      frameAgeMs: 75,
      reconnectCount: 0,
      reconnectAttempts: 0,
      reconnectMs: undefined,
    });

    telemetry.startReconnect(500);
    telemetry.recordFramePainted(540, undefined, Date.now());
    expect(telemetry.snapshot()).toMatchObject({
      frames: 2,
      reconnectCount: 1,
      reconnectAttempts: 1,
      reconnectMs: 40,
      frameAgeMs: undefined,
    });
  });

  it("counts owner acknowledgements but measures only acknowledgements correlated to this viewer", () => {
    const telemetry = new BrowserHubTelemetry();
    telemetry.recordInputApplied("not-from-this-viewer", 50);
    telemetry.recordFramePainted(60);
    expect(telemetry.snapshot()).toMatchObject({ inputApplied: 1, inputToPaintedFrameSamples: 0 });
  });
});
