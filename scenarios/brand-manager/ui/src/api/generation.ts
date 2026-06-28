import { createClient } from "@connectrpc/connect";
import {
  GenerationService,
  type GetProviderStatusResponse,
  type ProviderStatus,
} from "@vrooli/proto-types/brand-manager/v1/generation/generation_pb";

import { transport } from "./client";

export const generationClient = createClient(GenerationService, transport);

/** getProviderStatus reports whether any AI provider in the chain is reachable. */
export async function getProviderStatus(): Promise<GetProviderStatusResponse> {
  return generationClient.getProviderStatus({});
}

export type { GetProviderStatusResponse, ProviderStatus };
