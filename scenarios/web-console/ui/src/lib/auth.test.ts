import { afterEach, describe, expect, it, vi } from "vitest";
import { completeWebAuthCallback, signOut, startSignIn } from "./auth";
import { LANDING_PAGE_URL } from "../shared/upgradeDestination";

const { authRequest } = vi.hoisted(() => ({ authRequest: vi.fn() }));
vi.mock("@vrooli/react-component-library/AuthClient/1.0.0", () => ({
  AuthClient: class {
    request = authRequest;
  },
}));

const ACCESS_TOKEN_KEY = "vrooli.web.access-token";
const AUTH_STATE_KEY = "vrooli.web.auth-state";

afterEach(() => {
  vi.unstubAllGlobals();
  sessionStorage.removeItem(AUTH_STATE_KEY);
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
  authRequest.mockReset();
});

it("identifies Aquila to sign-in while retaining the callback and technical state key", async () => {
  const assign = vi.fn();
  const callback = "https://console.example.test/?workspace=current";
  vi.stubGlobal("window", { location: { href: callback, assign } });

  await startSignIn();

  expect(assign).toHaveBeenCalledTimes(1);
  const destination = new URL(String(assign.mock.calls[0]?.[0]));
  expect(destination.origin).toBe(new URL(LANDING_PAGE_URL).origin);
  expect(destination.pathname).toBe("/auth/login");
  expect(destination.searchParams.get("app")).toBe("Aquila");
  expect(destination.searchParams.get("redirect_uri")).toBe(callback);
  const state = sessionStorage.getItem("vrooli.web.auth-state");
  expect(state).toBeTruthy();
  expect(destination.searchParams.get("state")).toBe(state);
});

describe("completeWebAuthCallback", () => {
  function stubCallbackWindow(hash: string): { replaceState: ReturnType<typeof vi.fn> } {
    const replaceState = vi.fn();
    vi.stubGlobal("window", {
      location: { hash, pathname: "/app", search: "?x=1" },
      history: { replaceState },
    });
    vi.stubGlobal("document", { title: "WC" });
    return { replaceState };
  }

  it("returns false when no auth fragment is present", async () => {
    stubCallbackWindow("");
    await expect(completeWebAuthCallback()).resolves.toBe(false);
    expect(authRequest).not.toHaveBeenCalled();
  });

  it("provisions a session and stores the access token when state matches", async () => {
    const expiresAt = new Date(Date.now() + 3_600_000).toISOString();
    sessionStorage.setItem(AUTH_STATE_KEY, "state-1");
    const { replaceState } = stubCallbackWindow(
      `#access_token=at&refresh_token=rt&expires_at=${encodeURIComponent(expiresAt)}&state=state-1`,
    );
    authRequest.mockResolvedValue(undefined);

    await expect(completeWebAuthCallback()).resolves.toBe(true);
    expect(authRequest).toHaveBeenCalledWith(
      "/api/v1/auth/subscription/session",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ refresh_token: "rt" }) }),
    );
    expect(replaceState).toHaveBeenCalled();
    expect(JSON.parse(sessionStorage.getItem(ACCESS_TOKEN_KEY)!)).toEqual({ accessToken: "at", expiresAt });
    // Single-use: the technical state key is cleared.
    expect(sessionStorage.getItem(AUTH_STATE_KEY)).toBeNull();
  });

  it("rejects a callback whose state does not match the stored value", async () => {
    sessionStorage.setItem(AUTH_STATE_KEY, "expected");
    stubCallbackWindow("#access_token=at&refresh_token=rt&expires_at=2099-01-01T00:00:00Z&state=forged");
    await expect(completeWebAuthCallback()).rejects.toThrow("state validation failed");
    expect(authRequest).not.toHaveBeenCalled();
  });

  it("rejects an incomplete or expired callback", async () => {
    sessionStorage.setItem(AUTH_STATE_KEY, "s");
    stubCallbackWindow("#access_token=at&refresh_token=rt&expires_at=1999-01-01T00:00:00Z&state=s");
    await expect(completeWebAuthCallback()).rejects.toThrow("incomplete or expired");
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
