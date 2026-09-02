import { createClient } from "@connectrpc/connect";
import { MachineService, type GetMachineResponse as GeneratedGetMachineResponse, type Machine, type MachineTrust } from "@vrooli/proto-types/vrooli-bridge/v1/machines/machines_pb";

import { transport } from "./client";

// MachineService owns durable operator intent. Registry Nodes and Presence stay
// separate sources of paired identity and live state respectively.
export const machinesClient = createClient(MachineService, transport);
export interface MachineDrift {
  kind: string;
  name: string;
  reason: string;
}

export interface EffectivePolicy {
  profileId: string;
  profileVersion: string;
  setupPreset: string;
  scenarios: string[];
  optionalResources: string[];
}

// The generated package may be refreshed independently of this scenario's
// source tree. Optional fields keep old clients/mocks readable while the
// additive proto fields are available on refreshed clients.
export type GetMachineResponse = GeneratedGetMachineResponse & {
  drift?: MachineDrift[];
  effectivePolicy?: EffectivePolicy;
};
export type { Machine, MachineTrust };
