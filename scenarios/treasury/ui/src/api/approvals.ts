import { createClient } from "@connectrpc/connect";
import type { ApprovalRequest } from "@vrooli/proto-types/treasury/v1/approval/approval_pb";
import { ApprovalStatus } from "@vrooli/proto-types/treasury/v1/approval/approval_pb";
import { TreasuryAdmin } from "@vrooli/proto-types/treasury/v1/authorization/authorization_pb";

import { transport } from "./client";

const treasuryAdmin = createClient(TreasuryAdmin, transport);

function operatorHeaders(operatorToken: string): HeadersInit {
  return { Authorization: `Bearer ${operatorToken}` };
}

export async function listPendingApprovals(operatorToken: string): Promise<ApprovalRequest[]> {
  const response = await treasuryAdmin.listApprovals(
    { status: ApprovalStatus.QUEUED },
    { headers: operatorHeaders(operatorToken) },
  );
  return response.approvals;
}

export async function resolveApproval(
  operatorToken: string,
  approvalId: string,
  resolution: ApprovalStatus.APPROVED | ApprovalStatus.DECLINED,
): Promise<ApprovalRequest> {
  const response = await treasuryAdmin.resolveApproval(
    { approvalId, resolution },
    { headers: operatorHeaders(operatorToken) },
  );
  if (!response.approval) {
    throw new Error("Treasury returned an empty approval resolution");
  }
  return response.approval;
}

export type { ApprovalRequest };
export { ApprovalStatus };
