import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@vrooli/api-base", () => {
  const resolveApiBase = vi.fn((_opts?: { appendSuffix?: boolean }) => "http://localhost:8080/api/v1");
  const buildApiUrl = vi.fn((path: string, options?: { baseUrl?: string }) => {
    const base = (options?.baseUrl ?? "http://localhost:8080/api/v1").replace(/\/+$/, "");
    if (!path) return base;
    const normalized = path.startsWith("/") ? path : `/${path}`;
    return `${base}${normalized}`;
  });
  return { resolveApiBase, buildApiUrl };
});

describe("buildUrl", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("prefixes a /sessions path with the resolved API base", async () => {
    const { buildUrl } = await import("./http");
    expect(buildUrl("/sessions")).toBe("http://localhost:8080/api/v1/sessions");
  });

  it("handles nested session paths", async () => {
    const { buildUrl } = await import("./http");
    expect(buildUrl("/sessions/abc-123/control")).toBe(
      "http://localhost:8080/api/v1/sessions/abc-123/control",
    );
  });

  it("normalizes paths that are missing a leading slash", async () => {
    const { buildUrl } = await import("./http");
    expect(buildUrl("sessions")).toBe("http://localhost:8080/api/v1/sessions");
  });

  it("does not duplicate the /api/v1 prefix when composing routes", async () => {
    const { buildUrl } = await import("./http");
    const built = buildUrl("/sessions");
    expect(built.includes("/api/v1/api/v1")).toBe(false);
  });
});

describe("throwIfNotOk", () => {
  it("does not throw for 2xx responses", async () => {
    const { throwIfNotOk } = await import("./http");
    const res = new Response("ok", { status: 200, statusText: "OK" });
    await expect(throwIfNotOk(res)).resolves.toBeUndefined();
  });

  it("throws with status, statusText, and body for non-2xx responses", async () => {
    const { throwIfNotOk } = await import("./http");
    const res = new Response("session not found", { status: 404, statusText: "Not Found" });
    await expect(throwIfNotOk(res)).rejects.toThrow(/404/);
    const res2 = new Response("session not found", { status: 404, statusText: "Not Found" });
    await expect(throwIfNotOk(res2)).rejects.toThrow(/session not found/);
  });

  it("throws even when the body is empty", async () => {
    const { throwIfNotOk } = await import("./http");
    const res = new Response(null, { status: 500, statusText: "Internal Server Error" });
    await expect(throwIfNotOk(res)).rejects.toThrow(/500/);
  });
});
