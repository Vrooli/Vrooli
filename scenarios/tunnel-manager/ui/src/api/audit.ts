import { createClient } from "@connectrpc/connect";
import {
  AuditService,
  AuditStatus,
  type PortAuditResult,
  type RunAuditResponse,
} from "@vrooli/proto-types/tunnel-manager/v1/audit/audit_pb";

import { transport } from "./client";

// auditClient is the generated Connect-Web client for AuditService —
// port-compliance auditing of exposed scenarios' service.json. Backs the
// Audit surface under ui/src/features/audit/.
export const auditClient = createClient(AuditService, transport);

/** runAudit computes port-compliance findings across all manifested routes. */
export async function runAudit(): Promise<RunAuditResponse> {
  return auditClient.runAudit({});
}

export { AuditStatus };
export type { PortAuditResult, RunAuditResponse };
