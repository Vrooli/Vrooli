/**
 * Browser-side client for the sibling **scenario-authenticator** scenario.
 *
 * Device Sync Hub does not own identity — the owner is a scenario-authenticator
 * account (see DECISIONS.md, two-credential model). The first-run owner login
 * therefore posts straight to scenario-authenticator, NOT to this hub's own API
 * (whose transport is `api/transport.ts`). On success the returned owner JWT is
 * stored in the session and rides every owner-gated DevicesService RPC.
 *
 * Reaching the authenticator from the browser:
 *   1. an explicit `VITE_AUTH_API_BASE` (or alias) build/runtime env var, then
 *   2. the app-monitor proxy convention (`/apps/scenario-authenticator/proxy`)
 *      when this UI is itself served under `/apps/.../proxy`, else
 *   3. unresolved (`null`) — the login form then degrades to the Advanced
 *      owner-token paste, which needs no authenticator URL.
 */

const ENV_CANDIDATE_KEYS = [
  "VITE_AUTH_API_BASE",
  "VITE_AUTH_URL",
  "VITE_AUTH_BASE_URL",
  "VITE_AUTHENTICATOR_URL",
  "VITE_SCENARIO_AUTHENTICATOR_URL",
] as const;

const AUTH_SCENARIO_PROXY = "/apps/scenario-authenticator/proxy";

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/u, "");
}

function readEnv(key: string): string | undefined {
  const raw = (import.meta as unknown as { env?: Record<string, unknown> }).env?.[key];
  return typeof raw === "string" && raw.trim() ? raw.trim() : undefined;
}

/**
 * Resolve the authenticator's API origin, or null when this deployment hasn't
 * been told where the authenticator lives. Callers treat null as "owner login
 * unavailable here — offer the token-paste fallback".
 */
export function resolveAuthenticatorBaseUrl(): string | null {
  for (const key of ENV_CANDIDATE_KEYS) {
    const candidate = readEnv(key);
    if (candidate) return stripTrailingSlash(candidate);
  }
  if (typeof window !== "undefined") {
    const { origin, pathname } = window.location;
    if (origin && pathname.includes("/apps/")) {
      return `${stripTrailingSlash(origin)}${AUTH_SCENARIO_PROXY}`;
    }
  }
  return null;
}

/** Distinguishable failure modes the login form renders differently. */
export type AuthErrorCode = "invalid_credentials" | "unavailable";

export class AuthError extends Error {
  readonly code: AuthErrorCode;
  constructor(code: AuthErrorCode, message: string) {
    super(message);
    this.name = "AuthError";
    this.code = code;
  }
}

export interface OwnerLoginInput {
  email: string;
  password: string;
}

export interface OwnerIdentity {
  token: string;
  email?: string;
  userId?: string;
}

/** scenario-authenticator AuthResponse (api/models/user.go) — owner JWT is `token`. */
interface AuthResponseJson {
  success?: boolean;
  token?: string;
  message?: string;
  user?: { id?: string; email?: string };
}

/** scenario-authenticator ValidationResponse (api/models/user.go). */
interface ValidationResponseJson {
  valid?: boolean;
  user_id?: string;
  email?: string;
  roles?: string[];
}

export interface OwnerValidation {
  valid: boolean;
  userId?: string;
  email?: string;
  roles?: string[];
}

/** POST credentials to scenario-authenticator and return the owner JWT. */
export async function loginOwner(base: string, input: OwnerLoginInput): Promise<OwnerIdentity> {
  let res: Response;
  try {
    res = await fetch(`${stripTrailingSlash(base)}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: input.email, password: input.password }),
    });
  } catch (cause) {
    throw new AuthError("unavailable", `could not reach scenario-authenticator: ${String(cause)}`);
  }
  if (res.status === 401 || res.status === 403) {
    throw new AuthError("invalid_credentials", "authenticator rejected the credentials");
  }
  if (!res.ok) {
    throw new AuthError("unavailable", `authenticator returned ${res.status}`);
  }
  const data = (await res.json().catch(() => ({}))) as AuthResponseJson;
  if (!data.success || !data.token) {
    throw new AuthError("invalid_credentials", data.message || "authenticator returned no token");
  }
  return { token: data.token, email: data.user?.email, userId: data.user?.id };
}

/** Validate a stored owner token against scenario-authenticator. */
export async function validateOwner(base: string, token: string): Promise<OwnerValidation> {
  let res: Response;
  try {
    res = await fetch(`${stripTrailingSlash(base)}/api/v1/auth/validate`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  } catch (cause) {
    throw new AuthError("unavailable", `could not reach scenario-authenticator: ${String(cause)}`);
  }
  if (!res.ok) {
    throw new AuthError("unavailable", `validate returned ${res.status}`);
  }
  const data = (await res.json().catch(() => ({}))) as ValidationResponseJson;
  return {
    valid: Boolean(data.valid),
    userId: data.user_id,
    email: data.email,
    roles: data.roles,
  };
}
