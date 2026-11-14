import { afterEach, expect, it, vi } from "vitest";

const capture = vi.hoisted(() => ({ fetch: undefined as typeof fetch | undefined }));
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => window.location.origin,
  createScenarioConnectTransport: (options: { fetch: typeof fetch }) => { capture.fetch = options.fetch; return {}; },
}));
vi.mock("../../lib/auth", () => ({ getAccessToken: async () => null }));
import "../client";

afterEach(() => { delete window.desktop; vi.unstubAllGlobals(); });

it("forwards the runtime session only to the same-origin API", async () => {
  const localToken = vi.fn(async () => "private-local-session");
  window.desktop = { auth: { getLocalSessionToken: localToken } } as unknown as NonNullable<Window["desktop"]>;
  const request = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response("{}"));
  vi.stubGlobal("fetch", request);
  await capture.fetch!(window.location.origin + "/vrooli.test.Service/Call", { headers: { Authorization: "Bearer account-token" } });
  expect(new Headers(request.mock.calls[0]?.[1]?.headers).get("X-Vrooli-Local-Session")).toBe("private-local-session");
  await capture.fetch!(new Request("https://external.example.invalid/rpc"), {});
  expect(new Headers(request.mock.calls[1]?.[1]?.headers).has("X-Vrooli-Local-Session")).toBe(false);
  expect(new Headers(request.mock.calls[0]?.[1]?.headers).get("Authorization")).toBe("Bearer account-token");
  expect(localToken).toHaveBeenCalledTimes(1);
});
