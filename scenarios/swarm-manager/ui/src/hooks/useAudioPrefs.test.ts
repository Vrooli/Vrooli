import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useAudioPrefs, setAudioPrefs } from "./useAudioPrefs";

const STORAGE_KEY = "swarm-manager:audio-prefs:v1";

describe("useAudioPrefs", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to autoSpeak=false", () => {
    const { result } = renderHook(() => useAudioPrefs());
    expect(result.current[0].autoSpeak).toBe(false);
  });

  it("persists and reads back via localStorage", () => {
    const { result, rerender } = renderHook(() => useAudioPrefs());
    act(() => result.current[1]({ autoSpeak: true }));
    expect(result.current[0].autoSpeak).toBe(true);
    expect(JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "{}")).toMatchObject({ autoSpeak: true });

    rerender();
    expect(result.current[0].autoSpeak).toBe(true);
  });

  it("setAudioPrefs (non-hook) notifies subscribers", () => {
    const { result } = renderHook(() => useAudioPrefs());
    act(() => setAudioPrefs({ autoSpeak: true }));
    expect(result.current[0].autoSpeak).toBe(true);
  });
});
