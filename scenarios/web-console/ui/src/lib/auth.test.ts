import { afterEach, describe, expect, it, vi } from "vitest";
import { completeWebAuthCallback, signOut, startSignIn } from "./auth";

const { authRequest } = vi.hoisted(() => ({ authRequest: vi.fn() }));
vi.mock("@vrooli/react-component-library/AuthClient/1.0.0", () => ({
  AuthClient: class {
    request = authRequest;
  },
}));

const ACCESS_TOKEN_KEY = "vrooli.web.access-token";
const AUTH_STATE_KEY = "vrooli.web.auth-state";
const AUTH_VERIFIER_KEY = "vrooli.web.auth-verifier";
const AUTH_REDIRECT_KEY = "vrooli.web.auth-redirect";

afterEach(() => {
  vi.unstubAllGlobals();
  sessionStorage.removeItem(AUTH_STATE_KEY);
  sessionStorage.removeItem(AUTH_VERIFIER_KEY);
  sessionStorage.removeItem(AUTH_REDIRECT_KEY);
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
  authRequest.mockReset();
});

it("starts a PKCE sign-in and retains only the callback verifier and technical state locally", async () => {
  const assign = vi.fn();
  const callback = "http://127.0.0.1:3210/?workspace=current";
  vi.stubGlobal("window", { location: { href: callback, assign } });
  vi.stubGlobal("crypto", {
    randomUUID: () => "state-1",
    getRandomValues: (bytes: Uint8Array) => bytes.fill(7),
    subtle: { digest: vi.fn(async () => new Uint8Array(32).fill(9).buffer) },
  });
  authRequest.mockResolvedValueOnce({ authorization_url: "http://127.0.0.1:15000/auth/login" });

  await startSignIn();

  expect(assign).toHaveBeenCalledTimes(1);
  const destination = new URL(String(assign.mock.calls[0]?.[0]));
  expect(destination.origin).toBe("http://127.0.0.1:15000");
  expect(destination.pathname).toBe("/auth/login");
  expect(destination.searchParams.get("app")).toBe("Aquila");
  expect(destination.searchParams.get("redirect_uri")).toBe(callback);
  expect(destination.searchParams.get("code_challenge_method")).toBe("S256");
  expect(destination.searchParams.get("code_challenge")).toBeTruthy();
  expect(destination.searchParams.get("code_challenge")).not.toBe(sessionStorage.getItem(AUTH_VERIFIER_KEY));
  const state = sessionStorage.getItem("vrooli.web.auth-state");
  expect(state).toBeTruthy();
  expect(destination.searchParams.get("state")).toBe(state);
  expect(sessionStorage.getItem(AUTH_REDIRECT_KEY)).toBe(callback);
  expect(authRequest).toHaveBeenCalledWith("/api/v1/auth/subscription/authorization");
});

it("refuses a hosted callback that LPBS cannot redeem as a loopback app", async () => {
  vi.stubGlobal("window", { location: { href: "https://console.example.test/", assign: vi.fn() } });
  await expect(startSignIn()).rejects.toThrow("requires a local loopback address");
  expect(sessionStorage.getItem(AUTH_STATE_KEY)).toBeNull();
  expect(sessionStorage.getItem(AUTH_VERIFIER_KEY)).toBeNull();
});

describe("completeWebAuthCallback", () => {
  function stubCallbackWindow(search: string): { replaceState: ReturnType<typeof vi.fn> } {
    const replaceState = vi.fn();
    vi.stubGlobal("window", {
      location: { hash: "", pathname: "/app", search },
      history: { replaceState },
    });
    vi.stubGlobal("document", { title: "WC" });
    return { replaceState };
  }

  it("returns false when no authorization code is present", async () => {
    stubCallbackWindow("");
    await expect(completeWebAuthCallback()).resolves.toBe(false);
    expect(authRequest).not.toHaveBeenCalled();
  });

  it("exchanges a PKCE code through the same-origin API and stores only the access token", async () => {
    const expiresAt = new Date(Date.now() + 3_600_000).toISOString();
    sessionStorage.setItem(AUTH_STATE_KEY, "state-1");
    sessionStorage.setItem(AUTH_VERIFIER_KEY, "verifier-1");
    sessionStorage.setItem(AUTH_REDIRECT_KEY, "http://127.0.0.1:3210/?workspace=current");
    const { replaceState } = stubCallbackWindow("?workspace=current&code=one-use&state=state-1");
    authRequest.mockResolvedValue({ access_token: "at", expires_at: expiresAt });

    await expect(completeWebAuthCallback()).resolves.toBe(true);
    expect(authRequest).toHaveBeenCalledWith(
      "/api/v1/auth/subscription/session/exchange",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ code: "one-use", code_verifier: "verifier-1", redirect_uri: "http://127.0.0.1:3210/?workspace=current" }) }),
    );
    expect(replaceState).toHaveBeenCalledWith({}, "WC", "/app?workspace=current");
    expect(JSON.parse(sessionStorage.getItem(ACCESS_TOKEN_KEY)!)).toEqual({ accessToken: "at", expiresAt });
    // Single-use: state and verifier are cleared after the callback.
    expect(sessionStorage.getItem(AUTH_STATE_KEY)).toBeNull();
    expect(sessionStorage.getItem(AUTH_VERIFIER_KEY)).toBeNull();
    expect(sessionStorage.getItem(AUTH_REDIRECT_KEY)).toBeNull();
  });

  it("rejects a callback whose state does not match the stored value", async () => {
    sessionStorage.setItem(AUTH_STATE_KEY, "expected");
    sessionStorage.setItem(AUTH_VERIFIER_KEY, "verifier");
    sessionStorage.setItem(AUTH_REDIRECT_KEY, "http://127.0.0.1:3210/");
    stubCallbackWindow("?code=one-use&state=forged");
    await expect(completeWebAuthCallback()).rejects.toThrow("state validation failed");
    expect(authRequest).not.toHaveBeenCalled();
  });

  it("rejects an incomplete or expired callback", async () => {
    sessionStorage.setItem(AUTH_STATE_KEY, "s");
    stubCallbackWindow("?code=one-use&state=s");
    await expect(completeWebAuthCallback()).rejects.toThrow("incomplete or expired");
  });

  it("does not return a failed exchange as a signed-in session", async () => {
    sessionStorage.setItem(AUTH_STATE_KEY, "state-1");
    sessionStorage.setItem(AUTH_VERIFIER_KEY, "verifier-1");
    sessionStorage.setItem(AUTH_REDIRECT_KEY, "http://127.0.0.1:3210/");
    stubCallbackWindow("?code=one-use&state=state-1");
    authRequest.mockResolvedValue({ access_token: "", expires_at: "" });
    await expect(completeWebAuthCallback()).rejects.toThrow("incomplete or expired");
    expect(sessionStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
  });
});

describe("signOut", () => {
  it("revokes the server session and clears the local token", async () => {
    sessionStorage.setItem(ACCESS_TOKEN_KEY, JSON.stringify({ accessToken: "at", expiresAt: "2099-01-01T00:00:00Z" }));
    authRequest.mockResolvedValue(undefined);
    await signOut();
    expect(authRequest).toHaveBeenCalledWith(
      "/api/v1/auth/subscription/session",
      expect.objectContaining({ method: "DELETE" }),
    );
    expect(sessionStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
  });

  it("still clears the local token when the revoke request fails", async () => {
    sessionStorage.setItem(ACCESS_TOKEN_KEY, JSON.stringify({ accessToken: "at", expiresAt: "2099-01-01T00:00:00Z" }));
    authRequest.mockRejectedValue(new Error("offline"));
    await expect(signOut()).rejects.toThrow("offline");
    expect(sessionStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
  });
});
