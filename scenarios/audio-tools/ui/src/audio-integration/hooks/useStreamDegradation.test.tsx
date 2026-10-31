import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BUFFERED_MODE_NOTICE, useStreamDegradation } from "./useStreamDegradation";

describe("useStreamDegradation", () => {
  it("shows buffered mode after a degraded stream and clears after a clean completion", () => {
    const { result } = renderHook(() => useStreamDegradation());
    act(() => result.current.observeStatus("backend_degraded"));
    expect(result.current.notice).toBe(BUFFERED_MODE_NOTICE);
    act(() => result.current.observeCompletion(false));
    expect(result.current.notice).toBeNull();
  });

  it("uses buffered mode when the terminal stream envelope reports unary fallback", () => {
    const { result } = renderHook(() => useStreamDegradation());
    act(() => result.current.observeCompletion(true));
    expect(result.current.notice).toBe(BUFFERED_MODE_NOTICE);
  });
});
