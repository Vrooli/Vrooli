import { afterEach, describe, expect, it, vi } from "vitest";
import { _resetMicOwnershipForTesting } from "./micOwnership";
import { installVoicePageLifecycle } from "./pageLifecycle";

function setVisibility(state: DocumentVisibilityState): void {
  Object.defineProperty(document, "visibilityState", { configurable: true, value: state });
  document.dispatchEvent(new Event("visibilitychange"));
}

describe("installVoicePageLifecycle", () => {
  afterEach(() => {
    _resetMicOwnershipForTesting();
    vi.restoreAllMocks();
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
  });

  it("cancels starts, stops an active turn, and suspends audio when hidden", () => {
    const onVisibilityChange = vi.fn();
    const cancelPendingStarts = vi.fn();
    const stopActiveRecording = vi.fn();
    const suspendAudio = vi.fn();
    const uninstall = installVoicePageLifecycle({
      onVisibilityChange,
      cancelPendingStarts,
      isActiveRecording: () => true,
      stopActiveRecording,
      suspendAudio,
    });

    expect(onVisibilityChange).toHaveBeenCalledWith(false);
    setVisibility("hidden");

    expect(onVisibilityChange).toHaveBeenLastCalledWith(true);
    expect(cancelPendingStarts).toHaveBeenCalledOnce();
    expect(stopActiveRecording).toHaveBeenCalledOnce();
    expect(suspendAudio).toHaveBeenCalledOnce();
    uninstall();
  });

  it("does not stop or reacquire anything when becoming visible, and removes its listener", () => {
    const onVisibilityChange = vi.fn();
    const cancelPendingStarts = vi.fn();
    const stopActiveRecording = vi.fn();
    const suspendAudio = vi.fn();
    const uninstall = installVoicePageLifecycle({
      onVisibilityChange,
      cancelPendingStarts,
      isActiveRecording: () => false,
      stopActiveRecording,
      suspendAudio,
    });

    setVisibility("visible");
    expect(onVisibilityChange).toHaveBeenLastCalledWith(false);
    expect(cancelPendingStarts).not.toHaveBeenCalled();
    expect(stopActiveRecording).not.toHaveBeenCalled();
    expect(suspendAudio).not.toHaveBeenCalled();

    uninstall();
    setVisibility("hidden");
    expect(onVisibilityChange).toHaveBeenCalledTimes(2);
  });
});
