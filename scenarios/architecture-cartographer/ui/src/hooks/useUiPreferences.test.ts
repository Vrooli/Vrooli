import { describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useUiPreferences, UI_PREFERENCE_DEFAULTS } from "./useUiPreferences";
import type { PreferenceStorage } from "./usePersistedPreference";

function memoryStorage(): PreferenceStorage {
  const store = new Map<string, string>();
  return {
    getItem: (k) => store.get(k) ?? null,
    setItem: (k, v) => {
      store.set(k, v);
    },
  };
}

describe("useUiPreferences", () => {
  it("returns defaults when no value has been persisted", () => {
    const { result } = renderHook(() => useUiPreferences({ storage: memoryStorage() }));
    expect(result.current.preferences).toEqual(UI_PREFERENCE_DEFAULTS);
  });

  it("updates a single preference via updatePreference", () => {
    const { result } = renderHook(() => useUiPreferences({ storage: memoryStorage() }));
    act(() => result.current.updatePreference("density", "dense"));
    expect(result.current.preferences.density).toBe("dense");
  });

  it("rejects invalid stored values and falls back to defaults", () => {
    const storage = memoryStorage();
    storage.setItem("cartographer.uiPreferences", JSON.stringify({ density: "xxx" }));
    const { result } = renderHook(() => useUiPreferences({ storage }));
    expect(result.current.preferences.density).toBe(UI_PREFERENCE_DEFAULTS.density);
  });
});
