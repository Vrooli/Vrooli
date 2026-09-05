import { describe, expect, it } from "vitest";
import { StreamDiagnosticRecorder } from "@vrooli/audio-capture-browser";

describe("StreamDiagnosticRecorder", () => {
  it("[REQ:ATD-P1-002] exports bounded metadata without transcript or audio", () => {
    const diagnostic = new StreamDiagnosticRecorder("session-opaque", 0, "persistent");
    diagnostic.state("recording");
    diagnostic.captured(4n);
    diagnostic.sent(4n);
    diagnostic.processed(3n);
    diagnostic.status("queued");
    diagnostic.error("backend_unavailable");
    diagnostic.terminal("failed", "recovery_failed");

    const exported = diagnostic.exportJSON();
    expect(JSON.parse(exported)).toMatchObject({
      sessionId: "session-opaque", capturedSequence: 4, sentSequence: 4,
      processedSequence: 3, terminalReason: "recovery_failed",
    });
    expect(exported).not.toContain("transcript");
    expect(exported).not.toContain("audio");
  });

  it("bounds event history and codes", () => {
    const diagnostic = new StreamDiagnosticRecorder("session", 0, "reduced");
    for (let index = 0; index < 40; index++) diagnostic.status(`status_${index}`);
    const snapshot = diagnostic.read();
    expect(snapshot.events).toHaveLength(32);
    expect(snapshot.statusCodes).toHaveLength(12);
    expect(snapshot.statusCodes[0]).toBe("status_28");
  });
});
