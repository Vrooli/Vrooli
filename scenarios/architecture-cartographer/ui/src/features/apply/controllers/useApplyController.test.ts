import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";

vi.mock("../../../api/apply", () => ({
  applyClient: {
    getBuildBaseline: vi.fn().mockResolvedValue({ baseline: undefined }),
    listApplyHistory: vi.fn().mockResolvedValue({ runs: [], nextPageToken: "" }),
    planApply: vi.fn().mockResolvedValue({ plan: { id: "plan-1", operations: [] }, dryRun: false }),
    runApply: vi.fn().mockResolvedValue({ run: undefined }),
  },
}));

import { applyClient } from "../../../api/apply";
import {
  applyKeys,
  useApplyHistory,
  useApplyWorkspace,
  useBuildBaseline,
  usePlanApply,
  useRunApply,
} from "./useApplyController";

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

describe("applyKeys", () => {
  it("namespaces history by (scenario, domain)", () => {
    expect(applyKeys.history("s", "d")).toEqual(["apply", "history", "s", "d"]);
  });
  it("namespaces baseline by scenario", () => {
    expect(applyKeys.baseline("s")).toEqual(["apply", "baseline", "s"]);
  });
});

describe("useBuildBaseline", () => {
  it("calls getBuildBaseline with the scenario", async () => {
    vi.mocked(applyClient.getBuildBaseline).mockClear();
    const { result } = renderHookWith(() => useBuildBaseline({ scenario: "demo" }));
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(applyClient.getBuildBaseline).toHaveBeenCalledWith({ scenario: "demo" });
  });
});

describe("useApplyHistory", () => {
  it("calls listApplyHistory with the (scenario, domain) pair", async () => {
    vi.mocked(applyClient.listApplyHistory).mockClear();
    const { result } = renderHookWith(() =>
      useApplyHistory({ scenario: "demo", domain: "foo" }),
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(applyClient.listApplyHistory).toHaveBeenCalledWith({
      scenario: "demo",
      domain: "foo",
      pageSize: 50,
      pageToken: "",
    });
  });
});

describe("usePlanApply", () => {
  it("invokes planApply and stashes the plan in the cache", async () => {
    vi.mocked(applyClient.planApply).mockClear();
    const client = buildClient();
    const { result } = renderHookWith(() => usePlanApply(), client);
    const r = await result.current.mutateAsync({ scenario: "demo", domain: "foo" });
    expect(r.plan?.id).toBe("plan-1");
    expect(client.getQueryData(applyKeys.plan("demo", "foo"))).toBeDefined();
  });
});

describe("useRunApply", () => {
  it("forwards acknowledgeV01Unimplemented=true", async () => {
    vi.mocked(applyClient.runApply).mockClear();
    const { result } = renderHookWith(() => useRunApply());
    await result.current.mutateAsync({ scenario: "s", domain: "d", planId: "plan-1" });
    expect(applyClient.runApply).toHaveBeenCalledWith({
      planId: "plan-1",
      acknowledgeV01Unimplemented: true,
    });
  });
});

describe("useApplyWorkspace", () => {
  it("returns the baseline + history queries", async () => {
    const { result } = renderHookWith(() => useApplyWorkspace("demo", "foo"));
    await waitFor(() => expect(result.current.baseline.isSuccess).toBe(true));
    expect(result.current.history).toBeDefined();
  });
});
