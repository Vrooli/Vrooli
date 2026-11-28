// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// DOC: docs/internal/SEAMS.md — globalThis.fetch is the api.ts behavior seam
import { vi, type Mock } from "vitest";

/**
 * Specifies the response for a single matched fetch call.
 *
 * - `status` defaults to 200
 * - `body` is JSON-serialized when present and the response will resolve `.json()`
 * - `text` overrides body and is returned by `.text()`
 * - `headers` are passed through to the synthetic Response
 */
export interface MockFetchResponse {
    status?: number;
    body?: unknown;
    text?: string;
    headers?: Record<string, string>;
}

/**
 * A predicate matching a fetch call by URL and (optionally) method.
 * If `urlPattern` is a string, the call URL must include it; if a RegExp, the URL is tested against it.
 */
export interface FetchMatcher {
    urlPattern: string | RegExp;
    method?: string;
}

/**
 * One programmed entry in the fetch mock — a matcher plus the response (or error) to return.
 */
export interface FetchProgram {
    match: FetchMatcher;
    response?: MockFetchResponse;
    error?: Error;
}

/** Records a single observed fetch call against the mock. */
export interface FetchCall {
    url: string;
    method: string;
    body?: string;
    headers?: Record<string, string>;
}

/**
 * Controllable handle returned by `mockFetch`. Tests use this to:
 *   - program responses (`.respondTo(...)`)
 *   - inspect what was called (`.calls`)
 *   - restore the global fetch (`.restore()`)
 *
 * The seam is `globalThis.fetch`. We intentionally swap the global rather than
 * forcing a fetch parameter through every api.ts function — keeping callers
 * simple and using vitest's spyOn pattern as documented in SEAMS.md.
 */
export interface MockFetchHandle {
    /** All fetch invocations seen since this handle was created, in order. */
    readonly calls: FetchCall[];
    /** Mock function powering the swap (use for `.toHaveBeenCalledWith` assertions). */
    readonly fn: Mock;
    /** Program a response for any call whose URL matches `pattern` (and optional method). */
    respondTo(match: FetchMatcher, response: MockFetchResponse): MockFetchHandle;
    /** Program a thrown network error for any call whose URL matches `pattern`. */
    rejectWith(match: FetchMatcher, error: Error): MockFetchHandle;
    /** Restore the original `globalThis.fetch`. Tests should call this in teardown if they don't rely on global cleanup. */
    restore(): void;
}

const DEFAULT_FALLBACK: MockFetchResponse = {
    status: 500,
    text: "mockFetch: no response programmed for this call",
};

/**
 * Replace `globalThis.fetch` with a programmable mock. Returns a handle that
 * tests use to set up responses and inspect calls.
 *
 * Behavior:
 *   - Programs are matched in insertion order; the first matching program wins.
 *   - Unmatched calls return a 500 response with a diagnostic body so tests
 *     fail loudly instead of silently producing real HTTP traffic.
 *   - Calls are recorded with method/url/body/headers for assertions.
 *
 * This helper exists so api.ts behavior tests can swap the network seam without
 * any production-code changes — the global fetch IS the seam.
 *
 * Example:
 * ```ts
 * const fetchMock = mockFetch();
 * fetchMock.respondTo({ urlPattern: "/health" }, { body: createMockHealthResponse() });
 * await fetchHealth();
 * expect(fetchMock.calls[0].url).toContain("/health");
 * fetchMock.restore();
 * ```
 */
export function mockFetch(): MockFetchHandle {
    const programs: FetchProgram[] = [];
    const calls: FetchCall[] = [];
    const original = globalThis.fetch;

    const fn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
        const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
        const method = (init?.method ?? "GET").toUpperCase();
        const body = typeof init?.body === "string" ? init.body : undefined;
        const headers = init?.headers ? Object.fromEntries(new Headers(init.headers).entries()) : undefined;

        calls.push({ url, method, body, headers });

        const matched = programs.find((p) => matches(p.match, url, method));
        if (matched?.error) throw matched.error;

        const resp = matched?.response ?? DEFAULT_FALLBACK;
        return buildResponse(resp);
    });

    Object.defineProperty(globalThis, "fetch", { value: fn, writable: true, configurable: true });

    const handle: MockFetchHandle = {
        calls,
        fn,
        respondTo(match, response) {
            programs.push({ match, response });
            return handle;
        },
        rejectWith(match, error) {
            programs.push({ match, error });
            return handle;
        },
        restore() {
            Object.defineProperty(globalThis, "fetch", { value: original, writable: true, configurable: true });
        },
    };

    return handle;
}

function matches(matcher: FetchMatcher, url: string, method: string): boolean {
    if (matcher.method && matcher.method.toUpperCase() !== method) return false;
    if (typeof matcher.urlPattern === "string") return url.includes(matcher.urlPattern);
    return matcher.urlPattern.test(url);
}

function buildResponse(spec: MockFetchResponse): Response {
    const status = spec.status ?? 200;
    const headers = new Headers(spec.headers ?? {});
    let body: BodyInit | null = null;

    if (spec.text !== undefined) {
        body = spec.text;
    } else if (spec.body !== undefined) {
        body = JSON.stringify(spec.body);
        if (!headers.has("Content-Type")) headers.set("Content-Type", "application/json");
    }

    return new Response(body, { status, headers });
}
