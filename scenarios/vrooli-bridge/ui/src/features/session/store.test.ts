import { afterEach, describe, expect, it } from "vitest";

import {
  clearSession,
  emptySession,
  loadSession,
  readOwnerToken,
  saveSession,
} from "./store";

describe("session store", () => {
  afterEach(() => {
    clearSession();
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
});
