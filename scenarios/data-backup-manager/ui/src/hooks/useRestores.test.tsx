import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../api/restores", () => ({
  listRestores: vi.fn(),
  getRestore: vi.fn(),
  verifyTarget: vi.fn(),
  restoreTarget: vi.fn(),
}));

import * as api from "../api/restores";
import { makeApiError } from "../api/client";
import { useRestores, useVerifyTarget } from "./useRestores";

const buildClient = () =>
  new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

const wrapper =
  (client: QueryClient) =>
  ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => vi.clearAllMocks());

describe("useRestores", () => {
  it("returns restore history on success", async () => {
    vi.mocked(api.listRestores).mockResolvedValue([{ id: "rr1", targetId: "t1" } as never]);
    const { result } = renderHook(() => useRestores("t1"), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.listRestores).toHaveBeenCalledWith("t1");
  });

  it("surfaces an ApiError", async () => {
    vi.mocked(api.listRestores).mockRejectedValue(makeApiError("internal", "boom"));
    const { result } = renderHook(() => useRestores(), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useVerifyTarget", () => {
  it("invalidates restore history and the posture rollup (clears the unverified chip)", async () => {
    vi.mocked(api.verifyTarget).mockResolvedValue({ id: "rr1", status: 4 } as never);
    const client = buildClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useVerifyTarget(), { wrapper: wrapper(client) });

    await act(async () => {
      await result.current.mutateAsync({ targetId: "t1", destinationId: "d1", snapshotId: "s1" });
    });

    const keys = invalidate.mock.calls.map((c) => (c[0] as { queryKey: unknown[] }).queryKey[0]);
    expect(keys).toContain("restores");
    expect(keys).toContain("targetStatus");
  });
});
