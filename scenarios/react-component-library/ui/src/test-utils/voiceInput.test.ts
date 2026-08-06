import { describe, expect, it, vi } from "vitest";
import { VoiceInputController } from "../hooks/useVoiceInput/versions/1.0.0/useVoiceInput";
import { FakeVoiceAdapter, FakeVoiceClock, FakeVoiceCues, FakeVoiceMedia } from "./voiceInput";

const arrange = (mode: "always-on" | "timeout" = "always-on") => {
  const adapter = new FakeVoiceAdapter();
  const media = new FakeVoiceMedia();
  const clock = new FakeVoiceClock();
  const cues = new FakeVoiceCues();
  return {
    adapter,
    media,
    clock,
    cues,
    controller: new VoiceInputController({ adapter, media, clock, cues, mode, timeoutMs: 10 }),
  };
};

describe("VoiceInputController", () => {
  it("keeps always-on capture alive across settled silence segments and preserves their order", async () => {
    const { adapter, clock, controller } = arrange();
    await controller.start();
    adapter.emit({ type: "segment", segment: { id: "one", text: "first", final: true } });
    adapter.emit({ type: "segment", segment: { id: "one", text: "first", final: true } });
    adapter.emit({ type: "segment", segment: { id: "two", text: "second", final: true } });
    expect(controller.snapshot.state).toBe("recording");
    expect(controller.snapshot.settledSegments.map(({ text }) => text)).toEqual([
      "first",
      "second",
    ]);
    expect(clock.pending).toBe(0);
  });

  it("replays a new capture with fresh start and stop cues", async () => {
    const { adapter, cues, controller } = arrange();
    await controller.start();
    await controller.stop();
    await controller.start();
    await controller.stop();
    expect(adapter.stopReasons).toEqual(["explicit-stop", "explicit-stop"]);
    expect(cues.played).toEqual(["start", "stop", "start", "stop"]);
  });

  it("stops timeout capture once and releases every owned resource", async () => {
    const { adapter, media, clock, cues, controller } = arrange("timeout");
    await controller.start();
    expect(clock.pending).toBe(1);
    clock.fireAll();
    await vi.waitFor(() => expect(controller.snapshot.terminalReason).toBe("timeout"));
    expect(controller.snapshot.terminalReason).toBe("timeout");
    expect(adapter.stopReasons).toEqual(["timeout"]);
    expect(media.capture.stopped).toBe(1);
    expect(cues.played).toEqual(["start", "stop"]);
    await controller.stop();
    expect(adapter.stopReasons).toEqual(["timeout"]);
  });

  it("handles denied permission without starting transport or a stop cue", async () => {
    const { adapter, media, cues, controller } = arrange();
    media.error = new Error("denied");
    await controller.start();
    expect(controller.snapshot.state).toBe("unavailable");
    expect(adapter.connectCalls).toBe(0);
    expect(cues.played).toEqual([]);
  });

  it("recovers, exposes rejection, and terminates a device-ended capture exactly once", async () => {
    const { adapter, media, controller } = arrange();
    await controller.start();
    adapter.emit({ type: "recovering" });
    expect(controller.snapshot.state).toBe("recovering");
    adapter.emit({ type: "reconnected" });
    adapter.emit({ type: "rejected", reason: "not recognized" });
    media.capture.end();
    await vi.waitFor(() => expect(controller.snapshot.terminalReason).toBe("device-ended"));
    media.capture.end();
    expect(controller.snapshot.terminalReason).toBe("device-ended");
    expect(controller.snapshot.rejectionReason).toBe("not recognized");
    expect(media.capture.stopped).toBe(1);
  });
});
