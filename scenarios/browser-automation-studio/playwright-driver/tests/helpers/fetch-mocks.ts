export type FetchMock = jest.Mock<Promise<Response>, [RequestInfo | URL, RequestInit?]>;

export function installFetchMock(fetchMock: FetchMock = jest.fn<Promise<Response>, [RequestInfo | URL, RequestInit?]>()): FetchMock {
  globalThis.fetch = fetchMock as unknown as typeof fetch;
  return fetchMock;
}

export function fetchJsonResponse(value: unknown, init?: ResponseInit): Response {
  return new Response(JSON.stringify(value), {
    status: init?.status ?? 200,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });
}

export function fetchTextResponse(value: string, status: number): Response {
  return new Response(value, { status });
}

export function getFetchRequestOptions(fetchMock: FetchMock, callIndex = 0): RequestInit {
  const call = fetchMock.mock.calls[callIndex];
  if (!call) {
    throw new Error('Expected fetch to have been called');
  }
  const options = call[1];
  if (!options) {
    throw new Error('Expected fetch to have been called with options');
  }
  return options;
}

export function getFetchRequestBodyJson(fetchMock: FetchMock, callIndex = 0): Record<string, unknown> {
  const options = getFetchRequestOptions(fetchMock, callIndex);
  const body = options.body;
  if (typeof body !== 'string') {
    throw new Error('Expected request body to be a JSON string');
  }
  const parsed: unknown = JSON.parse(body);
  if (!isRecord(parsed)) {
    throw new Error('Expected JSON body to be an object');
  }
  return parsed;
}

export function getFetchHeaders(headers: HeadersInit | undefined): Record<string, string> {
  if (!headers) {
    return {};
  }
  if (headers instanceof Headers) {
    const record: Record<string, string> = {};
    headers.forEach((value, key) => {
      record[key] = value;
    });
    return record;
  }
  if (Array.isArray(headers)) {
    const record: Record<string, string> = {};
    for (const [key, value] of headers) {
      record[key] = value;
    }
    return record;
  }
  return headers;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}
