import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, artifactDownloadUrl, authedFetch, decodeApiError, uploadFile } from "./client";
import { clearSession, saveSession, SESSION_EXPIRED_EVENT } from "../features/session/store";

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

  it("builds an artifact download URL with the ref percent-encoded", () => {
    const url = artifactDownloadUrl("dsh://bundle/abc def");
    expect(url).toMatch(/\/api\/v1\/artifacts\/dsh%3A%2F%2Fbundle%2Fabc%20def\/download$/);
  });
});

describe("authedFetch owner-token attachment", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }));
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    clearSession();
  });

  const sentHeaders = (): Headers => {
    const [, init] = fetchSpy.mock.calls[0] as [RequestInfo, RequestInit];
    return new Headers(init.headers);
  };

  it("attaches the owner token as an Authorization bearer header when present", async () => {
    saveSession({ ownerToken: "jwt-xyz", ownerEmail: null });
    await authedFetch("http://api.test/rpc");
    expect(sentHeaders().get("Authorization")).toBe("Bearer jwt-xyz");
  });

  it("omits Authorization when no owner token is present", async () => {
    await authedFetch("http://api.test/rpc");
    expect(sentHeaders().get("Authorization")).toBeNull();
  });

  it("does not overwrite an Authorization header the caller already set", async () => {
    saveSession({ ownerToken: "jwt-xyz", ownerEmail: null });
    await authedFetch("http://api.test/rpc", { headers: { Authorization: "Bearer explicit" } });
    expect(sentHeaders().get("Authorization")).toBe("Bearer explicit");
  });

  it("announces session expiry on a 401 for a token-bearing request", async () => {
    saveSession({ ownerToken: "jwt-stale", ownerEmail: null });
    fetchSpy.mockResolvedValue(new Response("{}", { status: 401 }));
    const expired = vi.fn();
    window.addEventListener(SESSION_EXPIRED_EVENT, expired);

    await authedFetch("http://api.test/vrooli.vrooli_bridge.v1.registry.RegistryService/ListNodes");

    window.removeEventListener(SESSION_EXPIRED_EVENT, expired);
    expect(expired).toHaveBeenCalledTimes(1);
  });

  it("does not announce expiry on a 401 without a token or from IdentityService", async () => {
    fetchSpy.mockResolvedValue(new Response("{}", { status: 401 }));
    const expired = vi.fn();
    window.addEventListener(SESSION_EXPIRED_EVENT, expired);

    // No token: the 401 just means "not signed in yet".
    await authedFetch("http://api.test/rpc");
    // Signed in, but a failed re-login is a wrong password, not an expired session.
    saveSession({ ownerToken: "jwt-live", ownerEmail: null });
    await authedFetch("http://api.test/vrooli.vrooli_bridge.v1.identity.IdentityService/Login");

    window.removeEventListener(SESSION_EXPIRED_EVENT, expired);
    expect(expired).not.toHaveBeenCalled();
  });
});
