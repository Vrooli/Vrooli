import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  DeviceSchema,
  TrustState,
  type Device,
} from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";

import { saveSession } from "../features/session/store";

/** Build a Device for tests with sane defaults + overrides. */
export function makeDevice(overrides: MessageInitShape<typeof DeviceSchema> = {}): Device {
  return create(DeviceSchema, {
    id: "dev-1",
    ownerId: "owner-1",
    name: "Test Device",
    kind: "browser",
    platform: "web",
    trustState: TrustState.TRUSTED,
    online: true,
    ...overrides,
  });
}

/**
 * Seed a paired (and optionally owner) session into localStorage BEFORE a
 * render so `SessionProvider` initialises from it. Mirrors what a successful
 * pairing / owner sign-in would persist.
 */
export function seedSession(opts: {
  deviceToken?: string;
  device?: Device | null;
  ownerToken?: string;
} = {}): void {
  saveSession({
    deviceToken: opts.deviceToken ?? "device-token-test",
    device: opts.device ?? makeDevice(),
    ownerToken: opts.ownerToken ?? null,
  });
}
