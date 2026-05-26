import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../api/plans", () => ({
  listPlans: vi.fn(),
  getPlan: vi.fn(),
  createPlan: vi.fn(),
  updatePlan: vi.fn(),
  deletePlan: vi.fn(),
}));

import * as api from "../api/plans";
import { makeApiError } from "../api/client";
import { useCreatePlan, usePlans } from "./usePlans";

const buildClient = () =>
  new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

const wrapper =
  (client: QueryClient) =>
  ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => vi.clearAllMocks());

describe("usePlans", () => {
  it("returns plans on success", async () => {
    vi.mocked(api.listPlans).mockResolvedValue([{ id: "p1", name: "nightly" } as never]);
    const { result } = renderHook(() => usePlans(), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
  });

  it("surfaces an ApiError", async () => {
    vi.mocked(api.listPlans).mockRejectedValue(makeApiError("internal", "boom"));
    const { result } = renderHook(() => usePlans(), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useCreatePlan", () => {
  it("invalidates the plan list", async () => {
    vi.mocked(api.createPlan).mockResolvedValue({ id: "p1" } as never);
    const client = buildClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useCreatePlan(), { wrapper: wrapper(client) });

    await act(async () => {
      await result.current.mutateAsync({
        name: "nightly",
        targetIds: ["t1"],
        destinationIds: ["d1"],
        schedule: "0 2 * * *",
        keepLatest: 7,
        enabled: true,
      });
    });

    const keys = invalidate.mock.calls.map((c) => (c[0] as { queryKey: unknown[] }).queryKey[0]);
    expect(keys).toContain("plans");
  });
});
