// Provider-neutral browser status helpers. This module contains display
// semantics only; it never reads, parses, or stores an Access assertion.
export type SharedAuthState =
  | "signed_out"
  | "verified"
  | "agent"
  | "service"
  | "conflict"
  | "expired"
  | "error";

export interface SharedAuthStatus {
  authenticated: boolean;
  // Authentication alone is not mutation authority. Providers may report an
  // authenticated agent or service, so consumers should prefer this explicit
  // capability when deciding whether a write is enabled.
  canMutate?: boolean;
  authSource?: string;
  authState: SharedAuthState | string;
  email?: string;
  reason?: string;
  recoveryUrl?: string;
}

export function hasHumanAuthority(status: SharedAuthStatus): boolean {
  return status.canMutate ?? (status.authenticated && status.authState === "verified");
}

export function authStatusLabel(status: SharedAuthStatus): string {
  if (status.authState === "expired") {
    return status.authSource === "cloudflare_access"
      ? "Cloudflare Access session expired"
      : "Session expired";
  }
  if (status.authState === "conflict") return "Conflicting authentication";
  if (status.authState === "service") return "Service authentication is read-only";
  if (status.authState === "agent") return "Agent authentication is read-only";
  if (hasHumanAuthority(status)) return "Human authority ready";
  return "Read-only authority";
}

export function isRecoveryState(status: SharedAuthStatus): boolean {
  return !hasHumanAuthority(status) && ["signed_out", "expired", "error", "conflict"].includes(status.authState);
}
