import { createClient } from '@connectrpc/connect';
import { ExportsService } from '@vrooli/proto-types/browser-automation-studio/v1/exports/exports_pb';

import { newGatewayRootOperationID } from './ai';
import { transport } from './client';

export const exportsClient = createClient(ExportsService, transport);

const pendingCaptionOperations = new Map<string, string>();

/** Keep one recovery ID for an export caption until the Gateway confirms a
 * result. Uncertain retries reuse the ID; success starts a new operation next
 * time the user explicitly regenerates the caption.
 */
export async function generateExportCaption(exportID: string) {
  let operationID = pendingCaptionOperations.get(exportID);
  if (!operationID) {
    operationID = newGatewayRootOperationID();
    pendingCaptionOperations.set(exportID, operationID);
    while (pendingCaptionOperations.size > 64) {
      const oldest = pendingCaptionOperations.keys().next().value;
      if (!oldest) break;
      pendingCaptionOperations.delete(oldest);
    }
  }
  const headers = new Headers({ 'Idempotency-Key': operationID });
  const response = await exportsClient.generateExportCaption({ id: exportID }, { headers });
  if (pendingCaptionOperations.get(exportID) === operationID) pendingCaptionOperations.delete(exportID);
  return response;
}

export function _resetCaptionOperationIDs(): void {
  pendingCaptionOperations.clear();
}
