import assert from "node:assert/strict";
import { afterEach, test, vi } from "vitest";
import { fetchHealth, validatePath } from "../../src/lib/api.js";

afterEach(() => vi.unstubAllGlobals());

test("transport health check requests the canonical endpoint without cached operator state", async () => {
  const fetch = vi.fn(async () => new Response(JSON.stringify({ status: "ready", service: "agent-manager", timestamp: "now" }), { status: 200 }));
  vi.stubGlobal("fetch", fetch);

  assert.deepEqual(await fetchHealth(), { status: "ready", service: "agent-manager", timestamp: "now" });
  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/health");
  assert.equal(fetch.mock.calls[0]?.[1]?.cache, "no-store");
});

test("transport path validation encodes optional project roots and preserves endpoint failures", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ path: "/repo/a b", projectRoot: "/repo root", valid: true, exists: true, isDirectory: false, withinProjectRoot: true }), { status: 200 }))
    .mockResolvedValueOnce(new Response(null, { status: 503 }));
  vi.stubGlobal("fetch", fetch);

  const result = await validatePath("/repo/a b", "/repo root");
  assert.equal(result.valid, true);
  assert.match(String(fetch.mock.calls[0]?.[0]), /path=%2Frepo%2Fa\+b/);
  assert.match(String(fetch.mock.calls[0]?.[0]), /projectRoot=%2Frepo\+root/);
  await assert.rejects(validatePath("/missing"), /Path validation failed: 503/);
});
