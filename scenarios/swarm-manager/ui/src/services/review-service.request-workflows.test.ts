import { afterEach, describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createReviewService, type ReviewVerificationClient } from "./review-service";

type Fixture = { api: IApiClient; verifyAttemptEvidence: ReturnType<typeof vi.fn>; service: ReturnType<typeof createReviewService>; expectedGet: unknown[][]; expectedPost: unknown[][]; expectedVerify: unknown[][] };
const fixtures: Fixture[] = [];
function fixture(data: unknown = {}): Fixture {
  const api: IApiClient = { get: vi.fn().mockResolvedValue(data), post: vi.fn().mockResolvedValue(data), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
  const verifyAttemptEvidence = vi.fn().mockResolvedValue({});
  const f = { api, verifyAttemptEvidence, service: createReviewService(api, { verifyAttemptEvidence } as unknown as ReviewVerificationClient), expectedGet: [] as unknown[][], expectedPost: [] as unknown[][], expectedVerify: [] as unknown[][] };
  fixtures.push(f); return f;
}
afterEach(() => {
  for (const f of fixtures.splice(0)) {
    expect(vi.mocked(f.api.get).mock.calls).toEqual(f.expectedGet);
    expect(vi.mocked(f.api.post).mock.calls).toEqual(f.expectedPost);
    expect(f.verifyAttemptEvidence.mock.calls).toEqual(f.expectedVerify);
    for (const method of ["put", "patch", "delete"] as const) expect(f.api[method]).not.toHaveBeenCalled();
  }
});
describe("review request adapter serialization (fixture only; no Planner acceptance)", () => {
  it("retains round evidence bytes and defaults a missing list to empty", async () => {
    const rounds = [{ round: 2, status: "failed", evidence: [{ id: "fixture-proof", verified: false }], failure_reason: "original" }];
    const f = fixture({ rounds }); f.expectedGet = [["/backlog/execute/fixture-item/review"]]; expect(await f.service.listRounds("execute", "fixture-item")).toEqual(rounds);
    expect(f.api.get).toHaveBeenCalledTimes(1); expect(f.api.get).toHaveBeenCalledWith("/backlog/execute/fixture-item/review");
    const empty = fixture({}); empty.expectedGet = [["/backlog/execute/fixture-item/review"]]; expect(await empty.service.listRounds("execute", "fixture-item")).toEqual([]);
    expect(f.api.post).not.toHaveBeenCalled(); expect(empty.api.post).not.toHaveBeenCalled();
  });
  it.each([undefined, "fixture-proof"])("requests evidence with exact selected correlation (%s)", async (evidenceId) => {
    const f = fixture({ thread_id: "fixture-thread" });
    f.expectedPost = [["/backlog/execute/fixture-item/review/2/request", { message: "retain exact message", evidence_id: evidenceId }]];
    expect(await f.service.requestMoreEvidence("execute", "fixture-item", 2, "retain exact message", evidenceId)).toEqual({ thread_id: "fixture-thread" });
    expect(f.api.post).toHaveBeenCalledTimes(1); expect(f.api.post).toHaveBeenCalledWith("/backlog/execute/fixture-item/review/2/request", { message: "retain exact message", evidence_id: evidenceId });
    expect(f.verifyAttemptEvidence).not.toHaveBeenCalled(); expect(f.api.get).not.toHaveBeenCalled();
  });
  it("continues only the selected original round and thread", async () => {
    const f = fixture(); f.expectedPost = [["/backlog/execute/fixture-item/review/2/request/fixture-thread", { message: "exact continuation" }]]; await f.service.continueRequest("execute", "fixture-item", 2, "fixture-thread", "exact continuation");
    expect(f.api.post).toHaveBeenCalledTimes(1); expect(f.api.post).toHaveBeenCalledWith("/backlog/execute/fixture-item/review/2/request/fixture-thread", { message: "exact continuation" });
    expect(f.verifyAttemptEvidence).not.toHaveBeenCalled(); expect(f.api.get).not.toHaveBeenCalled();
  });
  it("dismisses only the selected thread without verifying evidence", async () => {
    const f = fixture(); f.expectedPost = [["/backlog/execute/fixture-item/review/2/request/fixture-thread/dismiss", {}]]; await f.service.dismissRequest("execute", "fixture-item", 2, "fixture-thread");
    expect(f.api.post).toHaveBeenCalledTimes(1); expect(f.api.post).toHaveBeenCalledWith("/backlog/execute/fixture-item/review/2/request/fixture-thread/dismiss", {});
    expect(f.verifyAttemptEvidence).not.toHaveBeenCalled();
  });
  it("constructs a capture path without transport effects", () => {
    const f = fixture(); expect(f.service.getCaptureUrl("execute", "fixture-item", "subdir/screenshot.png")).toBe("/backlog/execute/fixture-item/review/captures/subdir/screenshot.png");
    expect(f.api.get).not.toHaveBeenCalled(); expect(f.api.post).not.toHaveBeenCalled(); expect(f.verifyAttemptEvidence).not.toHaveBeenCalled();
  });
  it("preserves the same request refusal without retry or verification fallback", async () => {
    const f = fixture(); f.expectedPost = [["/backlog/execute/fixture-item/review/2/request", { message: "original", evidence_id: undefined }]]; const refusal = new Error("request refused"); vi.mocked(f.api.post).mockRejectedValue(refusal);
    await expect(f.service.requestMoreEvidence("execute", "fixture-item", 2, "original")).rejects.toBe(refusal);
    expect(f.api.post).toHaveBeenCalledTimes(1); expect(f.api.post).toHaveBeenCalledWith("/backlog/execute/fixture-item/review/2/request", { message: "original", evidence_id: undefined });
    expect(f.verifyAttemptEvidence).not.toHaveBeenCalled(); expect(f.api.get).not.toHaveBeenCalled();
  });
  it("trims declared annotation and preserves false verification (not caller identity)", async () => {
    const f = fixture(); f.expectedVerify = [[{ subjectKind: "backlog-item", subjectRef: "execute/fixture-item", roundNum: 2, evidenceId: "fixture-proof", verified: false, actor: "fixture-actor", reason: "original reason" }]]; await f.service.verifyEvidence("execute", "fixture-item", 2, "fixture-proof", false, "legacy-ignored", "  fixture-actor  ", "  original reason  ");
    expect(f.verifyAttemptEvidence).toHaveBeenCalledTimes(1); expect(f.verifyAttemptEvidence).toHaveBeenCalledWith({ subjectKind: "backlog-item", subjectRef: "execute/fixture-item", roundNum: 2, evidenceId: "fixture-proof", verified: false, actor: "fixture-actor", reason: "original reason" });
    expect(f.api.post).not.toHaveBeenCalled();
  });
  it("leaves absent annotation empty without synthesized actor or execution correlation", async () => {
    const f = fixture(); f.expectedVerify = [[{ subjectKind: "backlog-item", subjectRef: "execute/fixture-item", roundNum: 2, evidenceId: "fixture-proof", verified: false, actor: "", reason: "" }]]; await f.service.verifyEvidence("execute", "fixture-item", 2, "fixture-proof", false);
    expect(f.verifyAttemptEvidence).toHaveBeenCalledTimes(1); expect(f.verifyAttemptEvidence).toHaveBeenCalledWith({ subjectKind: "backlog-item", subjectRef: "execute/fixture-item", roundNum: 2, evidenceId: "fixture-proof", verified: false, actor: "", reason: "" });
    expect(f.api.post).not.toHaveBeenCalled();
  });
});
