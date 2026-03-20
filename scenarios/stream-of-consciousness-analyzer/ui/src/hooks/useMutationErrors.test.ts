import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { useMutationErrors } from "./useMutationErrors";
import type { UseMutationResult } from "@tanstack/react-query";

// [REQ:P0-001] Consolidated mutation error handling

function mockMutation(error: Error | null = null): UseMutationResult<unknown, Error, unknown, unknown> {
  return {
    error,
    reset: vi.fn(),
    // Minimal stubs for the rest of the interface
    data: undefined,
    isError: !!error,
    isIdle: !error,
    isPending: false,
    isSuccess: false,
    status: error ? "error" : "idle",
    variables: undefined,
    failureCount: 0,
    failureReason: null,
    isPaused: false,
    submittedAt: 0,
    context: undefined,
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
  } as unknown as UseMutationResult<unknown, Error, unknown, unknown>;
}

describe("useMutationErrors", () => {
  it("returns null when no mutations have errors", () => {
    const m1 = mockMutation();
    const m2 = mockMutation();
    const { result } = renderHook(() => useMutationErrors([m1, m2]));
    expect(result.current.activeError).toBeNull();
  });

  it("returns the first error found", () => {
    const err = new Error("conflict");
    const m1 = mockMutation();
    const m2 = mockMutation(err);
    const m3 = mockMutation(new Error("other"));
    const { result } = renderHook(() => useMutationErrors([m1, m2, m3]));
    expect(result.current.activeError).toBe(err);
  });

  it("resetAll calls reset on every mutation", () => {
    const m1 = mockMutation(new Error("a"));
    const m2 = mockMutation();
    const m3 = mockMutation(new Error("b"));
    const { result } = renderHook(() => useMutationErrors([m1, m2, m3]));

    result.current.resetAll();

    expect(m1.reset).toHaveBeenCalledOnce();
    expect(m2.reset).toHaveBeenCalledOnce();
    expect(m3.reset).toHaveBeenCalledOnce();
  });

  it("works with a single mutation", () => {
    const err = new Error("solo");
    const m = mockMutation(err);
    const { result } = renderHook(() => useMutationErrors([m]));
    expect(result.current.activeError).toBe(err);
  });

  it("works with an empty array", () => {
    const { result } = renderHook(() => useMutationErrors([]));
    expect(result.current.activeError).toBeNull();
  });
});
