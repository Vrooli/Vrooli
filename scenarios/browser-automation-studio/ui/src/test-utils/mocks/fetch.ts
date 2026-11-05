import { vi, type Mock } from 'vitest';

export type FetchMock = Mock<typeof fetch>;

export function installFetchMock(): FetchMock {
  const fetchMock = vi.fn<typeof fetch>();
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

export function fetchJsonResponse(body: unknown, init: ResponseInit = {}): Response {
  return {
    ok: init.status === undefined || (init.status >= 200 && init.status < 300),
    status: init.status ?? 200,
    statusText: init.statusText ?? 'OK',
    json: async () => body,
    text: async () => (typeof body === 'string' ? body : JSON.stringify(body)),
  } as Response;
}

export function fetchTextResponse(body: string, init: ResponseInit = {}): Response {
  return {
    ok: init.status === undefined || (init.status >= 200 && init.status < 300),
    status: init.status ?? 200,
    statusText: init.statusText ?? 'OK',
    json: async () => JSON.parse(body),
    text: async () => body,
  } as Response;
}

export function fetchEmptyResponse(init: ResponseInit = {}): Response {
  return {
    ok: init.status === undefined || (init.status >= 200 && init.status < 300),
    status: init.status ?? 204,
    statusText: init.statusText ?? 'No Content',
    json: async () => ({}),
    text: async () => '',
  } as Response;
}
