import {
  type ListQuarantinedOutputsResponse,
  type QuarantinedOutput,
} from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { restApiUrl } from "./client";
import { safetyClient } from "./safety";

export type { ListQuarantinedOutputsResponse, QuarantinedOutput };

/** List the refused outputs currently retained for admin review. */
export async function listQuarantinedOutputs(limit = 100, offset = 0): Promise<ListQuarantinedOutputsResponse> {
  return safetyClient.listQuarantinedOutputs({ limit, offset });
}

/**
 * URL for one quarantined image. The ordinary blob route refuses the quarantine
 * namespace, so this dedicated route is the only way the review surface can
 * request the bytes. Nothing is fetched until the operator reveals an entry.
 */
export function quarantineImageUrl(id: bigint | number, download = false): string {
  const suffix = download ? "?download=1" : "";
  return restApiUrl(`/safety/quarantine/${id.toString()}${suffix}`);
}
