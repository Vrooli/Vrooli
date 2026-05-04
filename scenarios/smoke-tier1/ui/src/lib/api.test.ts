/**
 * Unit tests for the lib/api seam.
 *
 * `fetchHealth` is a thin wrapper around `fetch` that decodes responses
 * into the generated proto type via `fromJson(ResponseSchema, ...)`. We
 * stub the global `fetch` rather than reaching for MSW — a 4-line
 * wrapper doesn't justify the heavy dep, and `vi.stubGlobal` +
 * `vi.unstubAllGlobals` give the same isolation per test without the
 * install footprint.
 *
 * What these tests pin:
 *   - the happy path returns a typed `HealthResponse` decoded from
 *     the JSON body via the proto schema
 *   - non-2xx responses throw with the status code in the message (so
 *     React Query surfaces it on the screen via `error.message`)
 *   - the request hits a URL ending in `/health` with `cache: "no-store"`
 *     and the JSON content-type header — the browser-cache gotcha that
 *     bit two scenarios pre-template-rebuild
 *   - unknown fields on the wire don't break decoding (the
 *     `ignoreUnknownFields: true` contract)
 *
 * If a future scenario inlines a second endpoint into `lib/api.ts`,
 * extend this file with a parallel describe — one block per endpoint,
 * same stub-fetch pattern, no helper extraction until the third caller.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchHealth } from "./api";

describe("lib/api.fetchHealth", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns a typed HealthResponse decoded via the proto schema on a 200 response", async () => {
    // Wire shape mirrors what api-core/health.Response produces. snake_case
    // fields prove the proto JSON parser accepts the canonical wire form.
    const payload = {
      status: "healthy",
      service: "react-vite-test",
      timestamp: "2026-01-01T00:00:00Z",
      readiness: true,
      version: "1.0.0",
      dependencies: {
        database: { connected: true, database: "sqlite" },
      },
    };
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify(payload), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const got = await fetchHealth();

    expect(got.status).toBe("healthy");
    expect(got.service).toBe("react-vite-test");
    expect(got.timestamp).toBe("2026-01-01T00:00:00Z");
    expect(got.readiness).toBe(true);
    expect(got.dependencies.database?.connected).toBe(true);
    expect(got.dependencies.database?.database).toBe("sqlite");
  });

  it("throws an Error containing the status code on a non-2xx response", async () => {
    fetchSpy.mockResolvedValueOnce(new Response("", { status: 503 }));

    await expect(fetchHealth()).rejects.toThrow(/503/);
  });

  it("requests /health with cache: 'no-store' and JSON content-type", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response('{"status":"healthy","service":"x","timestamp":"t","readiness":true}', { status: 200 }),
    );

    await fetchHealth();

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/health$/);
    expect(init).toMatchObject({
      cache: "no-store",
      headers: { "Content-Type": "application/json" },
    });
  });

  it("tolerates unknown fields on the wire (ignoreUnknownFields contract)", async () => {
    const payloadWithUnknowns = {
      status: "healthy",
      service: "x",
      timestamp: "t",
      readiness: true,
      // Future api-core/health additions land here without breaking
      // decode. If this assertion ever fails, ignoreUnknownFields was
      // unwired and every existing test would break the moment the wire
      // grows a field — which is exactly the failure mode this contract
      // exists to prevent.
      future_field_added_after_this_test: 42,
      another_unknown: { nested: true },
    };
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify(payloadWithUnknowns), { status: 200 }),
    );

    const got = await fetchHealth();
    expect(got.status).toBe("healthy");
  });
});
