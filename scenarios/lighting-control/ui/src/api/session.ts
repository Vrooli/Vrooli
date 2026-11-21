import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE } from "./client";

export interface BrowserSession {
  authenticated: boolean;
  email?: string;
  subject?: string;
  expiresAt?: string;
  mfaRequired?: boolean;
  mfaChallenge?: string;
  message?: string;
}
export interface SignIn {
  email: string;
  password: string;
  totpCode?: string;
  recoveryCode?: string;
  mfaChallenge?: string;
}
async function request(action?: "login" | "refresh" | "logout", body?: SignIn): Promise<BrowserSession> {
  const response = await fetch(buildApiUrl(`/api/session${action ? `/${action}` : ""}`, { baseUrl: API_BASE }), {
    method: action ? "POST" : "GET",
    credentials: "same-origin",
    cache: "no-store",
    headers: action ? { "X-Vrooli-Session": "1", "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  const result = await response.json() as BrowserSession;
  if (!response.ok) throw new Error(result.message || `Session request failed (${response.status}).`);
  return result;
}
export const sessionApi = {
  status: () => request(),
  login: (body: SignIn) => request("login", body),
  refresh: () => request("refresh"),
  logout: () => request("logout"),
};
