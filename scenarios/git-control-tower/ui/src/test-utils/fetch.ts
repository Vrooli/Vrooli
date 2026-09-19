import { vi } from "vitest";

export function jsonResponse(body: unknown, init: ResponseInit = {}) {
  const headers = new Headers(init.headers);
  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  return new Response(JSON.stringify(body), {
    ...init,
    status: init.status ?? 200,
    headers,
  });
}

export function textResponse(body: string, init: ResponseInit = {}) {
  return new Response(body, init);
}

export function mockFetchJson(body: unknown, init?: ResponseInit) {
  const fetchMock = vi.fn(async () => jsonResponse(body, init));
  globalThis.fetch = fetchMock as unknown as typeof fetch;
  return fetchMock;
}
