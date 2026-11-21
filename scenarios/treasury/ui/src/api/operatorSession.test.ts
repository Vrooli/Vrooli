import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getOperatorBudgetHeadroom, getOperatorSession, signInOperator, signOutOperator } from "./operatorSession";

describe("api/operatorSession budget headroom", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reads operator-owned budget headroom with the session cookie", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({
      headroom: {
        budgetId: "research-budget",
        currency: "USD",
        availableMinor: "100",
        fundingCreditedMinor: "260",
        fundingRemainingMinor: "260",
      },
    }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));

    const headroom = await getOperatorBudgetHeadroom("research-budget");

    expect(headroom.budgetId).toBe("research-budget");
    expect(headroom.fundingCreditedMinor).toBe(260n);
    expect(headroom.fundingRemainingMinor).toBe(260n);
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/api\/v1\/operator-session\/budget-headroom\?budget_id=research-budget$/);
    expect(init.credentials).toBe("include");
    expect(new Headers(init.headers).get("Authorization")).toBeNull();
  });

  it("sends sign-in credentials to the session endpoint and keeps the response token-free", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ authenticated: false, mfa_required: true, mfa_challenge: "challenge-1" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));

    const result = await signInOperator({ email: "operator@example.test", password: "secret", mfaChallenge: "challenge-1", totpCode: "123456", recoveryCode: "" });

    expect(result.mfa_required).toBe(true);
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/api\/v1\/operator-session\/sign-in$/);
    expect(init.credentials).toBe("include");
    expect(JSON.parse(String(init.body))).toEqual({ email: "operator@example.test", password: "secret", mfa_challenge: "challenge-1", totp_code: "123456", recovery_code: "" });
    expect(JSON.stringify(result)).not.toMatch(/token|secret/i);
  });

  it("treats an unauthenticated session check as signed out so refresh can run", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ message: "session required" }), {
      status: 401,
      headers: { "Content-Type": "application/json" },
    }));

    await expect(getOperatorSession()).resolves.toEqual({ authenticated: false });
  });

  it("signs out through the cookie session and accepts an empty 204 response", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(null, { status: 204 }));

    await expect(signOutOperator()).resolves.toBeUndefined();

    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/api\/v1\/operator-session\/sign-out$/);
    expect(init.credentials).toBe("include");
    expect(init.body).toBe("{}");
  });

  it("rejects an empty headroom response instead of showing synthetic funding data", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({}), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));

    await expect(getOperatorBudgetHeadroom("research-budget"))
      .rejects.toThrow("Treasury returned empty budget headroom");
  });
});
