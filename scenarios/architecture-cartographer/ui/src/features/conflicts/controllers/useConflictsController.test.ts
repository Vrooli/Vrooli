import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";

vi.mock("../../../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn().mockResolvedValue({ conflicts: [], nextPageToken: "" }),
    getConflict: vi.fn().mockResolvedValue({ conflict: undefined }),
    detectConflicts: vi.fn().mockResolvedValue({ conflicts: [] }),
    validateConflicts: vi.fn().mockResolvedValue({ conflicts: [], clean: true }),
  },
}));

import { conflictsClient } from "../../../api/conflicts";
import {
  conflictsKeys,
  useDetectConflicts,
  useGetConflict,
  useListConflicts,
  useValidateConflicts,
} from "./useConflictsController";

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

describe("conflictsKeys", () => {
  it("namespaces list keys per scenario", () => {
    expect(conflictsKeys.list("foo")).toEqual(["conflicts", "list", "foo"]);
  });

  it("namespaces detail keys per id", () => {
    expect(conflictsKeys.detail("c-1")).toEqual(["conflicts", "detail", "c-1"]);
  });

  it("groups every key under the all() prefix", () => {
    expect(conflictsKeys.all()).toEqual(["conflicts"]);
  });
});

describe("useListConflicts", () => {
  it("calls listConflicts with the scenario and a default page size", async () => {
    vi.mocked(conflictsClient.listConflicts).mockClear();
    const { result } = renderHookWith(() => useListConflicts({ scenario: "demo" }));
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(conflictsClient.listConflicts).toHaveBeenCalledWith({
      scenario: "demo",
      types: [],
      pageSize: 50,
      pageToken: "",
    });
  });

  it("does not fire when the scenario is empty", () => {
    vi.mocked(conflictsClient.listConflicts).mockClear();
    renderHookWith(() => useListConflicts({ scenario: "" }));
    expect(conflictsClient.listConflicts).not.toHaveBeenCalled();
  });
});

describe("useGetConflict", () => {
  it("calls getConflict with the supplied id", async () => {
    vi.mocked(conflictsClient.getConflict).mockClear();
    const { result } = renderHookWith(() => useGetConflict({ id: "c-1" }));
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(conflictsClient.getConflict).toHaveBeenCalledWith({ id: "c-1" });
  });

  it("does not fire when id is empty", () => {
    vi.mocked(conflictsClient.getConflict).mockClear();
    renderHookWith(() => useGetConflict({ id: "" }));
    expect(conflictsClient.getConflict).not.toHaveBeenCalled();
  });
});

describe("useDetectConflicts", () => {
  it("invalidates the list cache on success", async () => {
    const client = buildClient();
    const spy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHookWith(() => useDetectConflicts("demo"), client);
    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(conflictsClient.detectConflicts).toHaveBeenCalledWith({
      scenario: "demo",
      snapshotId: "",
      idempotencyKey: "",
    });
    expect(spy).toHaveBeenCalledWith({ queryKey: conflictsKeys.list("demo") });
  });
});

describe("useValidateConflicts", () => {
  it("calls validateConflicts with the scenario and invalidates the list cache", async () => {
    const client = buildClient();
    const spy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHookWith(() => useValidateConflicts("demo"), client);
    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(conflictsClient.validateConflicts).toHaveBeenCalledWith({ scenario: "demo" });
    expect(spy).toHaveBeenCalledWith({ queryKey: conflictsKeys.list("demo") });
  });
});
