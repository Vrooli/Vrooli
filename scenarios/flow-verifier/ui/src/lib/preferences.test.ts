import { describe, expect, it } from "vitest";

import { DEFAULT_SETTINGS, readCache, writeCache } from "./preferences";

describe("preferences cache layer", () => {
  it("read/write cache round trip", () => {
    window.localStorage.clear();
    writeCache({ ...DEFAULT_SETTINGS, theme: "dark" });
    expect(readCache()?.theme).toBe("dark");
  });

  it("readCache merges partial cache with defaults", () => {
    window.localStorage.clear();
    window.localStorage.setItem(
      "flow-verifier.settings.cache.v1",
      JSON.stringify({ theme: "dark" }),
    );
    const c = readCache();
    expect(c?.theme).toBe("dark");
    expect(c?.density).toBe("comfortable");
  });

  // fetchSettings + putSettings are now thin wrappers over the
  // generated SettingsService Connect client. Their request/response
  // shape is enforced by the proto-generated types, so unit-test
  // coverage lives on the API side (handlers/settings/connect_handler_test.go)
  // and the contract is asserted via the wire-drift compile-time test.
});
