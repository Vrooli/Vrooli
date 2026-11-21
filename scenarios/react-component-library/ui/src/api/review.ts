import { createClient } from "@connectrpc/connect";
import {
  DispositionDecision,
  FindingSeverity,
  FindingStatus,
  ReviewService,
  type Finding,
  type ReviewSummary,
} from "@vrooli/proto-types/react-component-library/v1/review/review_pb";

import { transport } from "./client";

const client = createClient(ReviewService, transport);

export { DispositionDecision, FindingSeverity, FindingStatus };
export type { Finding, ReviewSummary };

/** One page of findings plus the filtered total the pager needs. */
export interface FindingPage {
  findings: Finding[];
  total: number;
}

export interface ListFindingsInput {
  status?: FindingStatus;
  severity?: FindingSeverity;
  subjectKind?: string;
  query?: string;
  limit?: number;
  offset?: number;
}

async function fetchFindingsPage(input: ListFindingsInput = {}): Promise<FindingPage> {
  const response = await client.listFindings({
    status: input.status ?? FindingStatus.UNSPECIFIED,
    severity: input.severity ?? FindingSeverity.UNSPECIFIED,
    subjectKind: input.subjectKind ?? "",
    query: input.query ?? "",
    limit: input.limit ?? 25,
    offset: input.offset ?? 0,
  });
  return { findings: response.findings, total: response.total };
}

/** List findings without paging metadata. Existing consumers use this shape. */
export async function listFindings(input: ListFindingsInput = {}): Promise<Finding[]> {
  const page = await fetchFindingsPage({ ...input, limit: input.limit ?? 200, offset: 0 });
  return page.findings;
}

/** One filtered page plus the total the pager needs. */
export async function listFindingsPage(input: ListFindingsInput = {}): Promise<FindingPage> {
  return fetchFindingsPage({ ...input, limit: input.limit ?? 25, offset: input.offset ?? 0 });
}

export async function getFinding(id: string): Promise<Finding | null> {
  const response = await client.getFinding({ id });
  return response.finding ?? null;
}

export async function recordDisposition(input: {
  id: string;
  decision: DispositionDecision;
  reason: string;
  actor: string;
}): Promise<Finding | null> {
  const response = await client.recordDisposition(input);
  return response.finding ?? null;
}

export async function assignFinding(id: string, owner: string): Promise<Finding | null> {
  const response = await client.assignFinding({ id, owner });
  return response.finding ?? null;
}

export async function getReviewSummary(): Promise<ReviewSummary | null> {
  const response = await client.getReviewSummary({});
  return response.summary ?? null;
}
