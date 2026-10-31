import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { API_TIMEOUT_MS, ApiError, boundedFetch, decodeApiError, uploadFile } from "./client";

describe("api/client REST helpers", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("throws ApiError with the typed envelope on non-2xx responses", async () => {
    const err = await decodeApiError(
      new Response(JSON.stringify({ code: "internal", message: "store down" }), {
        status: 500,
      }),
    );

    expect(err).toBeInstanceOf(ApiError);
    expect(err.code).toBe("internal");
    expect(err.status).toBe(500);
    expect(err.message).toContain("store down");
  });

  it("falls back to an internal envelope when the error body is malformed", async () => {
    const err = await decodeApiError(new Response("not json", { status: 502 }));

    expect(err.code).toBe("internal");
    expect(err.status).toBe(502);
    expect(err.message).toContain("unexpected 502 response");
  });

  it("posts multipart form data through the REST helper", async () => {
    const formData = new FormData();
    formData.set("file", new File(["hello"], "hello.txt", { type: "text/plain" }));
    fetchSpy.mockResolvedValueOnce(new Response("{}", { status: 200 }));

    await uploadFile("/things/thing-1/attachments", formData);

    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/api\/v1\/things\/thing-1\/attachments$/);
    expect(init).toMatchObject({ method: "POST", body: formData, cache: "no-store" });
    expect(init.headers).toBeUndefined();
  });

  it("turns an unresponsive request into a typed timeout", async () => {
    vi.useFakeTimers();
    fetchSpy.mockImplementation((_input: RequestInfo | URL, init?: RequestInit) => {
      return new Promise((_resolve, reject) => {
        init?.signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")));
      });
    });

    const request = boundedFetch("/api/v1/health");
    const timedOut = expect(request).rejects.toMatchObject({ code: "timeout", status: 408 });
    await vi.advanceTimersByTimeAsync(API_TIMEOUT_MS);
    await timedOut;
    vi.useRealTimers();
  });
});
