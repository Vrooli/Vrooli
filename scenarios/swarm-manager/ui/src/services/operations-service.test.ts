import { describe, it, expect, vi } from "vitest";
import {
  createOperationsService,
  formatWindowSeconds,
  __test__,
} from "./operations-service";
import type { IApiClient } from "../lib/api-client";

function fakeClient(handler: (path: string) => unknown): IApiClient {
  return {
    async get<T>(path: string): Promise<T> {
      return handler(path) as T;
    },
    async post<T>(): Promise<T> {
      throw new Error("not implemented");
    },
    async put<T>(): Promise<T> {
      throw new Error("not implemented");
    },
    async patch<T>(): Promise<T> {
      throw new Error("not implemented");
    },
    async delete<T>(): Promise<T> {
      throw new Error("not implemented");
    },
  };
}

describe("formatWindowSeconds", () => {
  it("formats whole hours", () => {
    expect(formatWindowSeconds(3 * 3600)).toBe("PT3H");
  });

  it("formats hours plus minutes", () => {
    expect(formatWindowSeconds(3 * 3600 + 30 * 60)).toBe("PT3H30M");
  });

  it("formats minutes only", () => {
    expect(formatWindowSeconds(45 * 60)).toBe("PT45M");
  });

  it("formats seconds only when below a minute", () => {
    expect(formatWindowSeconds(30)).toBe("PT30S");
  });

  it("falls back to PT3H for non-positive input", () => {
    expect(formatWindowSeconds(0)).toBe("PT3H");
    expect(formatWindowSeconds(-10)).toBe("PT3H");
    expect(formatWindowSeconds(NaN)).toBe("PT3H");
  });
});

describe("buildOperationsQuery", () => {
  const { buildOperationsQuery } = __test__;

  it("returns empty string when no filters supplied", () => {
    expect(buildOperationsQuery()).toBe("");
    expect(buildOperationsQuery({})).toBe("");
  });

  it("adds the window when set", () => {
    expect(buildOperationsQuery({ windowSeconds: 3600 })).toBe("?window=PT1H");
  });

  it("uses repeated query keys for array filters", () => {
    const q = buildOperationsQuery({
      lanes: ["execute", "investigate"],
      statuses: ["running"],
    });
    // URLSearchParams encodes nothing here so we can match literally.
    expect(q).toContain("lane=execute");
    expect(q).toContain("lane=investigate");
    expect(q).toContain("status=running");
  });

  it("drops empty array entries", () => {
    expect(buildOperationsQuery({ lanes: ["", "   "] })).toBe("");
  });

  it("trims and includes the q search string", () => {
    expect(buildOperationsQuery({ q: "  auth  " })).toBe("?q=auth");
  });

  it("renames ownerTypes to owner_type", () => {
    expect(buildOperationsQuery({ ownerTypes: ["initiative"] })).toBe(
      "?owner_type=initiative",
    );
  });
});

describe("createOperationsService.fetchOperations", () => {
  const sampleResponse = {
    lanes: [
      { lane: "investigate", active: 2, capacity: 6, queue: 0 },
      { lane: "execute", active: 1, capacity: 3, queue: 2 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 2, max_depth: 50 },
    activities: [
      {
        activity_id: "a-1",
        run_id: "run-1",
        owner_type: "initiative",
        owner_name: "auth-rewrite",
        owner_title: "Auth Rewrite",
        purpose: "process",
        phase_kind: "execute",
        lane: "execute",
        status: "running",
        mode: "holistic-loop",
        phase: "execute",
        round: 4,
        initiative_name: "auth-rewrite",
        requested_at: "2026-05-02T01:00:00Z",
        started_at: "2026-05-02T01:00:01Z",
        runtime_seconds: 180,
      },
    ],
    recently_finished: [
      {
        activity_id: "a-2",
        run_id: "run-2",
        owner_type: "backlog",
        owner_kind: "fix",
        owner_name: "fix-login-flicker",
        owner_title: "Fix login flicker",
        purpose: "process",
        phase_kind: "execute",
        lane: "execute",
        status: "complete",
        requested_at: "2026-05-02T00:30:00Z",
        finished_at: "2026-05-02T00:35:00Z",
        runtime_seconds: 300,
      },
    ],
    generated_at: "2026-05-02T01:05:00Z",
    window_seconds: 10800,
  };

  it("normalizes the snake_case wire shape", async () => {
    const client = fakeClient(() => sampleResponse);
    const svc = createOperationsService(client);
    const view = await svc.fetchOperations();

    expect(view.lanes).toHaveLength(4);
    expect(view.lanes[0]).toEqual({ lane: "investigate", active: 2, capacity: 6, queue: 0 });
    expect(view.queue).toEqual({ depth: 2, maxDepth: 50 });
    expect(view.activities[0]).toMatchObject({
      activityId: "a-1",
      runId: "run-1",
      ownerType: "initiative",
      ownerTitle: "Auth Rewrite",
      phaseKind: "execute",
      lane: "execute",
      mode: "holistic-loop",
      round: 4,
      initiativeName: "auth-rewrite",
      runtimeSeconds: 180,
    });
    expect(view.recentlyFinished[0]?.finishedAt).toBe("2026-05-02T00:35:00Z");
    expect(view.windowSeconds).toBe(10800);
    expect(view.generatedAt).toBe("2026-05-02T01:05:00Z");
  });

  it("issues the request to /operations with no query when no filters", async () => {
    const seen: string[] = [];
    const client = fakeClient((path) => {
      seen.push(path);
      return sampleResponse;
    });
    await createOperationsService(client).fetchOperations();
    expect(seen).toEqual(["/operations"]);
  });

  it("appends repeated lane keys when filters carry an array", async () => {
    const seen: string[] = [];
    const client = fakeClient((path) => {
      seen.push(path);
      return sampleResponse;
    });
    await createOperationsService(client).fetchOperations({
      windowSeconds: 3600,
      lanes: ["execute", "review"],
      statuses: ["running"],
      q: "auth",
    });
    expect(seen).toHaveLength(1);
    const url = seen[0]!;
    expect(url.startsWith("/operations?")).toBe(true);
    expect(url).toContain("window=PT1H");
    expect(url).toContain("lane=execute");
    expect(url).toContain("lane=review");
    expect(url).toContain("status=running");
    expect(url).toContain("q=auth");
  });

  it("returns a defensive empty view when the body is malformed", async () => {
    const client = fakeClient(() => "not an object");
    const view = await createOperationsService(client).fetchOperations();
    expect(view.lanes).toEqual([]);
    expect(view.activities).toEqual([]);
    expect(view.recentlyFinished).toEqual([]);
    expect(view.queue).toEqual({ depth: 0, maxDepth: 0 });
  });

  it("propagates client errors", async () => {
    const client = fakeClient(() => {
      throw new Error("boom");
    });
    const fn = vi.fn(() => createOperationsService(client).fetchOperations());
    await expect(fn()).rejects.toThrow("boom");
  });
});
