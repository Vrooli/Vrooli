import { createClient, type Client } from "@connectrpc/connect";
import { WizardService } from "@vrooli/proto-types/business-health/v1/wizard/wizard_pb";

import { transport } from "./client";

/**
 * Connect-RPC client for the business-health WizardService (contract-authoring
 * interview + deterministic scaffold). See `api/contract.ts` for the
 * client-usage convention.
 */
export const wizardClient: Client<typeof WizardService> = createClient(
  WizardService,
  transport,
);
