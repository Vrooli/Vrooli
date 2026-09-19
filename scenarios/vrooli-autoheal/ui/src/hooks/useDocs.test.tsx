import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDocContent, useDocsManifest } from "./useDocs";

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("documentation query hooks", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("loads and validates the manifest", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ version: "1", title: "Docs", defaultDocument: "guide.md", sections: [] }),
    }));
    const { result } = renderHook(() => useDocsManifest(), { wrapper });
    await waitFor(() => expect(result.current.data?.title).toBe("Docs"));
  });

  it("rejects failed and malformed manifests", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce({ ok: false, status: 503 }).mockResolvedValueOnce({ ok: true, json: async () => ({}) }));
    const first = renderHook(() => useDocsManifest(), { wrapper });
    await waitFor(() => expect(first.result.current.error).toBeInstanceOf(Error));
    const second = renderHook(() => useDocsManifest(), { wrapper });
    await waitFor(() => expect(second.result.current.error).toBeInstanceOf(Error));
  });

  it("loads content, handles missing content, and leaves null paths disabled", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ content: "# Guide" }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({}) })
      .mockResolvedValueOnce({ ok: true, json: async () => null }));
    const loaded = renderHook(() => useDocContent("guide.md"), { wrapper });
    await waitFor(() => expect(loaded.result.current.data).toBe("# Guide"));
    const missing = renderHook(() => useDocContent("missing.md"), { wrapper });
    await waitFor(() => expect(missing.result.current.data).toBe(""));
    const invalid = renderHook(() => useDocContent("invalid.md"), { wrapper });
    await waitFor(() => expect(invalid.result.current.data).toBe(""));
    const disabled = renderHook(() => useDocContent(null), { wrapper });
    expect(disabled.result.current.fetchStatus).toBe("idle");
  });
});
