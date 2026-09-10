import { createClient } from "@connectrpc/connect";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { onboardingTransport } from "./base";
import {
  ReadinessService,
  ReadinessState,
  type GetReadinessResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/readiness/readiness_pb";
export interface CompletionBlocker {
  kind: "credential" | "host" | "recovery" | "apply" | "readiness";
  name: string;
  reason: string;
  remediation: string;
}
export interface ReadinessItem {
  name: string;
  category?: "integration" | "system";
  status: "ready" | "degraded" | "missing" | "unsupported" | "deferred";
  detail?: string;
  remediation?: string;
  kind?: "tool" | "safeguard";
  required?: boolean;
}

const client = createClient(ReadinessService, onboardingTransport());

export interface ReadinessResponse {
  status: "ready" | "degraded" | "missing" | "unsupported" | "deferred";
  scenarios: string[];
  resources: string[];
  credentials: Array<{
    resource: string;
    logical_id: string;
    field: string;
    label: string;
    description?: string;
    obtain_url?: string;
    required: boolean;
    provisioning?: string;
    derived_from?: string;
    status: string;
    detail?: string;
  }>;
  hosts: ReadinessItem[];
  integrations: ReadinessItem[];
  checked_at: string;
  credential_diagnosis?: Record<string, any>;
  recovery?: {
    receipt_exists: boolean;
    exported_at?: string;
    entry_count: number;
    uncovered: string[];
    required_absent?: string[];
    required_absent_details?: Array<{ address: string; description?: string }>;
    root_copy_issues: string[];
  };
  blockers: CompletionBlocker[];
  degraded: CompletionBlocker[];
  degraded_digest: string;
  degraded_acknowledged: boolean;
}

export function fetchReadiness(target = "local"): Promise<ReadinessResponse> {
  return (client.getReadiness({ target }) as unknown as Promise<GetReadinessResponse>).then(toReadinessResponse);
}

export function acknowledgeDegraded(readinessDigest: string, target = "local") {
  return client.acknowledgeDegradedReadiness({ target, readinessDigest });
}

function toReadinessResponse(response: GetReadinessResponse): ReadinessResponse {
  return {
    status: stateName(response.status),
    scenarios: response.scenarios,
    resources: response.resources,
    credentials: response.credentials.map((item) => ({
      resource: item.resource, logical_id: item.logicalId, field: item.field, label: item.label,
      description: item.description, obtain_url: item.obtainUrl, required: item.required,
      provisioning: item.provisioning, derived_from: item.derivedFrom,
      status: item.legacyStatus || stateName(item.status), detail: item.detail,
    })),
    hosts: response.hosts.map((host) => itemFromProto(host.item, host.kind, host.required)),
    integrations: response.integrations.map((item) => itemFromProto(item)),
    checked_at: response.checkedAt ? timestampDate(response.checkedAt).toISOString() : "",
    credential_diagnosis: response.credentialDiagnosis?.fields as Record<string, any> | undefined,
    recovery: response.recovery ? {
      receipt_exists: response.recovery.receiptExists, exported_at: response.recovery.exportedAt,
      entry_count: response.recovery.entryCount, uncovered: response.recovery.uncovered,
      required_absent: response.recovery.requiredAbsent,
      required_absent_details: response.recovery.requiredAbsentDetails.map((gap) => ({ address: gap.address, description: gap.description })),
      root_copy_issues: response.recovery.rootCopyIssues,
    } : undefined,
    blockers: response.blockers.map(blockerFromProto), degraded: response.degraded.map(blockerFromProto),
    degraded_digest: response.degradedDigest, degraded_acknowledged: response.degradedAcknowledged,
  };
}

function itemFromProto(item: GetReadinessResponse["integrations"][number] | undefined, kind?: string, required?: boolean): ReadinessItem {
  const category = item?.category === "integration" || item?.category === "system" ? item.category : undefined;
  const itemKind = kind === "tool" || kind === "safeguard" ? kind : undefined;
  return { name: item?.name ?? "", category, status: stateName(item?.status), detail: item?.detail, remediation: item?.remediation, kind: itemKind, required: required ?? item?.required };
}

function blockerFromProto(item: GetReadinessResponse["blockers"][number]): CompletionBlocker {
  return { kind: item.kind as CompletionBlocker["kind"], name: item.name, reason: item.reason, remediation: item.remediation };
}

function stateName(value: ReadinessState | undefined): ReadinessResponse["status"] {
  switch (value) {
    case ReadinessState.READY: return "ready";
    case ReadinessState.MISSING: return "missing";
    case ReadinessState.DEGRADED: return "degraded";
    case ReadinessState.UNSUPPORTED: return "unsupported";
    case ReadinessState.DEFERRED: return "deferred";
    case ReadinessState.NOT_APPLICABLE: return "deferred";
    default: return "deferred";
  }
}
