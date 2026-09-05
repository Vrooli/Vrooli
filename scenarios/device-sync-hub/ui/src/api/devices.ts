import { createClient } from "@connectrpc/connect";
import {
  DevicesService,
  TrustState,
  type Device,
  type DeviceProfile,
  type PairingCode,
} from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";

import { transport } from "./client";

/**
 * Typed client for the DevicesService. The owner-gated RPCs (list/get/issue/
 * approve/rename/revoke) require an owner JWT; the join RPCs (redeem/request)
 * are open. `authedFetch` attaches whichever credential is present, so callers
 * just invoke the method — the server enforces the gate.
 */
export const devicesClient = createClient(DevicesService, transport);

export { TrustState };
export type { Device, DeviceProfile, PairingCode };
