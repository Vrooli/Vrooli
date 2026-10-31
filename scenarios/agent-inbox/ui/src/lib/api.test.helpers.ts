/**
 * Shared test helpers for API tests
 */

import { vi } from "vitest";

// Mock fetch globally
export const mockFetch = vi.fn();
globalThis.fetch = mockFetch;

// Mock @vrooli/api-base
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: vi.fn(({ appendSuffix }: { appendSuffix: boolean }) =>
    appendSuffix ? "http://localhost:3000/api/v1" : "http://localhost:3000"
  ),
  buildApiUrl: vi.fn((path: string, { baseUrl }: { baseUrl: string }) =>
    `${baseUrl}${path}`
  ),
}));

// Helper to create mock Response
export function createMockResponse(
  data: unknown,
  options: { status?: number; ok?: boolean; headers?: Record<string, string> } = {}
): Response {
  const { status = 200, ok = true, headers = {} } = options;
  return {
    ok,
    status,
    headers: new Headers(headers),
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(typeof data === "string" ? data : JSON.stringify(data)),
    blob: () => Promise.resolve(new Blob([JSON.stringify(data)])),
  } as Response;
}

// Helper to create mock streaming Response
export function createStreamingResponse(events: string[], options: { status?: number; ok?: boolean } = {}): Response {
  const { status = 200, ok = true } = options;

  let readIndex = 0;
  const encoder = new TextEncoder();

  const reader: ReadableStreamDefaultReader<Uint8Array> = {
    read: () => {
      if (readIndex >= events.length) {
        return Promise.resolve({ done: true, value: undefined });
      }
      const value = encoder.encode(events[readIndex]);
      readIndex++;
      return Promise.resolve({ done: false, value });
    },
    releaseLock: () => {},
    cancel: () => Promise.resolve(),
    closed: Promise.resolve(undefined),
  };

  const body = {
    getReader: () => reader,
  } as ReadableStream<Uint8Array>;

  return {
    ok,
    status,
    body,
    headers: new Headers({ "Content-Type": "text/event-stream" }),
  } as Response;
}
