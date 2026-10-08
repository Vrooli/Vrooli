import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";

vi.mock("../../../api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn().mockResolvedValue({ snapshots: [], nextPageToken: "" }),
    extractGraph: vi.fn().mockResolvedValue({ snapshot: { id: "snap:x" }, fromCache: false }),
  },
}));

import { graphClient } from "../../../api/graph";
import { targetsKeys, useExtractGraph, useListSnapshots } from "./useTargetsController";

function buildClient(): QueryClient {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

function renderHookWith<T>(hook: () => T, client: QueryClient = buildClient()) {
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
  const { result } = renderHook(hook, { wrapper });
  return { result, client };
}

describe("targetsKeys", () => {
  it("namespaces snapshot keys per scenario", () => {
    expect(targetsKeys.snapshots()).toEqual(["targets", "snapshots", "_all"]);
    expect(targetsKeys.snapshots("foo")).toEqual(["targets", "snapshots", "foo"]);
  });

  it("groups every key under the all() prefix", () => {
    expect(targetsKeys.all()).toEqual(["targets"]);
  });
});

describe("useListSnapshots", () => {
  it("calls listGraphSnapshots with the supplied scenario filter", async () => {
    vi.mocked(graphClient.listGraphSnapshots).mockClear();
    const { result } = renderHookWith(() =>
      useListSnapshots({ scenario: "architecture-cartographer", pageSize: 5 }),
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(graphClient.listGraphSnapshots).toHaveBeenCalledWith({
      scenario: "architecture-cartographer",
      pageSize: 5,
      pageToken: "",
    });
  });

  it("defaults scenario to empty string when not supplied", async () => {
    vi.mocked(graphClient.listGraphSnapshots).mockClear();
    const { result } = renderHookWith(() => useListSnapshots());
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(graphClient.listGraphSnapshots).toHaveBeenCalledWith(
      expect.objectContaining({ scenario: "" }),
    );
  });

  it("does not fire when enabled=false", () => {
    vi.mocked(graphClient.listGraphSnapshots).mockClear();
    renderHookWith(() => useListSnapshots({ enabled: false }));
    expect(graphClient.listGraphSnapshots).not.toHaveBeenCalled();
  });
});

describe("useExtractGraph", () => {
  it("calls extractGraph with the supplied scenario and invalidates target caches", async () => {
    const client = buildClient();
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    vi.mocked(graphClient.extractGraph).mockClear();
    const { result } = renderHookWith(() => useExtractGraph(), client);

    result.current.mutate({ scenario: "demo" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(graphClient.extractGraph).toHaveBeenCalledWith({
      scenario: "demo",
      languages: [],
      idempotencyKey: "",
    });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: targetsKeys.all() });
  });
});
