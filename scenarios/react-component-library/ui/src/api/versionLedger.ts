import { API_BASE, boundedFetch, decodeApiError } from "./client";
import { createClient } from "@connectrpc/connect";
import {
  VersionLifecycleService,
  type CleanupItem,
} from "@vrooli/proto-types/react-component-library/v1/versions/lifecycle_pb";
import { transport } from "./client";

export const versionLifecycleClient = createClient(VersionLifecycleService, transport);
export type { CleanupItem };

const endpoint = `${API_BASE}/vrooli.react_component_library.v1.versions.VersionLifecycleService/ListVersionLedger`;
export interface VersionLedgerRow {
  libraryId: string;
  version: string;
  createdAt: string;
  releasedAt: string;
  retiredAt: string;
  lifecycleState: string;
  gatePassCount: number;
  gateFailCount: number;
  testRuns: number;
  testPassRate: number;
  adoptionCurrent: number;
  adoptionPeak: number;
  fileCount: number;
  linesOfCode: number;
  dependencyCount: number;
  presence: string;
}
export async function listVersionLedger(libraryId: string): Promise<VersionLedgerRow[]> {
  const response = await boundedFetch(endpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json", "Connect-Protocol-Version": "1" },
    body: JSON.stringify({ libraryId }),
  });
  if (!response.ok) throw await decodeApiError(response);
  const body = (await response.json()) as { rows?: VersionLedgerRow[] };
  return body.rows ?? [];
}
