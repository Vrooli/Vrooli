import { describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { usePersistedPreference, type PreferenceStorage } from "./usePersistedPreference";

const buildStorage = (initial: Record<string, string> = {}): PreferenceStorage => {
  const store = new Map<string, string>(Object.entries(initial));
  return {
    getItem: (k) => store.get(k) ?? null,
    setItem: (k, v) => {
      store.set(k, v);
    },
  };
};

const isStringPref = (raw: unknown): string | null =>
  typeof raw === "string" ? raw : null;

describe("usePersistedPreference", () => {
  it("returns the default when no value is stored", () => {
    const { result } = renderHook(() =>
      usePersistedPreference({
        key: "k",
        defaultValue: "x",
        validate: isStringPref,
        storage: buildStorage(),
      }),
    );
    expect(result.current[0]).toBe("x");
  });

  it("hydrates from storage when present", () => {
    const { result } = renderHook(() =>
      usePersistedPreference({
        key: "k",
        defaultValue: "x",
        validate: isStringPref,
        storage: buildStorage({ k: JSON.stringify("y") }),
      }),
    );
    expect(result.current[0]).toBe("y");
  });

  it("falls back to default when stored value fails validation", () => {
    const { result } = renderHook(() =>
      usePersistedPreference({
        key: "k",
        defaultValue: "x",
        validate: isStringPref,
        storage: buildStorage({ k: JSON.stringify(42) }),
      }),
    );
    expect(result.current[0]).toBe("x");
  });

  it("falls back to default when stored value is malformed JSON", () => {
    const { result } = renderHook(() =>
      usePersistedPreference({
        key: "k",
        defaultValue: "x",
        validate: isStringPref,
        storage: buildStorage({ k: "{not json" }),
      }),
    );
    expect(result.current[0]).toBe("x");
  });

  it("writes through to storage on update", () => {
    const storage = buildStorage();
    const { result } = renderHook(() =>
      usePersistedPreference({
        key: "k",
        defaultValue: "x",
        validate: isStringPref,
        storage,
      }),
    );
    act(() => {
      result.current[1]("z");
    });
    expect(result.current[0]).toBe("z");
    expect(storage.getItem("k")).toBe(JSON.stringify("z"));
  });
});
