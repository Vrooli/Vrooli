import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";

vi.mock("../../../api/manifest", () => ({
  manifestClient: {
    getManifest: vi.fn().mockResolvedValue({ manifest: undefined }),
    listDomains: vi.fn().mockResolvedValue({ domains: [] }),
    validateManifest: vi.fn().mockResolvedValue({ manifest: undefined, diagnostics: [], valid: true }),
  },
}));

import { manifestClient } from "../../../api/manifest";
import {
  manifestKeys,
  useGetManifest,
  useListDomains,
  useValidateManifest,
} from "./useManifestController";

function wrap(client: QueryClient) {
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
}

describe("manifestKeys", () => {
  it("namespaces keys", () => {
    expect(manifestKeys.get("s")).toEqual(["manifest", "get", "s"]);
    expect(manifestKeys.domains("s")).toEqual(["manifest", "domains", "s"]);
  });
});

describe("useGetManifest", () => {
  it("invokes getManifest", async () => {
    vi.mocked(manifestClient.getManifest).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useGetManifest("demo"), { wrapper: wrap(client) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(manifestClient.getManifest).toHaveBeenCalledWith({ scenario: "demo" });
  });
});

describe("useListDomains", () => {
  it("invokes listDomains", async () => {
    vi.mocked(manifestClient.listDomains).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useListDomains("demo"), { wrapper: wrap(client) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(manifestClient.listDomains).toHaveBeenCalledWith({ scenario: "demo" });
  });
});

describe("useValidateManifest", () => {
  it("posts an empty source by default", async () => {
    vi.mocked(manifestClient.validateManifest).mockClear();
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const { result } = renderHook(() => useValidateManifest("demo"), { wrapper: wrap(client) });
    await result.current.mutateAsync();
    expect(manifestClient.validateManifest).toHaveBeenCalledWith({
      scenario: "demo",
      source: expect.any(Uint8Array),
      contentType: "",
    });
  });
});
