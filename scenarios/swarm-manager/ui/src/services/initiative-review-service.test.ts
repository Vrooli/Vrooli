import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  createInitiativeReviewService,
  type IInitiativeReviewService,
} from "./initiative-review-service";
import type { IApiClient } from "../lib/api-client";

describe("Initiative Review Service", () => {
  let api: IApiClient;
  let service: IInitiativeReviewService;

  beforeEach(() => {
    api = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    service = createInitiativeReviewService(api);
  });

  it("lists rounds (defaulting to empty)", async () => {
    vi.mocked(api.get).mockResolvedValue({});
    expect(await service.listRounds("init-a")).toEqual([]);
    expect(api.get).toHaveBeenCalledWith("/initiatives/init-a/review");
  });

  it("fetches a round by number", async () => {
    vi.mocked(api.get).mockResolvedValue({
      round: 1,
      generated_at: "2026-04-23T00:00:00Z",
      status: "complete",
      evidence: [],
    });
    const r = await service.getRound("init-a", 1);
    expect(api.get).toHaveBeenCalledWith("/initiatives/init-a/review/1");
    expect(r.round).toBe(1);
  });

  it("triggers a manual review", async () => {
    vi.mocked(api.post).mockResolvedValue({ started: true, round: 1 });
    const r = await service.trigger("init-a");
    expect(api.post).toHaveBeenCalledWith("/initiatives/init-a/review/trigger", {});
    expect(r.started).toBe(true);
  });

  it("decides a review round and forwards rationale", async () => {
    vi.mocked(api.post).mockResolvedValue({
      initiative: "init-a",
      verdict: "accept",
      status: "completed",
      decided_at: "2026-04-23T00:00:00Z",
    });
    await service.decide("init-a", {
      verdict: "accept",
      rationale: "all items ship",
      decidedBy: "matt",
    });
    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/init-a/review/decide",
      expect.objectContaining({
        verdict: "accept",
        rationale: "all items ship",
        decided_by: "matt",
      }),
    );
  });

  it("lists decisions (empty-safe)", async () => {
    vi.mocked(api.get).mockResolvedValue({});
    expect(await service.listDecisions("init-a")).toEqual([]);
  });
});
