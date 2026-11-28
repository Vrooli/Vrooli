import { describe, expect, it, vi } from "vitest";
import { AudioCueSession } from "./audioCueSession";

describe("AudioCueSession", () => {
  it("pairs each active recording with at most one stop cue", () => {
    const cues = { start: vi.fn(), stop: vi.fn(), fault: vi.fn() };
    const session = new AudioCueSession(cues);

    session.start();
    expect(session.stop()).toBe(true);
    expect(session.stop()).toBe(false);
    expect(cues.start).toHaveBeenCalledOnce();
    expect(cues.stop).toHaveBeenCalledOnce();
  });

  it("does not emit completion cues after teardown and signals active faults distinctly", () => {
    const cues = { start: vi.fn(), stop: vi.fn(), fault: vi.fn() };
    const session = new AudioCueSession(cues);

    session.start();
    session.clear();
    expect(session.stop()).toBe(false);
    session.start();
    expect(session.fault()).toBe(true);
    expect(session.fault()).toBe(false);
    expect(cues.stop).not.toHaveBeenCalled();
    expect(cues.fault).toHaveBeenCalledOnce();
  });
});
