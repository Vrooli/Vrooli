import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import { requireHostMutationCapability } from "./hostCapabilities";
import { permissionRegistryClient } from "./permissionRegistry";
import {
  HostService,
  SafeguardDisposition,
  type GetHostFactsResponse,
  type ListHostRequirementsResponse,
  type ListTargetsResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/host/host_pb";

const client = createClient(HostService, onboardingTransport());

export interface HostFacts {
  available: boolean; reason?: string; cpu_count?: number; memory_total_bytes?: number;
  memory_available_bytes?: number; disk_free_bytes?: number; gpus?: string[]; platform?: string;
}
export interface HostConfigProperty {
  type?: string; title?: string; description?: string; enum?: unknown[]; default?: unknown;
}
export interface HostConfigSchema {
  type?: string; properties?: Record<string, HostConfigProperty>;
}
export interface HostRequirement {
  name: string; required: boolean; reason: string; notes?: string; description?: string;
  risk?: string; privilege?: string; bundling?: string; platforms?: string[]; commands?: string[];
  config_schema?: HostConfigSchema; config?: Record<string, unknown>; status: string;
  disposition?: "ready" | "missing" | "permission_denied" | "content_mismatch" | "deferred" | "unsupported" | "not_applicable";
  detail?: string; remediation?: string;
}
export interface HostRequirementsResponse { tools: HostRequirement[]; safeguards: HostRequirement[] }
export interface OnboardingTarget {
  id: string; name?: string; status?: string; os?: string; architecture?: string; kind?: string;
  online?: boolean; available?: boolean; reason?: string; next_action?: string;
  capabilities?: string[]; scopes?: string[];
  readiness?: Array<{ identity: string; label: string; passed: boolean; state: string; detail?: string; recovery_action?: string }>;
}
export interface TargetsResponse { targets: OnboardingTarget[]; error?: string }

export async function fetchHostFacts(target = "local"): Promise<HostFacts> {
  return hostFactsFromProto(await client.getHostFacts({ target }) as unknown as GetHostFactsResponse);
}
export async function fetchHostRequirements(target = "local"): Promise<HostRequirementsResponse> {
  return requirementsFromProto(await client.listHostRequirements({ target }) as unknown as ListHostRequirementsResponse);
}
export async function fetchTargets(): Promise<TargetsResponse> {
  const response = await client.listTargets({}) as unknown as ListTargetsResponse;
  return { targets: response.targets.map((target) => ({ id: target.id, name: target.name, status: target.status, os: target.os, architecture: target.architecture, kind: target.kind, online: target.online, available: target.available, reason: target.reason, next_action: target.nextAction, capabilities: target.capabilities, scopes: target.scopes, readiness: target.readiness.map((check) => ({ identity: check.identity, label: check.label, passed: check.passed, state: check.state, detail: check.detail, recovery_action: check.recoveryAction })) })), error: response.error || undefined };
}

export async function writeHostSafeguardPermission(name: string, value: Record<string, unknown>, target = "local") {
  requireHostMutationCapability(target, "safeguard");
  const permissionId = `onboarding/host-safeguard/${name}`;
  const current = await permissionRegistryClient.readPermission({ permissionId, owner: "vrooli-onboarding" });
  const randomId = idempotencyId();
  return permissionRegistryClient.writePermission({
    permissionId, owner: "vrooli-onboarding", expectedRevision: current.revision,
    idempotencyKey: `host-safeguard:${name}:${randomId}`,
    valueJson: new TextEncoder().encode(JSON.stringify(value)),
  });
}
export async function writeHostToolPermission(name: string, optedIn: boolean, target = "local") {
  requireHostMutationCapability(target, "tool");
  const permissionId = `onboarding/host-tool/selected-closure/${name}`;
  const current = await permissionRegistryClient.readPermission({ permissionId, owner: "vrooli-onboarding" });
  const randomId = idempotencyId();
  return permissionRegistryClient.writePermission({
    permissionId, owner: "vrooli-onboarding", expectedRevision: current.revision,
    idempotencyKey: `host-tool:${name}:${randomId}`,
    valueJson: new TextEncoder().encode(JSON.stringify({ opted_in: optedIn })),
  });
}
export function setNotificationRecipient(subject: string, target = "local") {
  requireHostMutationCapability(target, "notification-recipient");
  const permissionId = "onboarding/operator-state";
  const trimmedSubject = subject.trim();
  if (!trimmedSubject) throw new Error("Notification recipient subject is required");
  return permissionRegistryClient.readPermission({ permissionId, owner: "vrooli-onboarding" }).then((current) => {
    if (!current.revision) throw new Error("Operator-state owner returned no source revision");
    const randomId = idempotencyId();
    return permissionRegistryClient.writePermission({
      permissionId,
      owner: "vrooli-onboarding",
      expectedRevision: current.revision,
      idempotencyKey: `host-notification-recipient:${randomId}`,
      valueJson: new TextEncoder().encode(JSON.stringify({
        state: { notifications: { recipient: trimmedSubject } },
        update_mask: ["notifications.recipient"],
      })),
    });
  });
}

function idempotencyId(): string {
  return globalThis.crypto.randomUUID();
}

function hostFactsFromProto(response: GetHostFactsResponse): HostFacts {
  return { available: response.available, reason: response.reason || undefined, cpu_count: response.cpuCount, memory_total_bytes: response.memoryTotalBytes === undefined ? undefined : Number(response.memoryTotalBytes), memory_available_bytes: response.memoryAvailableBytes === undefined ? undefined : Number(response.memoryAvailableBytes), disk_free_bytes: response.diskFreeBytes === undefined ? undefined : Number(response.diskFreeBytes), gpus: response.gpus, platform: response.platform || undefined };
}
function requirementsFromProto(response: ListHostRequirementsResponse): HostRequirementsResponse {
  return { tools: response.tools.map(requirementFromProto), safeguards: response.safeguards.map(requirementFromProto) };
}
function requirementFromProto(item: ListHostRequirementsResponse["tools"][number]): HostRequirement {
  return { name: item.name, required: item.required, reason: item.reason, notes: item.notes || undefined, description: item.description || undefined, risk: item.risk || undefined, privilege: item.privilege || undefined, bundling: item.bundling || undefined, platforms: item.platforms, commands: item.commands, config_schema: item.configSchema, config: item.config, status: item.status, disposition: dispositionName(item.disposition), detail: item.detail || undefined, remediation: item.remediation || undefined };
}
function dispositionName(value: SafeguardDisposition): HostRequirement["disposition"] {
  switch (value) {
    case SafeguardDisposition.READY: return "ready";
    case SafeguardDisposition.MISSING: return "missing";
    case SafeguardDisposition.PERMISSION_DENIED: return "permission_denied";
    case SafeguardDisposition.CONTENT_MISMATCH: return "content_mismatch";
    case SafeguardDisposition.DEFERRED: return "deferred";
    case SafeguardDisposition.UNSUPPORTED: return "unsupported";
    case SafeguardDisposition.NOT_APPLICABLE: return "not_applicable";
    default: return undefined;
  }
}
