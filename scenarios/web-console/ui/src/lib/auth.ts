import { AuthClient } from "@vrooli/react-component-library/AuthClient/1.0.0";

const ACCESS_TOKEN_KEY = "vrooli.web.access-token";
const AUTH_STATE_KEY = "vrooli.web.auth-state";
const AUTH_VERIFIER_KEY = "vrooli.web.auth-verifier";
const AUTH_REDIRECT_KEY = "vrooli.web.auth-redirect";

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
  const params = new URLSearchParams(window.location.search);
  const code = params.get("code");
  const receivedState = params.get("state");
  if (!code && !receivedState) return false;

  const expectedState = sessionStorage.getItem(AUTH_STATE_KEY);
  const verifier = sessionStorage.getItem(AUTH_VERIFIER_KEY);
  const redirectURI = sessionStorage.getItem(AUTH_REDIRECT_KEY);
  sessionStorage.removeItem(AUTH_STATE_KEY);
  sessionStorage.removeItem(AUTH_VERIFIER_KEY);
  sessionStorage.removeItem(AUTH_REDIRECT_KEY);
  params.delete("code");
  params.delete("state");
  const remainingSearch = params.toString();
  window.history.replaceState({}, document.title, `${window.location.pathname}${remainingSearch ? `?${remainingSearch}` : ""}`);
  if (!expectedState || expectedState !== receivedState) {
    throw new Error("Authentication callback state validation failed");
  }
  if (!code || !verifier || !redirectURI) {
    throw new Error("Authentication callback was incomplete or expired");
  }

  const result = await authClient.request<{ access_token?: string; expires_at?: string }>("/api/v1/auth/subscription/session/exchange", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code, code_verifier: verifier, redirect_uri: redirectURI }),
  });
  const accessToken = result?.access_token;
  const expiresAt = result?.expires_at;
  if (!accessToken || !expiresAt || Number.isNaN(new Date(expiresAt).getTime()) || Date.now() >= new Date(expiresAt).getTime()) {
    throw new Error("Authentication callback was incomplete or expired");
  }
  sessionStorage.setItem(ACCESS_TOKEN_KEY, JSON.stringify({ accessToken, expiresAt } satisfies StoredAccessToken));
  return true;
}

function toBase64URL(bytes: Uint8Array): string {
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function isLoopbackCallback(url: URL): boolean {
  return url.protocol === "http:" && ["localhost", "127.0.0.1", "[::1]", "::1"].includes(url.hostname) && Boolean(url.port);
}

export async function startSignIn(): Promise<void> {
  if (window.desktop?.auth) {
    const result = await window.desktop.auth.signIn();
    sessionStorage.setItem(AUTH_STATE_KEY, result.state);
    return;
  }
  const callback = new URL(window.location.href);
  callback.hash = "";
  if (!isLoopbackCallback(callback)) {
    throw new Error("Web Console account sign-in requires a local loopback address");
  }
  const redirectURI = callback.toString();
  const state = crypto.randomUUID();
  const verifier = toBase64URL(crypto.getRandomValues(new Uint8Array(32)));
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  const challenge = toBase64URL(new Uint8Array(digest));
  const authority = await authClient.request<{ authorization_url?: string }>("/api/v1/auth/subscription/authorization");
  if (!authority?.authorization_url) throw new Error("Subscription authority is unavailable");
  const authURL = new URL(authority.authorization_url);
  if (!["http:", "https:"].includes(authURL.protocol) || !authURL.pathname.endsWith("/auth/login")) {
    throw new Error("Subscription authority returned an invalid sign-in address");
  }
  sessionStorage.setItem(AUTH_STATE_KEY, state);
  sessionStorage.setItem(AUTH_VERIFIER_KEY, verifier);
  sessionStorage.setItem(AUTH_REDIRECT_KEY, redirectURI);
  authURL.searchParams.set("redirect_uri", redirectURI);
  authURL.searchParams.set("app", "Aquila");
  authURL.searchParams.set("state", state);
  authURL.searchParams.set("code_challenge", challenge);
  authURL.searchParams.set("code_challenge_method", "S256");
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
