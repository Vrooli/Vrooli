import { describe, expect, it } from "vitest";
import { decidePersistentMode, PERSISTENT_STREAMING_UNAVAILABLE_MESSAGE } from "./persistentModeDecision";

describe("decidePersistentMode", () => {
  it("allows an explicitly requested persistent stream only for an available Whisper stream", () => {
    expect(decidePersistentMode(true, "whisper", true)).toEqual({ allowed: true });
  });

  it("refuses instead of silently downgrading when streaming is unavailable", () => {
    expect(decidePersistentMode(true, "whisper", false)).toEqual({
      allowed: false,
      reason: PERSISTENT_STREAMING_UNAVAILABLE_MESSAGE,
    });
    expect(decidePersistentMode(true, "none", false).allowed).toBe(false);
  });

  it("does not constrain an explicitly one-shot turn", () => {
    expect(decidePersistentMode(false, "whisper", false)).toEqual({ allowed: true });
  });
});
