import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getBudgetHeadroom } from "./funding";

describe("api/funding.getBudgetHeadroom", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reads Treasury headroom using the verified agent identity token", async () => {
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

    const headroom = await getBudgetHeadroom("verified-agent-token", "research-budget");

    expect(headroom.budgetId).toBe("research-budget");
    expect(headroom.fundingCreditedMinor).toBe(260n);
    expect(headroom.fundingRemainingMinor).toBe(260n);
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/vrooli\.treasury\.v1\.authorization\.AgentSpend\/GetBudgetHeadroom$/);
    expect(new Headers(init.headers).get("X-Agent-Identity-Token")).toBe("verified-agent-token");
  });

  it("rejects an empty headroom response instead of showing synthetic funding data", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({}), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));

    await expect(getBudgetHeadroom("verified-agent-token", "research-budget"))
      .rejects.toThrow("Treasury returned empty budget headroom");
  });
});
