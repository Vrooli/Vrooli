import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { armIdleSuspend, closeSharedAudioContext, getSharedAudioContext, holdSharedAudioContext, suspendSharedAudioContext } from "./sharedAudioContext";

class Context {
  state = "running";
  suspend = vi.fn(() => { this.state = "suspended"; return Promise.resolve(); });
  close = vi.fn(() => { this.state = "closed"; return Promise.resolve(); });
}
beforeEach(() => { vi.useFakeTimers(); vi.stubGlobal("AudioContext", Context); });
afterEach(() => { closeSharedAudioContext(); vi.useRealTimers(); vi.unstubAllGlobals(); });
describe("continuous audio ownership", () => {
  it("keeps duplex playback alive when another capture path goes idle", () => {
    const context = getSharedAudioContext();
    const release = holdSharedAudioContext();
    armIdleSuspend(); vi.advanceTimersByTime(2000);
    expect(context.state).toBe("running");
    release(); vi.advanceTimersByTime(2000);
    expect(context.state).toBe("suspended");
  });
  it("waits for all owners and releases each hold idempotently", () => {
    const context = getSharedAudioContext();
    const first = holdSharedAudioContext(), second = holdSharedAudioContext();
    first(); first(); armIdleSuspend(); vi.advanceTimersByTime(2000);
    expect(context.state).toBe("running");
    second(); vi.advanceTimersByTime(2000);
    expect(context.state).toBe("suspended");
  });
  it("permits immediate background suspension even while a hold exists", () => {
    const context = getSharedAudioContext(), release = holdSharedAudioContext();
    suspendSharedAudioContext(); expect(context.state).toBe("suspended"); release();
  });
});
