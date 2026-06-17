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

  it("round-trips a paired session through localStorage (device as proto-JSON)", () => {
    const device = makeDevice({ id: "dev-9", name: "Laptop" });
    saveSession({ deviceToken: "dt", device, ownerToken: "ot" });

    const loaded = loadSession();
    expect(loaded.deviceToken).toBe("dt");
    expect(loaded.ownerToken).toBe("ot");
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
    saveSession({ deviceToken: "dt", device: null, ownerToken: null });
    expect(readSessionCredentials()).toEqual({ deviceToken: "dt", ownerToken: null });
  });
});
