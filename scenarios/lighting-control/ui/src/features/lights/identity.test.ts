import { describe, expect, it } from "vitest";

import { hasStableControllerIdentity } from "./identity";

describe("controller identity eligibility", () => {
  it("allows current SDK serial and normalized PCI topology identities only", () => {
    expect(hasStableControllerIdentity({ identitySource: "openrgb_sdk_serial" })).toBe(true);
    expect(hasStableControllerIdentity({ identitySource: "openrgb_linux_pci_i2c_location" })).toBe(true);
    expect(hasStableControllerIdentity({ identitySource: "openrgb_serial_number" })).toBe(false);
    expect(hasStableControllerIdentity({ identitySource: "openrgb_device_location" })).toBe(false);
    expect(hasStableControllerIdentity({ identitySource: "openrgb_sdk_transient_handle" })).toBe(false);
    expect(hasStableControllerIdentity({ identitySource: "openrgb_identity_ambiguous" })).toBe(false);
    expect(hasStableControllerIdentity({ identitySource: "" })).toBe(false);
  });
});
