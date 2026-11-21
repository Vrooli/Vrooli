import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";

vi.mock("../../../api/analytics", () => ({
  analyticsClient: {
    getStats: vi.fn().mockResolvedValue({ stats: undefined }),
    listEvents: vi.fn().mockResolvedValue({ events: [], nextPageToken: "" }),
    listPlacements: vi.fn().mockResolvedValue({ placements: [], nextPageToken: "" }),
    recordOverride: vi.fn().mockResolvedValue({ override: undefined, dryRun: false }),
  },
}));

import { analyticsClient } from "../../../api/analytics";
import {
  analyticsKeys,
  useEvents,
  usePlacements,
  useRecordOverride,
  useStats,
} from "./useAnalyticsController";

function wrap(client: QueryClient) {
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
}

describe("analyticsKeys", () => {
  it("namespaces keys", () => {
    expect(analyticsKeys.stats("s")).toEqual(["analytics", "stats", "s"]);
    expect(analyticsKeys.events("s")).toEqual(["analytics", "events", "s"]);
    expect(analyticsKeys.placements("s")).toEqual(["analytics", "placements", "s"]);
  });
});

describe("useStats / useEvents / usePlacements", () => {
  it("invoke their respective client methods", async () => {
    vi.mocked(analyticsClient.getStats).mockClear();
    vi.mocked(analyticsClient.listEvents).mockClear();
    vi.mocked(analyticsClient.listPlacements).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const stats = renderHook(() => useStats("demo"), { wrapper: wrap(client) });
    const events = renderHook(() => useEvents("demo"), { wrapper: wrap(client) });
    const placements = renderHook(() => usePlacements("demo"), { wrapper: wrap(client) });
    await waitFor(() => expect(stats.result.current.isSuccess).toBe(true));
    await waitFor(() => expect(events.result.current.isSuccess).toBe(true));
    await waitFor(() => expect(placements.result.current.isSuccess).toBe(true));
    expect(analyticsClient.getStats).toHaveBeenCalledWith({ scenario: "demo" });
    expect(analyticsClient.listEvents).toHaveBeenCalled();
    expect(analyticsClient.listPlacements).toHaveBeenCalled();
  });
});

describe("useRecordOverride", () => {
  it("forwards a default idempotency key and dry_run=false", async () => {
    vi.mocked(analyticsClient.recordOverride).mockClear();
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const { result } = renderHook(() => useRecordOverride(), { wrapper: wrap(client) });
    await result.current.mutateAsync({
      scenario: "s",
      chunkId: "ck-1",
      verdictDomain: "foo",
      chosenDomain: "bar",
      note: "n",
    });
    expect(analyticsClient.recordOverride).toHaveBeenCalledWith({
      scenario: "s",
      chunkId: "ck-1",
      verdictDomain: "foo",
      chosenDomain: "bar",
      note: "n",
      verdictEventId: "",
      idempotencyKey: "",
      dryRun: false,
    });
  });
});
