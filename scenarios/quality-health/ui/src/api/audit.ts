import { createClient } from "@connectrpc/connect";
import {
  AuditService,
  type AuditQualityResponse,
  type ExplainFindingResponse,
  type FixConfigResponse,
  type ListContractsResponse,
} from "@vrooli/proto-types/quality-health/v1/audit/audit_pb";

import { transport } from "./client";

export const auditClient = createClient(AuditService, transport);

export type {
  AuditQualityResponse,
  ExplainFindingResponse,
  FixConfigResponse,
  ListContractsResponse,
};
