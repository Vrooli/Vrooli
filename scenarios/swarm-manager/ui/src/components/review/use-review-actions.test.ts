import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useReviewActions } from "./use-review-actions";

vi.mock("../../services", () => ({
  executionService: {
    triggerReview: vi.fn(),
  },
}));

// Re-import after mock is registered
const { executionService } = await import("../../services");
const mockTriggerReview = vi.mocked(executionService.triggerReview);

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useReviewActions", () => {
  it("triggerReview calls service with executionId", async () => {
    mockTriggerReview.mockResolvedValue({} as never);
    const { result } = renderHook(() => useReviewActions("exec-1"));

    await act(async () => {
      await result.current.triggerReview();
    });

    expect(mockTriggerReview).toHaveBeenCalledWith("exec-1");
    expect(result.current.isTriggering).toBe(false);
    expect(result.current.triggerError).toBeNull();
  });

  it("captures error on service failure", async () => {
    mockTriggerReview.mockRejectedValue(new Error("Network error"));
    const { result } = renderHook(() => useReviewActions("exec-1"));

    await act(async () => {
      await result.current.triggerReview();
    });

    await waitFor(() => {
      expect(result.current.triggerError).toBe("Network error");
    });
    expect(result.current.isTriggering).toBe(false);
  });

  it("is a no-op when executionId is undefined", async () => {
    const { result } = renderHook(() => useReviewActions(undefined));

    await act(async () => {
      await result.current.triggerReview();
    });

    expect(mockTriggerReview).not.toHaveBeenCalled();
  });
});
