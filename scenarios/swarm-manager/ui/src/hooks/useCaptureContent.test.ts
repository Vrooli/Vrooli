import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { createElement } from "react";

// Mock buildApiUrl to return a predictable URL.
vi.mock("@vrooli/api-base", () => ({
  buildApiUrl: (path: string) => `http://test${path}`,
}));

import { useCaptureContent } from "./useCaptureContent";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("useCaptureContent", () => {
  it("returns loading state initially", () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(
      () => new Promise(() => {}), // never resolves
    );

    const { result } = renderHook(
      () => useCaptureContent("fix", "my-item", "output.txt"),
      { wrapper: createWrapper() },
    );

    expect(result.current.isLoading).toBe(true);
    expect(result.current.content).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it("fetches and returns text content", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("hello world", {
        status: 200,
        headers: { "Content-Type": "text/plain" },
      }),
    );

    const { result } = renderHook(
      () => useCaptureContent("fix", "my-item", "output.txt"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.content).toBe("hello world");
    expect(result.current.error).toBeNull();
    expect(result.current.isTruncated).toBe(false);
  });

  it("returns error on HTTP failure", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(null, { status: 404, statusText: "Not Found" }),
    );

    const { result } = renderHook(
      () => useCaptureContent("fix", "my-item", "missing.txt"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.error).toBeTruthy());
    expect(result.current.content).toBeNull();
    expect(result.current.error).toContain("404");
  });

  it("does not fetch when capturePath is empty", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");

    renderHook(
      () => useCaptureContent("fix", "my-item", ""),
      { wrapper: createWrapper() },
    );

    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("shows error for binary content", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(new Uint8Array([0x89, 0x50]), {
        status: 200,
        headers: { "Content-Type": "application/octet-stream" },
      }),
    );

    const { result } = renderHook(
      () => useCaptureContent("fix", "my-item", "binary.bin"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.error).toBeTruthy());
    expect(result.current.error).toContain("binary");
  });

  it("truncates large content and sets isTruncated", async () => {
    const largeContent = "x".repeat(60_000);
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(largeContent, {
        status: 200,
        headers: { "Content-Type": "text/plain" },
      }),
    );

    const { result } = renderHook(
      () => useCaptureContent("fix", "my-item", "big.txt"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isTruncated).toBe(true);
    expect(result.current.content!.length).toBe(50_000);
  });
});
