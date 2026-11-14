import { createClient, type Client } from "@connectrpc/connect";
import { ScenarioValidationService } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

/**
 * Connect-RPC client for the shared ScenarioValidationService that
 * business-health serves (PreviewFix / ApplyFix deterministic remediation).
 * See `api/contract.ts` for the client-usage convention. An Unimplemented
 * response from PreviewFix means the provider ships no fixer for that scenario.
 */
export const validationClient: Client<typeof ScenarioValidationService> = createClient(
  ScenarioValidationService,
  transport,
);
