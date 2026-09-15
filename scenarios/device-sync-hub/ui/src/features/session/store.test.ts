import { afterEach, describe, expect, it } from "vitest";

import {
  clearSession,
  emptySession,
  loadSession,
  readSessionCredentials,
  saveSession,
} from "./store";
import { makeDevice } from "../../test-utils/session";

describe("session store", () => {
  afterEach(() => {
    clearSession();
  });

  it("round-trips pairing metadata without persisting the owner JWT", () => {
    const device = makeDevice({ id: "dev-9", name: "Laptop" });
    saveSession({ deviceToken: "dt", device, ownerToken: "ot", ownerEmail: "owner@example.com" });

    const loaded = loadSession();
    expect(loaded.deviceToken).toBe("dt");
    expect(loaded.ownerToken).toBeNull();
    expect(loaded.ownerEmail).toBe("owner@example.com");
    expect(loaded.device?.id).toBe("dev-9");
    expect(loaded.device?.name).toBe("Laptop");
  });

  it("returns the empty session when nothing is stored", () => {
    expect(loadSession()).toEqual(emptySession);
  });

  it("tolerates a corrupt payload by resolving to the empty session", () => {
    window.localStorage.setItem("device-sync-hub.session", "{not json");
    expect(loadSession()).toEqual(emptySession);
  });

  it("exposes just the credentials for the fetch wrapper", () => {
    saveSession({ deviceToken: "dt", device: null, ownerToken: null, ownerEmail: null });
    expect(readSessionCredentials()).toEqual({ deviceToken: "dt", ownerToken: null });
  });

  it("removes a legacy persisted owner JWT during migration", () => {
    window.localStorage.setItem(
      "device-sync-hub.session",
      JSON.stringify({ deviceToken: "dt", ownerToken: "legacy-jwt", ownerEmail: "owner@example.com" }),
    );

    expect(loadSession().ownerToken).toBeNull();
    expect(window.localStorage.getItem("device-sync-hub.session")).not.toContain("legacy-jwt");
  });
});
