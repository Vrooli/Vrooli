import { createClient } from "@connectrpc/connect";
import { MachineService, type GetMachineResponse, type Machine, type MachineTrust } from "@vrooli/proto-types/vrooli-bridge/v1/machines/machines_pb";

import { transport } from "./client";

// MachineService owns durable operator intent. Registry Nodes and Presence stay
// separate sources of paired identity and live state respectively.
export const machinesClient = createClient(MachineService, transport);
export type { GetMachineResponse, Machine, MachineTrust };
