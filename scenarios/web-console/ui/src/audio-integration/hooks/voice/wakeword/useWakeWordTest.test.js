import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { _resetMicOwnershipForTesting, getActiveMicLeases, installMicLifecycleCleanup } from "../micOwnership";
import { useWakeWordTest } from "./useWakeWordTest";
class FakeMediaRecorder {
    constructor(stream) {
        this.stream = stream;
        this.state = "inactive";
        this.ondataavailable = null;
        this.onstop = null;
        this.onerror = null;
    }
    start() { this.state = "recording"; }
    stop() { this.state = "inactive"; this.onstop?.(); }
    static isTypeSupported() { return true; }
}
function fakeTrack() {
    const track = {
        readyState: "live",
        muted: false,
        kind: "audio",
        stop: vi.fn(() => { track.readyState = "ended"; }),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
    };
    return track;
}
const engine = {
    extractFeatures: vi.fn(),
    compare: vi.fn(),
    compareBest: vi.fn(),
    calibrate: vi.fn(() => null),
};
const samples = [{}];
describe("useWakeWordTest lifecycle mic ownership", () => {
    let stop;
    beforeEach(() => {
        const track = fakeTrack();
        stop = track.stop;
        const stream = { getTracks: () => [track] };
        Object.defineProperty(navigator, "mediaDevices", {
            configurable: true,
            value: { getUserMedia: vi.fn().mockResolvedValue(stream) },
        });
        vi.stubGlobal("MediaRecorder", FakeMediaRecorder);
        vi.stubGlobal("requestAnimationFrame", vi.fn(() => 1));
        vi.stubGlobal("cancelAnimationFrame", vi.fn());
    });
    afterEach(() => {
        _resetMicOwnershipForTesting();
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
        Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
    });
    it("releases the mic and cancels processing when the tab is hidden mid-test", async () => {
        const uninstall = installMicLifecycleCleanup();
        const { result } = renderHook(() => useWakeWordTest({ engine, samples, threshold: 0.5, disabled: false }));
        await act(async () => { result.current.startRecording(); await Promise.resolve(); });
        expect(result.current.state.status).toBe("recording");
        expect(getActiveMicLeases().map((l) => l.owner)).toEqual(["wake-word-test"]);
        // Tab hidden → registry backstop releases the (non-active) wake-word-test lease.
        Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
        await act(async () => { document.dispatchEvent(new Event("visibilitychange")); await Promise.resolve(); });
        expect(stop).toHaveBeenCalledTimes(1);
        expect(getActiveMicLeases()).toHaveLength(0);
        expect(result.current.state.status).toBe("idle");
        // No comparison result was produced from the cancelled capture.
        expect(result.current.state.currentResult).toBeNull();
        uninstall();
    });
    it("releases the mic lease on unmount", async () => {
        const { result, unmount } = renderHook(() => useWakeWordTest({ engine, samples, threshold: 0.5, disabled: false }));
        await act(async () => { result.current.startRecording(); await Promise.resolve(); });
        expect(getActiveMicLeases()).toHaveLength(1);
        unmount();
        expect(getActiveMicLeases()).toHaveLength(0);
        expect(stop).toHaveBeenCalled();
    });
});
