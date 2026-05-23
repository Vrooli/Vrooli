import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";

vi.mock("../../../api/graph", () => ({
  graphClient: {
    getGraphSnapshot: vi.fn().mockResolvedValue({ snapshot: { id: "snap-x" } }),
    listGraphSnapshots: vi.fn().mockResolvedValue({
      snapshots: [{ id: "snap-latest" }],
      nextPageToken: "",
    }),
  },
}));

vi.mock("../../../api/manifest", () => ({
  manifestClient: {
    listDomains: vi.fn().mockResolvedValue({ domains: [{ name: "graph", paths: [] }] }),
  },
}));

vi.mock("../../../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn().mockResolvedValue({ conflicts: [], nextPageToken: "" }),
  },
}));

import { graphClient } from "../../../api/graph";
import { manifestClient } from "../../../api/manifest";
import { conflictsClient } from "../../../api/conflicts";
import {
  graphKeys,
  useGetGraphSnapshot,
  useGraphWorkspace,
  useListDomains,
} from "./useGraphController";

function buildClient(): QueryClient {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

function renderHookWith<T>(hook: () => T) {
  const client = buildClient();
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
  return renderHook(hook, { wrapper });
}

describe("graphKeys", () => {
  it("namespaces snapshot keys per scenario+id", () => {
    expect(graphKeys.snapshot("demo", "snap-1")).toEqual([
      "graph",
      "snapshot",
      "demo",
      "snap-1",
    ]);
  });

  it("groups every key under all()", () => {
    expect(graphKeys.all()).toEqual(["graph"]);
  });
});

describe("useGetGraphSnapshot", () => {
  it("calls getGraphSnapshot when an id is provided", async () => {
    vi.mocked(graphClient.getGraphSnapshot).mockClear();
    const { result } = renderHookWith(() =>
      useGetGraphSnapshot({ scenario: "demo", snapshotId: "snap-1" }),
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(graphClient.getGraphSnapshot).toHaveBeenCalledWith({ id: "snap-1" });
  });

  it("falls back to listGraphSnapshots(scenario) when no id is provided", async () => {
    vi.mocked(graphClient.listGraphSnapshots).mockClear();
    const { result } = renderHookWith(() => useGetGraphSnapshot({ scenario: "demo" }));
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(graphClient.listGraphSnapshots).toHaveBeenCalledWith({
      scenario: "demo",
      pageSize: 1,
      pageToken: "",
    });
  });

  it("does not fire when scenario is empty", () => {
    vi.mocked(graphClient.listGraphSnapshots).mockClear();
    renderHookWith(() => useGetGraphSnapshot({ scenario: "" }));
    expect(graphClient.listGraphSnapshots).not.toHaveBeenCalled();
  });
});

describe("useListDomains", () => {
  it("calls listDomains with the scenario", async () => {
    vi.mocked(manifestClient.listDomains).mockClear();
    const { result } = renderHookWith(() => useListDomains({ scenario: "demo" }));
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(manifestClient.listDomains).toHaveBeenCalledWith({ scenario: "demo" });
  });
});

describe("useGraphWorkspace", () => {
  it("fans out to snapshot + domains + conflicts for the same scenario", async () => {
    vi.mocked(graphClient.listGraphSnapshots).mockClear();
    vi.mocked(manifestClient.listDomains).mockClear();
    vi.mocked(conflictsClient.listConflicts).mockClear();
    const { result } = renderHookWith(() => useGraphWorkspace("demo"));
    await waitFor(() => {
      expect(result.current.snapshot.isSuccess).toBe(true);
      expect(result.current.domains.isSuccess).toBe(true);
      expect(result.current.conflicts.isSuccess).toBe(true);
    });
    expect(graphClient.listGraphSnapshots).toHaveBeenCalled();
    expect(manifestClient.listDomains).toHaveBeenCalled();
    expect(conflictsClient.listConflicts).toHaveBeenCalledWith({
      scenario: "demo",
      statuses: [],
      types: [],
      pageSize: 200,
      pageToken: "",
    });
  });
});
