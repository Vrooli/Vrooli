import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  AuthError,
  loginOwner,
  resolveAuthenticatorBaseUrl,
  validateOwner,
} from "./authenticator";

describe("resolveAuthenticatorBaseUrl", () => {
  // NOTE: the VITE_AUTH_* env precedence isn't unit-tested here — Vite snapshots
  // `import.meta.env` per module at transform time, so the resolver's env object
  // is not the one a test can mutate. The env path is trivial string iteration;
  // it's exercised in the live validation. The null + proxy branches (the logic
  // worth pinning) are covered below.
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns null when nothing configures the authenticator URL", () => {
    expect(resolveAuthenticatorBaseUrl()).toBeNull();
  });

  it("derives the app-monitor proxy path when served under /apps/", () => {
    vi.stubGlobal("location", {
      origin: "https://host.example",
      pathname: "/apps/device-sync-hub/proxy/",
    });
    expect(resolveAuthenticatorBaseUrl()).toBe(
      "https://host.example/apps/scenario-authenticator/proxy",
    );
  });
});

describe("loginOwner", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });
  afterEach(() => vi.unstubAllGlobals());

  it("returns the owner token on success", async () => {
    fetchSpy.mockResolvedValue(
      new Response(JSON.stringify({ success: true, token: "jwt-1", user: { email: "o@e.com", id: "u1" } }), {
        status: 200,
      }),
    );
    const identity = await loginOwner("http://auth.example", { email: "o@e.com", password: "pw" });
    expect(identity.token).toBe("jwt-1");
    expect(identity.email).toBe("o@e.com");
  });

  it("maps 401 to an invalid_credentials AuthError", async () => {
    fetchSpy.mockResolvedValue(new Response("{}", { status: 401 }));
    await expect(loginOwner("http://auth.example", { email: "o@e.com", password: "bad" })).rejects.toMatchObject({
      code: "invalid_credentials",
    });
  });

  it("treats success:false as invalid credentials", async () => {
    fetchSpy.mockResolvedValue(
      new Response(JSON.stringify({ success: false, message: "nope" }), { status: 200 }),
    );
    await expect(loginOwner("http://auth.example", { email: "o@e.com", password: "x" })).rejects.toBeInstanceOf(
      AuthError,
    );
  });

  it("maps a 5xx to an unavailable AuthError", async () => {
    fetchSpy.mockResolvedValue(new Response("boom", { status: 503 }));
    await expect(loginOwner("http://auth.example", { email: "o@e.com", password: "x" })).rejects.toMatchObject({
      code: "unavailable",
    });
  });
});

describe("validateOwner", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });
  afterEach(() => vi.unstubAllGlobals());

  it("decodes the validation response", async () => {
    fetchSpy.mockResolvedValue(
      new Response(JSON.stringify({ valid: true, user_id: "u1", email: "o@e.com", roles: ["user"] }), {
        status: 200,
      }),
    );
    const result = await validateOwner("http://auth.example", "jwt-1");
    expect(result.valid).toBe(true);
    expect(result.userId).toBe("u1");
    expect(result.email).toBe("o@e.com");
  });
});
