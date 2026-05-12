/**
 * scenarios.ts API client tests. Mirrors inventory.test.ts: stubs
 * `global.fetch` directly, asserts the URL the client emits and the
 * typed shape it returns, including the error-envelope path.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";
import { fetchScenarioDetail, fetchScenarios } from "./scenarios";

type FetchArgs = [input: RequestInfo | URL, init?: RequestInit];

const jsonResponse = (body: unknown, init: ResponseInit = {}) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });

const errorEnvelope = (status: number, code: string, message: string) =>
  new Response(JSON.stringify({ code, message }), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const lastUrl = (fetchMock: ReturnType<typeof vi.fn>): string => {
  const calls = fetchMock.mock.calls as FetchArgs[];
  expect(calls.length).toBeGreaterThan(0);
  const input = calls[calls.length - 1]![0];
  if (typeof input === "string") return input;
  if (input instanceof URL) return input.toString();
  return input.url;
};

describe("scenarios API client", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe("fetchScenarios", () => {
    it("GETs /api/v1/scenarios and returns root + array", async () => {
      fetchMock.mockResolvedValue(
        jsonResponse({
          vrooliRoot: "/repo",
          scenarios: [
            { id: "alpha", displayName: "Alpha", path: "/repo/scenarios/alpha", flowCount: 2 },
          ],
        }),
      );
      const got = await fetchScenarios();
      expect(lastUrl(fetchMock)).toMatch(/\/api\/v1\/scenarios$/);
      expect(got.vrooliRoot).toBe("/repo");
      expect(got.scenarios).toHaveLength(1);
      expect(got.scenarios[0]).toMatchObject({ id: "alpha", flowCount: 2 });
    });

    it("normalises a missing scenarios field to an empty array", async () => {
      fetchMock.mockResolvedValue(jsonResponse({ vrooliRoot: "/repo" }));
      const got = await fetchScenarios();
      expect(got.scenarios).toEqual([]);
    });

    it("throws ApiError on non-2xx", async () => {
      fetchMock.mockResolvedValue(errorEnvelope(500, "internal", "boom"));
      await expect(fetchScenarios()).rejects.toBeInstanceOf(ApiError);
    });
  });

  describe("fetchScenarioDetail", () => {
    it("GETs /api/v1/scenarios/:id and returns the detail", async () => {
      fetchMock.mockResolvedValue(
        jsonResponse({
          id: "alpha",
          displayName: "Alpha",
          path: "/repo/scenarios/alpha",
          flowCount: 1,
          flows: [
            { flowId: "alpha.workflow", contractPath: "alpha/flow/flow.json", language: "go", schemaVersion: 6 },
          ],
        }),
      );
      const got = await fetchScenarioDetail("alpha");
      expect(lastUrl(fetchMock)).toMatch(/\/api\/v1\/scenarios\/alpha$/);
      expect(got.id).toBe("alpha");
      expect(got.flows).toHaveLength(1);
    });

    it("url-encodes the id segment", async () => {
      fetchMock.mockResolvedValue(jsonResponse({ id: "weird id", flows: [] }));
      await fetchScenarioDetail("weird id");
      expect(lastUrl(fetchMock)).toContain("weird%20id");
    });

    it("normalises a missing flows field to an empty array", async () => {
      fetchMock.mockResolvedValue(jsonResponse({ id: "alpha", displayName: "Alpha" }));
      const got = await fetchScenarioDetail("alpha");
      expect(got.flows).toEqual([]);
    });
  });
});
