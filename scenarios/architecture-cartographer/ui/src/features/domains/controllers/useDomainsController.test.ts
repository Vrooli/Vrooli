import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";

vi.mock("../../../api/domains", () => ({
  domainsClient: {
    getDomainMap: vi.fn().mockResolvedValue({ domainMap: undefined }),
    extractDomains: vi.fn().mockResolvedValue({ domainMap: undefined }),
    convergenceReport: vi.fn().mockResolvedValue({ scenario: "demo", authority: 0, findings: [] }),
  },
}));

vi.mock("../../../api/signals", () => ({
  signalsClient: {
    boundaryHealth: vi.fn().mockResolvedValue({ scenario: "demo", totalDomains: 0, domains: [] }),
  },
}));

import { domainsClient } from "../../../api/domains";
import { signalsClient } from "../../../api/signals";
import {
  domainsKeys,
  useGetDomainMap,
  useExtractDomains,
  useConvergenceReport,
  useBoundaryHealth,
} from "./useDomainsController";

function wrap(client: QueryClient) {
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
}

describe("domainsKeys", () => {
  it("namespaces keys", () => {
    expect(domainsKeys.get("s")).toEqual(["domains", "get", "s"]);
    expect(domainsKeys.convergence("s")).toEqual(["domains", "convergence", "s"]);
    expect(domainsKeys.boundaries("s")).toEqual(["domains", "boundaries", "s"]);
  });
});

describe("useGetDomainMap", () => {
  it("invokes getDomainMap", async () => {
    vi.mocked(domainsClient.getDomainMap).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useGetDomainMap("demo"), { wrapper: wrap(client) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(domainsClient.getDomainMap).toHaveBeenCalledWith({ scenario: "demo" });
  });
});

describe("useExtractDomains", () => {
  it("invokes extractDomains and invalidates the get query", async () => {
    vi.mocked(domainsClient.extractDomains).mockClear();
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const { result } = renderHook(() => useExtractDomains("demo"), { wrapper: wrap(client) });
    await result.current.mutateAsync();
    expect(domainsClient.extractDomains).toHaveBeenCalledWith({ scenario: "demo" });
  });
});

describe("useConvergenceReport", () => {
  it("invokes convergenceReport", async () => {
    vi.mocked(domainsClient.convergenceReport).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useConvergenceReport("demo"), { wrapper: wrap(client) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(domainsClient.convergenceReport).toHaveBeenCalledWith({ scenario: "demo" });
  });
});

describe("useBoundaryHealth", () => {
  it("invokes boundaryHealth on the signals client", async () => {
    vi.mocked(signalsClient.boundaryHealth).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useBoundaryHealth("demo"), { wrapper: wrap(client) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(signalsClient.boundaryHealth).toHaveBeenCalledWith({ scenario: "demo" });
  });
});
