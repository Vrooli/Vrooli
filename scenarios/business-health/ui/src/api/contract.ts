import { createClient, type Client } from "@connectrpc/connect";
import { ContractService } from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

import { transport } from "./client";

/**
 * Connect-RPC client for the business-health ContractService. Feature hooks
 * call methods (`contractClient.getMatrix(...)`) directly through React Query
 * rather than wrapping them in ad-hoc fetch helpers — the generated client
 * owns proto encoding, error parsing, and cancellation.
 */
export const contractClient: Client<typeof ContractService> = createClient(
  ContractService,
  transport,
);
