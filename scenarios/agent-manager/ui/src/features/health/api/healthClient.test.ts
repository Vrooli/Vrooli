import { afterEach, expect, test, vi } from "vitest";
import { fetchHealthAudit, fetchModelHealth, fetchModelPolicyDrift, fetchRunnerHealth } from "./healthClient.js";

afterEach(() => vi.unstubAllGlobals());

function stubFetch(body: unknown = {}) {
  const fetch = vi.fn(async () => new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetch);
  return fetch;
}
const urls = (fetch: ReturnType<typeof stubFetch>) => fetch.mock.calls.map(([url]) => String(url));

test("each health read goes to its own endpoint", async () => {
  const fetch = stubFetch({ models: [], runners: [] });
  await fetchModelHealth();
  await fetchRunnerHealth();
  await fetchModelPolicyDrift();
  expect(urls(fetch).map((url) => url.split("/api/v1")[1])).toEqual(["/health/models", "/health/runners", "/health/model-policy-drift"]);
});

test("the audit read sends only the filters the operator set", async () => {
  const fetch = stubFetch({ entries: [] });
  await fetchHealthAudit({ scope: "model", runner: "codex", model: "gpt-5", status: "failed", since: "2026-01-01", until: "2026-02-01", limit: 50 });
  await fetchHealthAudit({ scope: "runner" });
  const [full, bare] = urls(fetch).map((url) => new URL(url).searchParams);
  expect(Object.fromEntries(full)).toEqual({ scope: "model", runner: "codex", model: "gpt-5", status: "failed", since: "2026-01-01", until: "2026-02-01", limit: "50" });
  expect(Object.fromEntries(bare)).toEqual({ scope: "runner" });
});

test("a failed read surfaces the server's message", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ message: "health store unavailable" }), { status: 503 })));
  await expect(fetchModelHealth()).rejects.toThrow("health store unavailable");
});
