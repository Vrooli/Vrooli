import { afterEach, describe, expect, it, vi } from "vitest";
import { sessionApi } from "./session";
afterEach(() => vi.unstubAllGlobals());
describe("same-origin session API", () => {
  it("uses cookies and an explicit anti-CSRF header for mutations", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ authenticated: false })));
    vi.stubGlobal("fetch", fetch);
    await sessionApi.login({ email: "owner@example.test", password: "password" });
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining("/api/session/login"), expect.objectContaining({ method: "POST", credentials: "same-origin", cache: "no-store", headers: { "X-Vrooli-Session": "1", "Content-Type": "application/json" } }));
  });
  it("shows the provider-safe failure instead of claiming success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ message: "Sign-in or session verification failed." }), { status: 401 })));
    await expect(sessionApi.refresh()).rejects.toThrow("verification failed");
  });
});
