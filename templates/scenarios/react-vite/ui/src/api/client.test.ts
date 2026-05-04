import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ResponseSchema } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";

import { ApiError, protoFetch } from "./client";

describe("api/client.protoFetch", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("decodes proto-name snake_case response fields into generated lowerCamel properties", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          status: "healthy",
          service: "react-vite-test",
          timestamp: "2026-01-01T00:00:00Z",
          readiness: true,
          uptime_seconds: 12.5,
          dependencies: {
            database: { connected: true, latency_ms: 3 },
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    const got = await protoFetch("GET", "/health", { responseSchema: ResponseSchema });

    expect(got.uptimeSeconds).toBe(12.5);
    expect(got.dependencies.database?.latencyMs).toBe(3);
  });

  it("also decodes lowerCamel response fields accepted by protobuf JSON parsers", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          status: "healthy",
          service: "react-vite-test",
          timestamp: "2026-01-01T00:00:00Z",
          readiness: true,
          uptimeSeconds: 9,
          dependencies: {
            database: { connected: true, latencyMs: 2 },
          },
        }),
        { status: 200 },
      ),
    );

    const got = await protoFetch("GET", "/health", { responseSchema: ResponseSchema });

    expect(got.uptimeSeconds).toBe(9);
    expect(got.dependencies.database?.latencyMs).toBe(2);
  });

  it("serializes request bodies with proto field names", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          status: "healthy",
          service: "react-vite-test",
          timestamp: "2026-01-01T00:00:00Z",
          readiness: true,
        }),
        { status: 200 },
      ),
    );

    await protoFetch("POST", "/echo-health", {
      requestSchema: ResponseSchema,
      request: {
        status: "healthy",
        service: "react-vite-test",
        timestamp: "2026-01-01T00:00:00Z",
        readiness: true,
        uptimeSeconds: 44,
        dependencies: {
          database: { connected: true, latencyMs: 7 },
        },
      },
      responseSchema: ResponseSchema,
    });

    const [, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toMatchObject({
      uptime_seconds: 44,
      dependencies: {
        database: { connected: true, latency_ms: 7 },
      },
    });
    const body = String(init.body);
    expect(body).not.toContain("uptimeSeconds");
    expect(body).not.toContain("latencyMs");
  });

  it("throws ApiError with the typed envelope on non-2xx responses", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "internal", message: "store down" }), {
        status: 500,
      }),
    );

    const err = await protoFetch("GET", "/health", { responseSchema: ResponseSchema }).catch(
      (e: unknown) => e,
    );

    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe("internal");
    expect((err as ApiError).status).toBe(500);
    expect((err as ApiError).message).toContain("store down");
  });
});
