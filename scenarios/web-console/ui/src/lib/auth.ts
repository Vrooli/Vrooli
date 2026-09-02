import { AuthClient } from "@vrooli/react-component-library/AuthClient/1.0.0";
import { LANDING_PAGE_URL } from "../shared/upgradeDestination";

const ACCESS_TOKEN_KEY = "vrooli.web.access-token";
const AUTH_STATE_KEY = "vrooli.web.auth-state";

interface StoredAccessToken {
  accessToken: string;
  expiresAt: string;
}

const authClient = new AuthClient({ baseURL: window.location.origin });

function readStoredAccessToken(): StoredAccessToken | null {
  try {
    const raw = sessionStorage.getItem(ACCESS_TOKEN_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<StoredAccessToken>;
    if (typeof parsed.accessToken !== "string" || typeof parsed.expiresAt !== "string") return null;
    if (Date.now() >= new Date(parsed.expiresAt).getTime() - 30_000) {
      sessionStorage.removeItem(ACCESS_TOKEN_KEY);
      return null;
    }
    return { accessToken: parsed.accessToken, expiresAt: parsed.expiresAt };
  } catch {
    return null;
  }
}

export function getWebAccessToken(): string | null {
  return readStoredAccessToken()?.accessToken ?? null;
}

function clearWebAccessToken(): void {
  try {
    sessionStorage.removeItem(ACCESS_TOKEN_KEY);
  } catch {
    // A disabled session store must not prevent local sign-out.
  }
}

export async function getAccessToken(): Promise<string | null> {
  if (window.desktop?.auth?.getAccessToken) {
    return window.desktop.auth.getAccessToken();
  }
  return getWebAccessToken();
}

export async function completeWebAuthCallback(): Promise<boolean> {
  if (window.desktop?.auth) return false;
  const hash = window.location.hash.startsWith("#") ? window.location.hash.slice(1) : "";
  const params = new URLSearchParams(hash);
  const accessToken = params.get("access_token");
  const refreshToken = params.get("refresh_token");
  const expiresAt = params.get("expires_at");
  if (!accessToken && !refreshToken && !expiresAt) return false;

  const expectedState = sessionStorage.getItem(AUTH_STATE_KEY);
  const receivedState = params.get("state");
  sessionStorage.removeItem(AUTH_STATE_KEY);
  window.history.replaceState({}, document.title, `${window.location.pathname}${window.location.search}`);
  if (!expectedState || expectedState !== receivedState) {
    throw new Error("Authentication callback state validation failed");
  }
  if (!accessToken || !refreshToken || !expiresAt || Number.isNaN(new Date(expiresAt).getTime()) || Date.now() >= new Date(expiresAt).getTime()) {
    throw new Error("Authentication callback was incomplete or expired");
  }

  await authClient.request("/api/v1/auth/subscription/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  sessionStorage.setItem(ACCESS_TOKEN_KEY, JSON.stringify({ accessToken, expiresAt } satisfies StoredAccessToken));
  return true;
}

export async function startSignIn(): Promise<void> {
  if (window.desktop?.auth) {
    const result = await window.desktop.auth.signIn();
    sessionStorage.setItem(AUTH_STATE_KEY, result.state);
    return;
  }
  const state = crypto.randomUUID();
  sessionStorage.setItem(AUTH_STATE_KEY, state);
  const authURL = new URL("/auth/login", LANDING_PAGE_URL);
  authURL.searchParams.set("redirect_uri", window.location.href);
  authURL.searchParams.set("app", "Vrooli Web Console");
  authURL.searchParams.set("state", state);
  window.location.assign(authURL.toString());
}

export async function signOut(): Promise<void> {
  if (window.desktop?.auth) {
    await window.desktop.auth.signOut();
  } else {
    try {
      await authClient.request("/api/v1/auth/subscription/session", { method: "DELETE" });
    } finally {
      clearWebAccessToken();
    }
  }
}
