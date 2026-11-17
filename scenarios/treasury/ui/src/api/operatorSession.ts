import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { buildApiUrl } from "@vrooli/api-base";
import { AgentBudgetHeadroomSchema, type AgentBudgetHeadroom } from "@vrooli/proto-types/treasury/v1/authorization/authorization_pb";

import { API_BASE } from "./client";

export type OperatorSession = {
  authenticated: boolean;
  subject?: string;
};

export type OperatorSignInInput = {
  email: string;
  password: string;
  mfaChallenge?: string;
  totpCode?: string;
  recoveryCode?: string;
};

export type OperatorSignInResult = {
  authenticated: boolean;
  mfa_required?: boolean;
  mfa_challenge?: string;
};

export async function getOperatorSession(): Promise<OperatorSession> {
  try {
    return await sessionRequest("", "GET");
  } catch (reason) {
    if (reason instanceof OperatorSessionError && reason.status === 401) return { authenticated: false };
    throw reason;
  }
}

export async function signInOperator(input: OperatorSignInInput): Promise<OperatorSignInResult> {
  return sessionRequest("/sign-in", "POST", {
    email: input.email,
    password: input.password,
    mfa_challenge: input.mfaChallenge ?? "",
    totp_code: input.totpCode ?? "",
    recovery_code: input.recoveryCode ?? "",
  });
}

export async function refreshOperatorSession(): Promise<OperatorSession> {
  return sessionRequest("/refresh", "POST", {});
}

export async function signOutOperator(): Promise<void> {
  await sessionRequest("/sign-out", "POST", {});
}

export async function getOperatorBudgetHeadroom(budgetId: string): Promise<AgentBudgetHeadroom> {
  const query = new URLSearchParams({ budget_id: budgetId });
  const response = await fetch(buildApiUrl(`/api/v1/operator-session/budget-headroom?${query}`, { baseUrl: API_BASE }), {
    method: "GET",
    credentials: "include",
    cache: "no-store",
  });
  const payload = await response.json().catch(() => ({})) as { headroom?: unknown; message?: string };
  if (!response.ok) throw new Error(payload.message || `Treasury headroom request failed (${response.status})`);
  if (!payload.headroom || typeof payload.headroom !== "object") throw new Error("Treasury returned empty budget headroom");
  return fromJson(AgentBudgetHeadroomSchema, payload.headroom as JsonValue, { ignoreUnknownFields: true });
}

async function sessionRequest<T>(suffix: string, method: "GET" | "POST", body?: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(`/api/v1/operator-session${suffix}`, { baseUrl: API_BASE }), {
    method,
    credentials: "include",
    cache: "no-store",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const payload = await response.json().catch(() => ({})) as T & { message?: string };
  if (!response.ok) {
    throw new OperatorSessionError(payload.message || `Treasury session request failed (${response.status})`, response.status);
  }
  return payload;
}

class OperatorSessionError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = "OperatorSessionError";
  }
}
