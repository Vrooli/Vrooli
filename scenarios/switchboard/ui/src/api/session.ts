/**
 * Owner session for the console. Reads are unauthenticated operator probes;
 * writes (tier changes, bindings, gate answers, agent creation) require an
 * owner bearer token issued by the same-origin login facade the API exposes.
 *
 * The token lives in `sessionStorage` so it dies with the tab, never in a
 * cookie the API might read implicitly.
 */
import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "./client";

const STORAGE_KEY = "switchboard.session";

export interface Session {
  token: string;
  refresh_token?: string;
  subject: string;
  email?: string;
}

type Listener = (session: Session | null) => void;
const listeners = new Set<Listener>();

function read(): Session | null {
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<Session>;
    return typeof parsed.token === "string" && typeof parsed.subject === "string" ? { token: parsed.token, refresh_token: parsed.refresh_token, subject: parsed.subject, email: parsed.email } : null;
  } catch {
    return null;
  }
}

function write(session: Session | null) {
  try {
    if (session) window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    else window.sessionStorage.removeItem(STORAGE_KEY);
  } catch {
    // Storage may be unavailable (private mode); the in-memory copy still works for this page.
  }
  listeners.forEach((listener) => listener(session));
}

export const session = {
  get: read,
  token: (): string | undefined => read()?.token,
  set: write,
  clear: () => write(null),
  subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
};

export class LoginError extends Error {
  readonly status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = "LoginError";
    this.status = status;
  }
}

export async function login(email: string, password: string): Promise<Session> {
  const res = await fetch(buildApiUrl("/api/v1/session/login", { baseUrl: API_BASE }), {
    method: "POST",
    cache: "no-store",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const detail = (await res.text().catch(() => "")).trim();
    throw new LoginError(detail || `${res.status} ${res.statusText}`, res.status);
  }
  const body = (await res.json()) as Partial<Session>;
  if (typeof body.token !== "string" || typeof body.subject !== "string") {
    throw new LoginError("login response had no token", 502);
  }
  const next: Session = { token: body.token, refresh_token: body.refresh_token, subject: body.subject, email: body.email ?? email };
  write(next);
  return next;
}

export interface SessionStatus {
  authenticated: boolean;
  login_available: boolean;
}

export async function fetchSessionStatus(signal?: AbortSignal): Promise<SessionStatus> {
  const res = await fetch(buildApiUrl("/api/v1/session", { baseUrl: API_BASE }), { cache: "no-store", signal, headers: authHeaders() });
  if (!res.ok) return { authenticated: false, login_available: false };
  return (await res.json()) as SessionStatus;
}

/** Authorization header for the current session, or an empty object. */
export function authHeaders(): Record<string, string> {
  const token = session.token();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
