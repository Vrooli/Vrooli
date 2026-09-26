import { createClient } from "@connectrpc/connect";
import {
  ApplyService,
  type ApplyAction,
  type ApplyResponse,
  type SkipReason,
} from "@vrooli/proto-types/brand-manager/v1/apply/apply_pb";

import { transport } from "./client";

export const applyClient = createClient(ApplyService, transport);

/**
 * previewApply reports which files applying a brand to a scenario WOULD write,
 * without touching the filesystem. The mutating ApplyBrand RPC is a CLI/wizard
 * action (it writes into another scenario's source tree), so the UI surfaces
 * only the safe preview.
 */
export async function previewApply(input: {
  brandId: string;
  scenarioName: string;
  elements?: string[];
}): Promise<ApplyResponse> {
  return applyClient.previewApply({
    brandId: input.brandId,
    scenarioName: input.scenarioName,
    elements: input.elements ?? [],
  });
}

export type { ApplyAction, ApplyResponse, SkipReason };
