/**
 * inventory.ts API client tests.
 *
 * The module wraps three plain-JSON endpoints (no proto codec on this
 * surface yet), so the tests stub `global.fetch` directly rather than
 * pulling in MSW for this scenario. We assert the URL+method the client
 * sends and the typed shape it returns, plus the ApiError envelope path
 * for non-2xx responses.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";
import { fetchFlows, fetchRuns, verifyFlow } from "./inventory";

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

const lastFetchCall = (fetchMock: ReturnType<typeof vi.fn>): FetchArgs => {
  const calls = fetchMock.mock.calls as FetchArgs[];
  expect(calls.length).toBeGreaterThan(0);
  return calls[calls.length - 1]!;
};

const urlOf = (args: FetchArgs): string => {
  const input = args[0];
  if (typeof input === "string") return input;
  if (input instanceof URL) return input.toString();
  return input.url;
};

describe("inventory API client", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe("fetchFlows", () => {
    it("issues GET to /api/v1/flows with the encoded root and returns the flows array", async () => {
      fetchMock.mockResolvedValue(
        jsonResponse({
          flows: [
            {
              flowId: "example.workflow.api",
              contractPath: "api/example/flow/flow.json",
              language: "go",
              schemaVersion: 6,
            },
          ],
        }),
      );

      const got = await fetchFlows("templates/scenarios/react-vite");

      expect(got).toHaveLength(1);
      expect(got[0]?.flowId).toBe("example.workflow.api");

      const args = lastFetchCall(fetchMock);
      expect(args[1]?.method).toBe("GET");
      expect(args[1]?.cache).toBe("no-store");
      const url = urlOf(args);
      expect(url).toContain("/api/v1/flows");
      expect(url).toContain("root=templates%2Fscenarios%2Freact-vite");
    });

    it("returns an empty array when the server omits flows", async () => {
      fetchMock.mockResolvedValue(jsonResponse({}));
      await expect(fetchFlows(".")).resolves.toEqual([]);
    });

    it("throws ApiError on a 5xx envelope so the UI surfaces a typed error", async () => {
      fetchMock.mockResolvedValue(errorEnvelope(500, "internal", "boom"));
      await expect(fetchFlows(".")).rejects.toBeInstanceOf(ApiError);
    });
  });

  describe("fetchRuns", () => {
    it("issues GET with no query string when no filters are supplied", async () => {
      fetchMock.mockResolvedValue(jsonResponse({ runs: [] }));
      await fetchRuns();
      const url = urlOf(lastFetchCall(fetchMock));
      expect(url).toContain("/api/v1/runs");
      expect(url).not.toContain("?");
    });

    it("encodes flowId and limit as query params", async () => {
      fetchMock.mockResolvedValue(jsonResponse({ runs: [] }));
      await fetchRuns({ flowId: "notes.attachment-upload.ui", limit: 50 });
      const url = urlOf(lastFetchCall(fetchMock));
      expect(url).toContain("flowId=notes.attachment-upload.ui");
      expect(url).toContain("limit=50");
    });

    it("returns an empty array when the server omits runs", async () => {
      fetchMock.mockResolvedValue(jsonResponse({}));
      await expect(fetchRuns()).resolves.toEqual([]);
    });

    it("throws ApiError on a 4xx envelope (e.g. bad limit)", async () => {
      fetchMock.mockResolvedValue(errorEnvelope(400, "invalid_request", "bad limit"));
      await expect(fetchRuns({ limit: -1 })).rejects.toBeInstanceOf(ApiError);
    });
  });

  describe("verifyFlow", () => {
    it("POSTs the verification body with mode=check and returns the typed response", async () => {
      fetchMock.mockResolvedValue(
        jsonResponse({ status: "passed", runs: [] }),
      );

      const got = await verifyFlow("/abs/root", "example.workflow.api");

      expect(got.status).toBe("passed");

      const args = lastFetchCall(fetchMock);
      expect(args[1]?.method).toBe("POST");
      expect(args[1]?.cache).toBe("no-store");
      expect(args[1]?.headers).toMatchObject({ "Content-Type": "application/json" });
      const body = typeof args[1]?.body === "string" ? args[1].body : "null";
      const sent = JSON.parse(body);
      expect(sent).toMatchObject({
        root: "/abs/root",
        flowId: "example.workflow.api",
        mode: "check",
      });
    });

    it("omits flowId when called without one (verify-all path)", async () => {
      fetchMock.mockResolvedValue(jsonResponse({ status: "passed", runs: [] }));
      await verifyFlow("/abs/root");
      const rawBody = lastFetchCall(fetchMock)[1]?.body;
      const sent = JSON.parse(typeof rawBody === "string" ? rawBody : "null");
      expect(sent.flowId).toBeUndefined();
    });

    it("throws ApiError on a 4xx envelope", async () => {
      fetchMock.mockResolvedValue(errorEnvelope(400, "invalid_request", "bad mode"));
      await expect(verifyFlow(".")).rejects.toBeInstanceOf(ApiError);
    });
  });
});
