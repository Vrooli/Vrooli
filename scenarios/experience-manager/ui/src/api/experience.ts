import { createClient } from "@connectrpc/connect";
import { ContractService } from "@vrooli/proto-types/experience-manager/v1/contract/contract_pb";

import { transport } from "./client";

export const contractClient = createClient(ContractService, transport);

export async function fetchFleet() {
  return contractClient.listFleet({});
}
