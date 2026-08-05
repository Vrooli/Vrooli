import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../api/destinations", () => ({
  listDestinations: vi.fn(),
  getDestination: vi.fn(),
  getDestinationUsage: vi.fn(),
  analyzeDestination: vi.fn(),
  createDestination: vi.fn(),
  updateDestination: vi.fn(),
  deleteDestination: vi.fn(),
  BackendKind: { FILESYSTEM: 1 },
  CapPolicy: { ALERT_BLOCK: 1 },
}));

import * as api from "../api/destinations";
import { BackendKind, CapPolicy } from "../api/destinations";
import { makeApiError } from "../api/client";
import {
  useAnalyzeDestination,
  useCreateDestination,
  useDestination,
  useDestinationUsage,
  useDestinations,
} from "./useDestinations";

const buildClient = () =>
  new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

const wrapper =
  (client: QueryClient) =>
  ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => vi.clearAllMocks());

describe("useDestinations", () => {
  it("returns destinations on success", async () => {
    vi.mocked(api.listDestinations).mockResolvedValue([{ id: "d1", name: "local" } as never]);
    const { result } = renderHook(() => useDestinations(), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
  });

  it("surfaces an ApiError", async () => {
    vi.mocked(api.listDestinations).mockRejectedValue(makeApiError("internal", "boom"));
    const { result } = renderHook(() => useDestinations(), { wrapper: wrapper(buildClient()) });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useCreateDestination", () => {
  it("invalidates destination + usage queries", async () => {
    vi.mocked(api.createDestination).mockResolvedValue({ id: "d1" } as never);
    const client = buildClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useCreateDestination(), { wrapper: wrapper(client) });

    await act(async () => {
      await result.current.mutateAsync({
        name: "local",
        backendKind: BackendKind.FILESYSTEM,
        location: "/var/backups",
        capBytes: 0n,
        capPolicy: CapPolicy.ALERT_BLOCK,
      });
    });

    const keys = invalidate.mock.calls.map((c) => (c[0] as { queryKey: unknown[] }).queryKey[0]);
    expect(keys).toContain("destinations");
    expect(keys).toContain("destinationUsage");
  });
});

describe("destination detail hooks", () => {
  it("loads a destination and its usage when an id is present", async () => {
    vi.mocked(api.getDestination).mockResolvedValue({ id: "d1" } as never);
    vi.mocked(api.getDestinationUsage).mockResolvedValue({ physicalBytes: 1n } as never);
    const client = buildClient();
    const destination = renderHook(() => useDestination("d1"), { wrapper: wrapper(client) });
    const usage = renderHook(() => useDestinationUsage("d1"), { wrapper: wrapper(client) });
    await waitFor(() => expect(destination.result.current.isSuccess).toBe(true));
    await waitFor(() => expect(usage.result.current.isSuccess).toBe(true));
    expect(api.getDestination).toHaveBeenCalledWith("d1");
    expect(api.getDestinationUsage).toHaveBeenCalledWith("d1");
  });

  it("runs readiness analysis", async () => {
    vi.mocked(api.analyzeDestination).mockResolvedValue({} as never);
    const { result } = renderHook(() => useAnalyzeDestination(), { wrapper: wrapper(buildClient()) });
    await act(async () => {
      await result.current.mutateAsync({ location: "/backups" });
    });
    expect(api.analyzeDestination).toHaveBeenCalledWith({ location: "/backups" });
  });
});
