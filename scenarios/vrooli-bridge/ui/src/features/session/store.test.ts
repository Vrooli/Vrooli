import { afterEach, describe, expect, it, vi } from "vitest";
import * as browserSession from "./browser_session";

import {
  clearSession,
  clearEnrollmentBootstrapToken,
  emptySession,
  loadSession,
  readOwnerToken,
  restoreLocalSession,
  saveSession,
  setEnrollmentBootstrapToken,
} from "./store";

describe("session store", () => {
  afterEach(() => {
    clearSession();
    clearEnrollmentBootstrapToken();
    vi.clearAllMocks();
  });

  it("round-trips an owner session through localStorage", () => {
    saveSession({ ownerToken: "ot", ownerEmail: "owner@example.com" });

    const loaded = loadSession();
    expect(loaded.ownerToken).toBe("ot");
    expect(loaded.ownerEmail).toBe("owner@example.com");
  });

  it("returns the empty session when nothing is stored", () => {
    expect(loadSession()).toEqual(emptySession);
  });

  it("tolerates a corrupt payload by resolving to the empty session", () => {
    window.localStorage.setItem("vrooli-bridge.session", "{not json");
    expect(loadSession()).toEqual(emptySession);
  });

  it("exposes just the owner token for the fetch wrapper", () => {
    saveSession({ ownerToken: "ot", ownerEmail: null });
    expect(readOwnerToken()).toBe("ot");
  });

  it("reads null when no owner token is stored", () => {
    expect(readOwnerToken()).toBeNull();
  });

  it("prioritizes and then clears the one-RPC bootstrap token", () => {
    saveSession({ ownerToken: "session-token", ownerEmail: null });
    setEnrollmentBootstrapToken("bootstrap-token");
    expect(readOwnerToken()).toBe("bootstrap-token");

    clearEnrollmentBootstrapToken();
    expect(readOwnerToken()).toBe("session-token");
  });

  it("does not discard an in-memory token when durable enrollment exists", () => {
    saveSession({ ownerToken: "session-token", ownerEmail: null });
    window.localStorage.setItem("vrooli-bridge.operator-session", JSON.stringify({ enrolled: true }));

    expect(loadSession().ownerToken).toBe("session-token");
  });

  it("restores a local session from enrollment and handles mint failures", async () => {
    const loadEnrollment = vi.spyOn(browserSession, "loadBrowserEnrollment").mockReturnValue({
      operatorId: "operator-1",
      identityProvider: "test",
      mode: "test",
      reference: "enrollment-1",
      enrolledAt: "",
      scopeCeiling: [],
      privateKeyPkcs8: "private-key",
    });
    const mintSession = vi.spyOn(browserSession, "mintBrowserSession").mockResolvedValue("OS1.restored");
    window.localStorage.setItem("vrooli-bridge.operator-session", "enrolled");
    const restored = await restoreLocalSession();
    expect(loadEnrollment).toHaveBeenCalledTimes(1);
    expect(mintSession).toHaveBeenCalledTimes(1);
    expect(restored).toMatchObject({ ownerToken: "OS1.restored" });

    clearSession();
    mintSession.mockRejectedValue(new Error("unsupported browser"));
    await expect(restoreLocalSession()).resolves.toEqual(emptySession);
  });

  it("returns null when no enrollment is available or a session is already live", async () => {
    vi.spyOn(browserSession, "loadBrowserEnrollment").mockReturnValue(null);
    await expect(restoreLocalSession()).resolves.toBeNull();

    saveSession({ ownerToken: "already-live", ownerEmail: null });
    await expect(restoreLocalSession()).resolves.toBeNull();
  });
});
