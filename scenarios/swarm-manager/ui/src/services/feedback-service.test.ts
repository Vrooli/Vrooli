import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  createFeedbackService,
  FeedbackBusyError,
  FeedbackLockConflictError,
  type IFeedbackService,
} from "./feedback-service";
import { ApiError, type IApiClient } from "../lib/api-client";
import type { FeedbackRound } from "../types";

function makeRound(overrides: Partial<FeedbackRound> = {}): FeedbackRound {
  return {
    initiative_name: "i1",
    number: 1,
    slug: "round",
    type: "feedback",
    status: "awaiting_user",
    submission: { text: "hello", created_at: "2026-04-23T00:00:00Z" },
    thread: [],
    proposals: [],
    created_at: "2026-04-23T00:00:00Z",
    updated_at: "2026-04-23T00:00:00Z",
    ...overrides,
  };
}

describe("Feedback Service", () => {
  let api: IApiClient;
  let service: IFeedbackService;

  beforeEach(() => {
    api = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    service = createFeedbackService(api);
  });

  it("lists rounds for an initiative", async () => {
    vi.mocked(api.get).mockResolvedValue({ rounds: [makeRound()], count: 1 });
    const result = await service.list("i1");
    expect(api.get).toHaveBeenCalledWith("/initiatives/i1/feedback");
    expect(result).toHaveLength(1);
  });

  it("returns empty array when server omits rounds", async () => {
    vi.mocked(api.get).mockResolvedValue({});
    expect(await service.list("i1")).toEqual([]);
  });

  it("fetches a single round by number", async () => {
    vi.mocked(api.get).mockResolvedValue(makeRound({ number: 2 }));
    const result = await service.get("i1", 2);
    expect(api.get).toHaveBeenCalledWith("/initiatives/i1/feedback/2");
    expect(result.number).toBe(2);
  });

  it("starts a round as JSON when no files are attached", async () => {
    const created = makeRound();
    vi.mocked(api.post).mockResolvedValue(created);
    await service.start("i1", {
      type: "feedback",
      text: "please look",
      decidedBy: "me",
    });
    expect(api.post).toHaveBeenCalledWith("/initiatives/i1/feedback", expect.objectContaining({
      type: "feedback",
      text: "please look",
      decided_by: "me",
    }));
    // Not FormData:
    const bodyArg = vi.mocked(api.post).mock.calls[0]![1];
    expect(bodyArg).not.toBeInstanceOf(FormData);
  });

  it("starts a round as multipart form data when files are present", async () => {
    const created = makeRound();
    vi.mocked(api.post).mockResolvedValue(created);
    const file = new File(["dummy"], "shot.png", { type: "image/png" });
    await service.start("i1", {
      type: "feedback",
      text: "with screenshot",
      files: [file],
      override: true,
    });
    const body = vi.mocked(api.post).mock.calls[0]![1];
    expect(body).toBeInstanceOf(FormData);
    const fd = body as FormData;
    expect(fd.get("text")).toBe("with screenshot");
    expect(fd.get("override")).toBe("true");
    expect(fd.getAll("files")).toHaveLength(1);
  });

  it("maps a 409 conflict with a holder body into FeedbackLockConflictError", async () => {
    const holderError = new ApiError(
      "http",
      JSON.stringify({ error: "initiative is locked", holder: { run_id: "r1", purpose: "feedback" } }),
      { status: 409 },
    );
    vi.mocked(api.post).mockRejectedValue(holderError);
    await expect(
      service.start("i1", { type: "feedback", text: "hi" }),
    ).rejects.toBeInstanceOf(FeedbackLockConflictError);
  });

  // A 409 whose body carries `activities` instead of `holder` describes
  // busy backlog items rather than a competing initiative-level round.
  // The UI renders a different warning for each — so the service must
  // preserve that distinction instead of collapsing both into a single
  // "initiative is locked" error.
  it("maps a 409 conflict with an activities body into FeedbackBusyError", async () => {
    const busyError = new ApiError(
      "http",
      JSON.stringify({
        error: "initiative has active item-level agent runs",
        activities: [
          { ref: "execute/foo", run_id: "run-foo", purpose: "execute" },
          { ref: "research/bar", run_id: "run-bar", purpose: "workshop" },
        ],
      }),
      { status: 409 },
    );
    vi.mocked(api.post).mockRejectedValue(busyError);
    const err = await service
      .start("i1", { type: "feedback", text: "hi" })
      .catch((e) => e);
    expect(err).toBeInstanceOf(FeedbackBusyError);
    expect((err as FeedbackBusyError).activities.map((a) => a.ref)).toEqual([
      "execute/foo",
      "research/bar",
    ]);
  });

  it("lets non-409 errors bubble up unchanged", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    await expect(service.start("i1", { type: "feedback", text: "hi" })).rejects.toThrow("boom");
  });

  it("decides a round and forwards accepted mutation IDs", async () => {
    const round = makeRound({ status: "applied" });
    vi.mocked(api.post).mockResolvedValue({
      round,
      apply_result: { outcomes: [], applied: 1, failed: 0, skipped: 0 },
    });
    await service.decide("i1", 1, {
      kind: "partial_accept",
      acceptedMutationIds: ["m1", "m3"],
      rationale: "skipping m2",
    });
    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/i1/feedback/1/decide",
      expect.objectContaining({
        kind: "partial_accept",
        accepted_mutation_ids: ["m1", "m3"],
        rationale: "skipping m2",
      }),
    );
  });

  it("continues a round as JSON by default", async () => {
    vi.mocked(api.post).mockResolvedValue(makeRound());
    await service.continue_("i1", 1, { text: "revise please" });
    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/i1/feedback/1/continue",
      expect.objectContaining({ text: "revise please" }),
    );
    const body = vi.mocked(api.post).mock.calls[0]![1];
    expect(body).not.toBeInstanceOf(FormData);
  });

  it("dismisses a round", async () => {
    vi.mocked(api.post).mockResolvedValue(makeRound({ status: "dismissed" }));
    await service.dismiss("i1", 1, { rationale: "not useful" });
    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/i1/feedback/1/dismiss",
      expect.objectContaining({ rationale: "not useful" }),
    );
  });

  it("cancels a stuck round", async () => {
    vi.mocked(api.post).mockResolvedValue(makeRound({ status: "dismissed" }));
    const result = await service.cancel("i1", 1, { rationale: "agent stuck" });
    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/i1/feedback/1/cancel",
      expect.objectContaining({ rationale: "agent stuck" }),
    );
    expect(result.status).toBe("dismissed");
  });

  it("deletes a terminal round", async () => {
    vi.mocked(api.delete).mockResolvedValue(undefined);
    await service.delete("i1", 3);
    expect(api.delete).toHaveBeenCalledWith("/initiatives/i1/feedback/3");
  });

  it("reports lock status", async () => {
    vi.mocked(api.get).mockResolvedValue({ locked: true, holder: { run_id: "r", purpose: "feedback" } });
    const status = await service.lockStatus("i1");
    expect(api.get).toHaveBeenCalledWith("/initiatives/i1/feedback/lock");
    expect(status.locked).toBe(true);
    expect(status.holder?.run_id).toBe("r");
  });

  it("surfaces item_activities on the lock status response", async () => {
    vi.mocked(api.get).mockResolvedValue({
      locked: false,
      item_activities: [
        { ref: "execute/foo", run_id: "run-foo", purpose: "execute" },
      ],
    });
    const status = await service.lockStatus("i1");
    expect(status.item_activities?.[0]?.ref).toBe("execute/foo");
  });

  it("builds attachment URLs using the canonical endpoint shape", () => {
    expect(service.attachmentUrl("i1", 3, "abc.png")).toBe(
      "/initiatives/i1/feedback/3/attachments/abc.png",
    );
  });
});
