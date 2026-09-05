import { createClient } from "@connectrpc/connect";
import {
  GenerationService,
  type GetImageBackendStatusResponse,
  type GetProviderStatusResponse,
  type ImageOperationStatus,
  type ProviderStatus,
} from "@vrooli/proto-types/brand-manager/v1/generation/generation_pb";

import { transport } from "./client";

export const generationClient = createClient(GenerationService, transport);

/** getProviderStatus reports whether any text AI provider in the chain is reachable. */
export async function getProviderStatus(): Promise<GetProviderStatusResponse> {
  return generationClient.getProviderStatus({});
}

/**
 * getImageBackendStatus reports image-tools' readiness for the brand image
 * operations (generate / edit / remove_background): which model/tier it would
 * use, and an actionable hint when an operation is not ready.
 */
export async function getImageBackendStatus(): Promise<GetImageBackendStatusResponse> {
  return generationClient.getImageBackendStatus({});
}

export type {
  GetImageBackendStatusResponse,
  GetProviderStatusResponse,
  ImageOperationStatus,
  ProviderStatus,
};
