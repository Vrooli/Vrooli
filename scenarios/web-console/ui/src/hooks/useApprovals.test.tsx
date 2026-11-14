import { beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type PropsWithChildren } from "react";
import { createTestQueryClient } from "../test-utils";
import { useApprovalsStore } from "../stores/useApprovalsStore";
import { useApprovals, useApprovalsBadgePoll } from "./useApprovals";

const listApprovals = vi.hoisted(() => vi.fn());
vi.mock("../api/approvals", () => ({ listApprovals }));

function makeWrapper() {
  const client = createTestQueryClient();
  return function Wrapper({ children }: PropsWithChildren) {
    return createElement(QueryClientProvider, { client }, children);
  };
}

describe("approval polling", () => {
  beforeEach(() => {
    listApprovals.mockReset();
    useApprovalsStore.setState({ pendingCount: 7 });
  });

  it("counts pending decisions and ignores resolved results", async () => {
    listApprovals.mockResolvedValue([{ state: "pending" }, { state: "answered" }, { state: "pending" }]);
    renderHook(() => useApprovals(), { wrapper: makeWrapper() });
    await waitFor(() => expect(useApprovalsStore.getState().pendingCount).toBe(2));
    expect(listApprovals).toHaveBeenCalledWith(true, 100);
  });

  it("does not overwrite the pending badge from the history view", async () => {
    listApprovals.mockResolvedValue([{ state: "answered" }]);
    const { result } = renderHook(() => useApprovals({ openOnly: false }), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(useApprovalsStore.getState().pendingCount).toBe(7);
    expect(listApprovals).toHaveBeenCalledWith(false, 100);
  });

  it("retains the badge through a temporary polling failure", async () => {
    listApprovals.mockRejectedValue(new Error("offline"));
    const { result } = renderHook(() => useApprovalsBadgePoll(), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(useApprovalsStore.getState().pendingCount).toBe(7);
  });

  it("does not poll while disabled", () => {
    renderHook(() => useApprovalsBadgePoll(false), { wrapper: makeWrapper() });
    expect(listApprovals).not.toHaveBeenCalled();
    expect(useApprovalsStore.getState().pendingCount).toBe(7);
  });
});
