import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../api/targets", () => ({
  listTargets: vi.fn(),
  getTarget: vi.fn(),
  registerTarget: vi.fn(),
  deregisterTarget: vi.fn(),
  SourceKind: { FILESYSTEM: 1 },
}));

import * as api from "../api/targets";
import { SourceKind } from "../api/targets";
import { makeApiError } from "../api/client";
import { useRegisterTarget, useTargets } from "./useTargets";

const buildClient = () =>
  new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

const wrapper =
  (client: QueryClient) =>
  ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => vi.clearAllMocks());

describe("useTargets", () => {
  it("returns the target list on success", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([
      { id: "t1", owner: "prompt-manager", name: "store" } as never,
    ]);
    const { result } = renderHook(() => useTargets("prompt-manager"), {
      wrapper: wrapper(buildClient()),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(api.listTargets).toHaveBeenCalledWith("prompt-manager");
  });

  it("surfaces an ApiError to the error state", async () => {
    vi.mocked(api.listTargets).mockRejectedValue(makeApiError("internal", "boom"));
    const { result } = renderHook(() => useTargets(), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useRegisterTarget", () => {
  it("calls the API and invalidates target + posture queries", async () => {
    vi.mocked(api.registerTarget).mockResolvedValue({ id: "t1" } as never);
    const client = buildClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useRegisterTarget(), { wrapper: wrapper(client) });

    await act(async () => {
      await result.current.mutateAsync({
        owner: "prompt-manager",
        name: "store",
        sourceKind: SourceKind.FILESYSTEM,
        locator: "store/teams",
      });
    });

    expect(api.registerTarget).toHaveBeenCalled();
    const keys = invalidate.mock.calls.map((c) => (c[0] as { queryKey: unknown[] }).queryKey[0]);
    expect(keys).toContain("targets");
    expect(keys).toContain("targetStatus");
  });
});
