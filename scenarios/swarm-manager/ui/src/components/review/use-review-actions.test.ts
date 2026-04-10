import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useReviewActions } from "./use-review-actions";

vi.mock("../../services", () => ({
  executionService: {
    triggerReview: vi.fn(),
    cancel: vi.fn(),
  },
}));

vi.mock("../../services/review-service", () => ({
  reviewService: {
    triggerReviewAgent: vi.fn(),
  },
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => ({
      setQueryData: vi.fn(),
      invalidateQueries: vi.fn().mockResolvedValue(undefined),
    }),
  };
});

// Re-import after mock is registered
const { executionService } = await import("../../services");
const { reviewService } = await import("../../services/review-service");
const mockTriggerReview = vi.mocked(executionService.triggerReview);
const mockCancel = vi.mocked(executionService.cancel);
const mockTriggerReviewAgent = vi.mocked(reviewService.triggerReviewAgent);

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useReviewActions", () => {
  it("triggerReview calls service with executionId", async () => {
    mockTriggerReview.mockResolvedValue({} as never);
    const { result } = renderHook(() => useReviewActions("exec-1", "task", "item-1"));

    await act(async () => {
      await result.current.triggerReview();
    });

    expect(mockTriggerReview).toHaveBeenCalledWith("exec-1");
    expect(result.current.isTriggering).toBe(false);
    expect(result.current.triggerError).toBeNull();
  });

  it("captures error on service failure", async () => {
    mockTriggerReview.mockRejectedValue(new Error("Network error"));
    const { result } = renderHook(() => useReviewActions("exec-1", "task", "item-1"));

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

  it("triggerEvidenceOnly calls reviewService.triggerReviewAgent", async () => {
    mockTriggerReviewAgent.mockResolvedValue(undefined);
    const { result } = renderHook(() => useReviewActions("exec-1", "task", "item-1"));

    await act(async () => {
      await result.current.triggerEvidenceOnly();
    });

    expect(mockTriggerReviewAgent).toHaveBeenCalledWith("exec-1");
    expect(result.current.isTriggeringEvidence).toBe(false);
    expect(result.current.triggerError).toBeNull();
  });

  it("triggerEvidenceOnly still runs without backlog params", async () => {
    mockTriggerReviewAgent.mockResolvedValue(undefined);
    const { result } = renderHook(() => useReviewActions("exec-1"));

    await act(async () => {
      await result.current.triggerEvidenceOnly();
    });

    expect(mockTriggerReviewAgent).toHaveBeenCalledWith("exec-1");
  });

  it("cancelReview calls executionService.cancel", async () => {
    mockCancel.mockResolvedValue({} as never);
    const { result } = renderHook(() => useReviewActions("exec-1", "task", "item-1"));

    await act(async () => {
      await result.current.cancelReview();
    });

    expect(mockCancel).toHaveBeenCalledWith("exec-1");
    expect(result.current.isCancelling).toBe(false);
  });

  it("cancelReview captures error", async () => {
    mockCancel.mockRejectedValue(new Error("Cancel failed"));
    const { result } = renderHook(() => useReviewActions("exec-1", "task", "item-1"));

    await act(async () => {
      await result.current.cancelReview();
    });

    await waitFor(() => {
      expect(result.current.triggerError).toBe("Cancel failed");
    });
    expect(result.current.isCancelling).toBe(false);
  });
});
