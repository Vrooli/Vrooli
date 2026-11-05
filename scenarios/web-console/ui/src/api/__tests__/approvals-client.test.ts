import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ApprovalAlreadyResolvedError,
  ApprovalsUnauthorizedError,
  ApprovalsUnavailableError,
  answerApproval,
  countPendingApprovals,
  getApproval,
  listApprovals,
} from "../approvals";

vi.mock("../../lib/deviceIdentity", () => ({ deviceIdentity: () => ({ id: "device-1" }) }));
vi.mock("../../lib/auth", () => ({ getAccessToken: async () => "operator-token" }));

describe("approvals client", () => {
  beforeEach(() => vi.useFakeTimers().setSystemTime(new Date("2026-10-28T12:00:00Z")));
  afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });

  it("preserves decision context and excludes expired or answered requests from the pending count", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ approvals: [
      { id: "active", state: "pending", source: "agent-manager", source_event_type: "agent-manager.run-review.decision_requested.v1", options: [{ key: "accept" }], big_picture: "Context", deadline: "2026-10-29T00:00:00Z" },
      { id: "expired", state: "pending", deadline: "2026-10-27T00:00:00Z" },
      { id: "answered", state: "pending", answered_at: "2026-10-28T10:00:00Z" },
    ] }), { status: 200 }));
    vi.stubGlobal("fetch", fetcher);
    const results = await listApprovals();
    expect(results.map(({ id, state }) => ({ id, state }))).toEqual([
      { id: "active", state: "pending" }, { id: "expired", state: "expired" }, { id: "answered", state: "answered" },
    ]);
    expect(results[0]).toMatchObject({ source: "agent-manager", kind: "run-review", bigPicture: "Context", options: [{ key: "accept", label: "accept" }] });
    // A Response body is single-use, so give the count call a fresh response.
    fetcher.mockResolvedValueOnce(new Response(JSON.stringify({ approvals: [
      { id: "active", state: "pending", deadline: "2026-10-29T00:00:00Z" },
      { id: "expired", state: "pending", deadline: "2026-10-27T00:00:00Z" },
    ] }), { status: 200 }));
    await expect(countPendingApprovals()).resolves.toBe(1);
    const [url, init] = fetcher.mock.calls[0]!;
    expect(String(url)).toContain("/api/v1/approvals?open_only=true&limit=100");
    expect(init.headers.get("Authorization")).toBe("Bearer operator-token");
    expect(init.headers.get("X-Vrooli-Device-Id")).toBe("device-1");
    expect(init.cache).toBe("no-store");
  });

  it("uses the same-origin facade for lookup and the chosen answer", async () => {
    const fetcher = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ approval: { id: "ask/1", state: "answered", source: "other" } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetcher);
    await expect(getApproval("ask/1")).resolves.toMatchObject({ id: "ask/1", state: "answered", source: "unknown" });
    expect(String(fetcher.mock.calls[0]![0])).toContain("/api/v1/approvals/ask%2F1");
    await answerApproval("ask/1", "reject", "please revise");
    expect(JSON.parse(fetcher.mock.calls[1]![1].body)).toEqual({ ask_id: "ask/1", answer: "reject", note: "please revise" });
  });

  it.each([
    [401, ApprovalsUnauthorizedError], [403, ApprovalsUnauthorizedError],
    [409, ApprovalAlreadyResolvedError], [503, ApprovalsUnavailableError],
  ])("exposes status %i as an actionable error", async (status, errorType) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status })));
    await expect(answerApproval("ask", "approve")).rejects.toBeInstanceOf(errorType);
  });

  it("does not invent a decision when the facade returns none", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("{}", { status: 200 })));
    await expect(getApproval("missing")).rejects.toThrow("decision not found");
  });
});
