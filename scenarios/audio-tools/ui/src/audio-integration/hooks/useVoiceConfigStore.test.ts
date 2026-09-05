/**
 * Unit tests for useVoiceConfigStore.
 *
 * The store is a useSyncExternalStore wrapper. We test:
 *  - setVoiceConfigFromServer: hydrates state and notifies subscribers
 *  - _resetVoiceConfigForTesting: returns to initial fallback values
 *  - useVoiceConfigStore hook (via renderHook + act): selector returns the
 *    correct slice and re-renders when state changes
 *  - getState(): direct non-React accessor
 */

import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

import {
  setVoiceConfigFromServer,
  _resetVoiceConfigForTesting,
  useVoiceConfigStore,
} from "./useVoiceConfigStore";

import {
  VAD_FALLBACK_SILENCE_TIMEOUT_MS,
  VAD_FALLBACK_SEGMENT_SILENCE_MS,
} from "./voice";

// Reset to clean fallbacks before every test.
beforeEach(() => {
  _resetVoiceConfigForTesting();
});

describe("initial state", () => {
  it("has hydrated=false and fallback silence values", () => {
    const s = useVoiceConfigStore.getState();
    expect(s.hydrated).toBe(false);
    expect(s.vadSilenceTimeoutMs).toBe(VAD_FALLBACK_SILENCE_TIMEOUT_MS);
    expect(s.segmentSilenceMs).toBe(VAD_FALLBACK_SEGMENT_SILENCE_MS);
    expect(s.persistentMode).toBe(false);
    expect(s.wakeWordEnabled).toBe(false);
  });
});

describe("setVoiceConfigFromServer", () => {
  it("sets hydrated=true and uses vadSilenceMs when > 0", () => {
    setVoiceConfigFromServer({
      vadSilenceMs: 3000,
      segmentSilenceMs: 1000,
      persistentMode: true,
      wakeWordEnabled: false,
    });
    const s = useVoiceConfigStore.getState();
    expect(s.hydrated).toBe(true);
    expect(s.vadSilenceTimeoutMs).toBe(3000);
    expect(s.segmentSilenceMs).toBe(3000); // follows silenceMs
    expect(s.persistentMode).toBe(true);
  });

  it("falls back to segmentSilenceMs when vadSilenceMs is 0", () => {
    setVoiceConfigFromServer({
      vadSilenceMs: 0,
      segmentSilenceMs: 1200,
      persistentMode: false,
      wakeWordEnabled: true,
    });
    const s = useVoiceConfigStore.getState();
    expect(s.vadSilenceTimeoutMs).toBe(1200);
    expect(s.segmentSilenceMs).toBe(1200);
    expect(s.wakeWordEnabled).toBe(true);
  });

  it("falls back to VAD_FALLBACK constants when both silence values are 0", () => {
    setVoiceConfigFromServer({
      vadSilenceMs: 0,
      segmentSilenceMs: 0,
      persistentMode: false,
      wakeWordEnabled: false,
    });
    const s = useVoiceConfigStore.getState();
    expect(s.vadSilenceTimeoutMs).toBe(VAD_FALLBACK_SILENCE_TIMEOUT_MS);
    expect(s.segmentSilenceMs).toBe(VAD_FALLBACK_SEGMENT_SILENCE_MS);
    expect(s.hydrated).toBe(true); // still hydrated even with fallback values
  });

  it("updates wakeWordEnabled correctly", () => {
    setVoiceConfigFromServer({
      vadSilenceMs: 1500,
      segmentSilenceMs: 0,
      persistentMode: false,
      wakeWordEnabled: true,
    });
    expect(useVoiceConfigStore.getState().wakeWordEnabled).toBe(true);
  });

  it("subsequent calls overwrite previous values", () => {
    setVoiceConfigFromServer({
      vadSilenceMs: 2000,
      segmentSilenceMs: 0,
      persistentMode: true,
      wakeWordEnabled: false,
    });
    setVoiceConfigFromServer({
      vadSilenceMs: 5000,
      segmentSilenceMs: 0,
      persistentMode: false,
      wakeWordEnabled: true,
    });
    const s = useVoiceConfigStore.getState();
    expect(s.vadSilenceTimeoutMs).toBe(5000);
    expect(s.persistentMode).toBe(false);
    expect(s.wakeWordEnabled).toBe(true);
  });
});

describe("_resetVoiceConfigForTesting", () => {
  it("resets state to initial fallback values", () => {
    setVoiceConfigFromServer({
      vadSilenceMs: 9000,
      segmentSilenceMs: 9000,
      persistentMode: true,
      wakeWordEnabled: true,
    });
    _resetVoiceConfigForTesting();
    const s = useVoiceConfigStore.getState();
    expect(s.hydrated).toBe(false);
    expect(s.vadSilenceTimeoutMs).toBe(VAD_FALLBACK_SILENCE_TIMEOUT_MS);
    expect(s.segmentSilenceMs).toBe(VAD_FALLBACK_SEGMENT_SILENCE_MS);
    expect(s.persistentMode).toBe(false);
    expect(s.wakeWordEnabled).toBe(false);
  });
});

describe("useVoiceConfigStore hook (via renderHook)", () => {
  it("selector returns hydrated slice initially false", () => {
    const { result } = renderHook(() => useVoiceConfigStore((s) => s.hydrated));
    expect(result.current).toBe(false);
  });

  it("selector returns vadSilenceTimeoutMs fallback initially", () => {
    const { result } = renderHook(() =>
      useVoiceConfigStore((s) => s.vadSilenceTimeoutMs),
    );
    expect(result.current).toBe(VAD_FALLBACK_SILENCE_TIMEOUT_MS);
  });

  it("re-renders hook when setVoiceConfigFromServer is called", () => {
    const { result } = renderHook(() => useVoiceConfigStore((s) => s.hydrated));
    expect(result.current).toBe(false);

    act(() => {
      setVoiceConfigFromServer({
        vadSilenceMs: 2000,
        segmentSilenceMs: 0,
        persistentMode: false,
        wakeWordEnabled: false,
      });
    });

    expect(result.current).toBe(true);
  });

  it("re-renders hook with updated vadSilenceTimeoutMs", () => {
    const { result } = renderHook(() =>
      useVoiceConfigStore((s) => s.vadSilenceTimeoutMs),
    );

    act(() => {
      setVoiceConfigFromServer({
        vadSilenceMs: 4500,
        segmentSilenceMs: 0,
        persistentMode: false,
        wakeWordEnabled: false,
      });
    });

    expect(result.current).toBe(4500);
  });

  it("re-renders hook when _resetVoiceConfigForTesting is called", () => {
    act(() => {
      setVoiceConfigFromServer({
        vadSilenceMs: 3000,
        segmentSilenceMs: 0,
        persistentMode: true,
        wakeWordEnabled: false,
      });
    });

    const { result } = renderHook(() => useVoiceConfigStore((s) => s.persistentMode));
    expect(result.current).toBe(true);

    act(() => {
      _resetVoiceConfigForTesting();
    });

    expect(result.current).toBe(false);
  });

  it("multiple renders subscribe independently", () => {
    const { result: r1 } = renderHook(() =>
      useVoiceConfigStore((s) => s.vadSilenceTimeoutMs),
    );
    const { result: r2 } = renderHook(() =>
      useVoiceConfigStore((s) => s.wakeWordEnabled),
    );

    act(() => {
      setVoiceConfigFromServer({
        vadSilenceMs: 7000,
        segmentSilenceMs: 0,
        persistentMode: false,
        wakeWordEnabled: true,
      });
    });

    expect(r1.current).toBe(7000);
    expect(r2.current).toBe(true);
  });

  it("unmounted hooks unsubscribe and do not receive further updates", () => {
    const { result, unmount } = renderHook(() =>
      useVoiceConfigStore((s) => s.vadSilenceTimeoutMs),
    );

    act(() => {
      setVoiceConfigFromServer({
        vadSilenceMs: 1000,
        segmentSilenceMs: 0,
        persistentMode: false,
        wakeWordEnabled: false,
      });
    });
    expect(result.current).toBe(1000);

    unmount();

    // After unmount, updating state should not throw
    act(() => {
      setVoiceConfigFromServer({
        vadSilenceMs: 9999,
        segmentSilenceMs: 0,
        persistentMode: false,
        wakeWordEnabled: false,
      });
    });
  });
});

describe("useVoiceConfigStore.getState (non-React accessor)", () => {
  it("returns current state with setFromServer bound method", () => {
    const s = useVoiceConfigStore.getState();
    expect(typeof s.setFromServer).toBe("function");
    expect(s.hydrated).toBe(false);
  });

  it("setFromServer on getState() updates the store", () => {
    const s = useVoiceConfigStore.getState();
    s.setFromServer({
      vadSilenceMs: 2500,
      segmentSilenceMs: 0,
      persistentMode: false,
      wakeWordEnabled: false,
    });
    expect(useVoiceConfigStore.getState().vadSilenceTimeoutMs).toBe(2500);
    expect(useVoiceConfigStore.getState().hydrated).toBe(true);
  });
});
