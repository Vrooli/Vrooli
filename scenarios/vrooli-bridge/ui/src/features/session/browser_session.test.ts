import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  generateBrowserKeyMaterial,
  loadBrowserEnrollment,
  mintBrowserSession,
  saveBrowserEnrollment,
  type BrowserEnrollment,
} from "./browser_session";

const enrollment: BrowserEnrollment = {
  operatorId: "operator-1",
  identityProvider: "scenario-authenticator",
  mode: "personal",
  reference: "enrollment-1",
  enrolledAt: "2026-08-17T00:00:00Z",
  scopeCeiling: ["vrooli-bridge:read", "vrooli-bridge:cleanup"],
  privateKeyPkcs8: "BAUG",
};

describe("browser enrollment session material", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("round-trips durable enrollment metadata and defaults omitted optional fields", () => {
    saveBrowserEnrollment(enrollment);
    expect(loadBrowserEnrollment()).toEqual(enrollment);

    window.localStorage.setItem("vrooli-bridge.operator-session", JSON.stringify({
      operatorId: "operator-2",
      identityProvider: "scenario-authenticator",
      mode: "personal",
      reference: "enrollment-2",
      privateKeyPkcs8: "BAUG",
    }));
    expect(loadBrowserEnrollment()).toMatchObject({ operatorId: "operator-2", enrolledAt: "", scopeCeiling: [] });
  });

  it("fails closed for missing, malformed, or incomplete enrollment records", () => {
    expect(loadBrowserEnrollment()).toBeNull();
    window.localStorage.setItem("vrooli-bridge.operator-session", "not-json");
    expect(loadBrowserEnrollment()).toBeNull();
    window.localStorage.setItem("vrooli-bridge.operator-session", JSON.stringify({ operatorId: "operator-1" }));
    expect(loadBrowserEnrollment()).toBeNull();
  });

  it("generates exportable public and private browser key material", async () => {
    const generateKey = vi.fn().mockResolvedValue({ publicKey: "public", privateKey: "private" });
    const exportKey = vi.fn()
      .mockResolvedValueOnce(new Uint8Array([1, 2, 3]).buffer)
      .mockResolvedValueOnce(new Uint8Array([4, 5, 6]).buffer);
    vi.stubGlobal("crypto", { subtle: { generateKey, exportKey } } as unknown as Crypto);

    const material = await generateBrowserKeyMaterial();

    expect(generateKey).toHaveBeenCalledWith({ name: "Ed25519" }, true, ["sign", "verify"]);
    expect(material.publicKey).toEqual(new Uint8Array([1, 2, 3]));
    expect(material.privateKeyPkcs8).toBe("BAUG");
  });

  it("mints a short-lived signed OS1 session with enrollment-scoped claims", async () => {
    const importKey = vi.fn().mockResolvedValue("private-key");
    const sign = vi.fn().mockResolvedValue(new Uint8Array([7, 8, 9]));
    vi.stubGlobal("crypto", { subtle: { importKey, sign } } as unknown as Crypto);

    const token = await mintBrowserSession(enrollment, 1_700_000_000_000);
    const [, encodedClaims] = token.split(".");
    if (!encodedClaims) throw new Error("session token did not contain claims");
    const paddedClaims = encodedClaims.replace(/-/gu, "+").replace(/_/gu, "/") + "===";
    const claims = JSON.parse(atob(paddedClaims.slice(0, paddedClaims.length - (paddedClaims.length % 4))));

    expect(token.startsWith("OS1.")).toBe(true);
    expect(claims).toEqual({
      enrollment_reference: "enrollment-1",
      operator_id: "operator-1",
      scopes: ["vrooli-bridge:read", "vrooli-bridge:cleanup"],
      iat: 1_700_000_000,
      exp: 1_700_000_900,
    });
    expect(importKey).toHaveBeenCalledWith("pkcs8", expect.any(ArrayBuffer), { name: "Ed25519" }, false, ["sign"]);
    expect(sign).toHaveBeenCalledWith({ name: "Ed25519" }, "private-key", expect.anything());
  });
});
