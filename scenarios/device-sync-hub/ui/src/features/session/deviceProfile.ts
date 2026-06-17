import type { MessageInitShape } from "@bufbuild/protobuf";
import { DeviceProfileSchema } from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";

/**
 * Build the self-description a joining browser supplies during pairing. The
 * name is owner-typed; kind/platform are coarse advisory labels the hub stores
 * as-is. We keep this dependency-free and avoid the deprecated
 * `navigator.platform` — "web" is a truthful platform for a browser client.
 */
export function browserDeviceProfile(deviceName: string): MessageInitShape<typeof DeviceProfileSchema> {
  return {
    deviceName,
    kind: "browser",
    platform: "web",
    capabilities: ["clipboard"],
  };
}
