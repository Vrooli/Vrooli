import type { Controller } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";

const stableIdentitySources = new Set(["openrgb_sdk_serial", "openrgb_linux_pci_i2c_location"]);

export function hasStableControllerIdentity(controller: Pick<Controller, "identitySource">): boolean {
  return stableIdentitySources.has(controller.identitySource);
}
