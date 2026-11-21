import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../api/drills", () => ({
  listDrills: vi.fn(),
  runDrill: vi.fn(),
  DrillStatus: { UNSPECIFIED: 0, REQUESTED: 1, RUNNING: 2, VERIFIED: 3, FAILED: 4 },
}));

import * as api from "../api/drills";
import { useDrills, useRunDrill } from "./useDrills";

const wrapper = (queryClient: QueryClient) =>
  ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;

const makeClient = () => new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

beforeEach(() => vi.clearAllMocks());

describe("useDrills", () => {
  it("loads drills and keeps polling while one is active", async () => {
    vi.mocked(api.listDrills).mockResolvedValue([{ status: api.DrillStatus.RUNNING }] as never);
    const { result } = renderHook(() => useDrills("plan-1"), { wrapper: wrapper(makeClient()) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.listDrills).toHaveBeenCalledWith("plan-1");
    expect(result.current.data).toHaveLength(1);
  });

  it("loads a terminal result without active polling", async () => {
    vi.mocked(api.listDrills).mockResolvedValue([{ status: api.DrillStatus.VERIFIED }] as never);
    const { result } = renderHook(() => useDrills(), { wrapper: wrapper(makeClient()) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.listDrills).toHaveBeenCalledWith("");
    expect(result.current.data).toHaveLength(1);
  });
});

describe("useRunDrill", () => {
  it("runs a drill and invalidates drill queries", async () => {
    vi.mocked(api.runDrill).mockResolvedValue({ id: "drill-1" } as never);
    const queryClient = makeClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useRunDrill(), { wrapper: wrapper(queryClient) });

    await act(async () => {
      await result.current.mutateAsync({ planId: "plan-1", targetId: "target-1", destinationId: "dest-1" });
    });

    expect(api.runDrill).toHaveBeenCalledWith("plan-1", "target-1", "dest-1", expect.stringMatching(/^ui:plan-1:target-1:dest-1:/));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["drills"] });
  });

  it("allows a plan-wide drill without target or destination overrides", async () => {
    vi.mocked(api.runDrill).mockResolvedValue({ id: "drill-2" } as never);
    const { result } = renderHook(() => useRunDrill(), { wrapper: wrapper(makeClient()) });

    await act(async () => {
      await result.current.mutateAsync({ planId: "plan-2" });
    });

    expect(api.runDrill).toHaveBeenCalledWith("plan-2", undefined, undefined, expect.stringMatching(/^ui:plan-2:::/));
  });
});
