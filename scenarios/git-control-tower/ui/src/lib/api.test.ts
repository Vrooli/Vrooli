import { fetchHealth, fetchBranches, createBranch, fetchGroupingRules } from "./api";
import { jsonResponse, mockFetchJson, textResponse } from "../test-utils";

// [REQ:GCT-OT-P0-001] Health check endpoint

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:18700/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) => `${opts.baseUrl}${path}`
}));

test("fetchHealth returns parsed JSON on success", async () => {
  mockFetchJson({ status: "healthy", service: "x", timestamp: "t" });

  const result = await fetchHealth();
  expect(result.status).toBe("healthy");
});

test("fetchHealth throws when API returns non-200", async () => {
  globalThis.fetch = vi.fn(async () => {
    return textResponse("Service unavailable", { status: 503 });
  }) as unknown as typeof fetch;

  await expect(fetchHealth()).rejects.toThrow(/Service unavailable|Request failed: 503/);
});

test("fetchBranches returns parsed JSON on success", async () => {
  mockFetchJson({ current: "main", locals: [], remotes: [], timestamp: "t" });

  const result = await fetchBranches();
  expect(result.current).toBe("main");
});

test("createBranch returns parsed JSON on success", async () => {
  mockFetchJson({ success: true, branch: { name: "feature/test" }, timestamp: "t" });

  const result = await createBranch({ name: "feature/test" });
  expect(result.success).toBe(true);
  expect(result.branch?.name).toBe("feature/test");
});

test("fetchGroupingRules uses cache: no-store to bypass proxy caching", async () => {
  let capturedInit: RequestInit | undefined;
  globalThis.fetch = vi.fn(async (_url: string | URL | Request, init?: RequestInit) => {
    capturedInit = init;
    return jsonResponse({ enabled: true, rules: [] });
  }) as unknown as typeof fetch;

  await fetchGroupingRules("repo-1");
  expect(capturedInit?.cache).toBe("no-store");
});
