import assert from "node:assert/strict";
import { afterEach, test, vi } from "vitest";
import { validatePath } from "./api.js";

afterEach(() => vi.unstubAllGlobals());

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
