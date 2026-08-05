import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../api/runs", () => ({
  listRuns: vi.fn(),
  getRun: vi.fn(),
  triggerRun: vi.fn(),
  browseSnapshot: vi.fn(),
  getRunStats: vi.fn(),
}));

import * as api from "../api/runs";
import { makeApiError } from "../api/client";
import { useRun, useRunStats, useRuns, useSnapshotEntries, useTriggerRun } from "./useRuns";

const buildClient = () =>
  new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

const wrapper =
  (client: QueryClient) =>
  ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => vi.clearAllMocks());

describe("useRuns", () => {
  it("returns runs on success", async () => {
    vi.mocked(api.listRuns).mockResolvedValue([{ id: "r1", planId: "p1" } as never]);
    const { result } = renderHook(() => useRuns("p1"), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.listRuns).toHaveBeenCalledWith("p1");
  });

  it("surfaces an ApiError", async () => {
    vi.mocked(api.listRuns).mockRejectedValue(makeApiError("internal", "boom"));
    const { result } = renderHook(() => useRuns(), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useTriggerRun", () => {
  it("invalidates run history and the posture rollup", async () => {
    vi.mocked(api.triggerRun).mockResolvedValue({ id: "r1" } as never);
    const client = buildClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useTriggerRun(), { wrapper: wrapper(client) });

    await act(async () => {
      await result.current.mutateAsync("p1");
    });

    const keys = invalidate.mock.calls.map((c) => (c[0] as { queryKey: unknown[] }).queryKey[0]);
    expect(keys).toContain("runs");
    expect(keys).toContain("targetStatus");
  });
});

describe("run detail queries", () => {
  it("loads a run, stats, and snapshot entries when identifiers are present", async () => {
    vi.mocked(api.getRun).mockResolvedValue({ id: "r1", status: 4 } as never);
    vi.mocked(api.getRunStats).mockResolvedValue({ totalRuns: 1 } as never);
    vi.mocked(api.browseSnapshot).mockResolvedValue([]);
    const client = buildClient();
    const run = renderHook(() => useRun("r1"), { wrapper: wrapper(client) });
    const stats = renderHook(() => useRunStats("p1"), { wrapper: wrapper(client) });
    const snapshot = renderHook(() => useSnapshotEntries("d1", "s1"), { wrapper: wrapper(client) });
    await waitFor(() => expect(run.result.current.isSuccess).toBe(true));
    await waitFor(() => expect(stats.result.current.isSuccess).toBe(true));
    await waitFor(() => expect(snapshot.result.current.isSuccess).toBe(true));
    expect(api.getRun).toHaveBeenCalledWith("r1");
    expect(api.getRunStats).toHaveBeenCalledWith("p1");
    expect(api.browseSnapshot).toHaveBeenCalledWith("d1", "s1", "");
  });
});
