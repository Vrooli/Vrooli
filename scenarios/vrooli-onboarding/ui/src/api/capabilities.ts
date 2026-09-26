import { createClient } from "@connectrpc/connect";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { JsonObject } from "@bufbuild/protobuf";
import { onboardingTransport } from "./base";
import {
  CapabilitiesService,
  CapabilityState,
  type CapabilityCandidate as ProtoCapabilityCandidate,
  type CapabilityEvidence as ProtoCapabilityEvidence,
  type CapabilityStatus as ProtoCapabilityStatus,
  type ListCapabilitiesResponse,
  type PreviewCapabilityResponse,
  type ApplyCapabilityResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/capabilities/capabilities_pb";

const client = createClient(CapabilitiesService, onboardingTransport());

export type CapabilityStateName =
  | "discovered" | "needs_operator_input" | "ready_to_preview" | "applying"
  | "verifying" | "ready" | "retryable_failure" | "degraded" | "unsupported";
export type CapabilityInputKind = "secret" | "choice" | "confirm" | "path" | "enum" | "boolean" | "duration" | "confirmation";
export interface CapabilityCandidate {
  id: string; kind: string; label: string; location?: string; stable_identity?: string;
  device_identity?: string; writable: boolean; physical_independence?: string;
  status: string; risk?: string; remediation?: string; metadata?: Record<string, string>;
}
export interface CapabilityInput {
  id: string; kind: CapabilityInputKind; label: string; description?: string;
  required: boolean; declinable?: boolean; options?: string[]; default?: string; candidates?: CapabilityCandidate[]; validation?: string;
  constraints?: { min_length?: number; max_length?: number; min_duration?: string; max_duration?: string };
}
export interface CapabilityDescriptor {
  version: string; id: string; owner: string; title: string; description?: string; risk?: string;
  scope?: string; purpose?: string; sensitivity?: string; disposition?: string; disposition_reason?: string; reference_url?: string;
  applicability?: { platforms?: string[]; environments?: string[]; targets?: string[] };
  provenance?: { requester?: string; scope?: string; grant_source?: string; revocation_limit?: string };
  lifecycle?: { preview: boolean; apply: boolean; verify: boolean; revoke: boolean; recover: boolean; recovery?: string };
  inputs?: CapabilityInput[]; prerequisites?: string[];
  policy: { requires_confirmation: boolean; idempotent: boolean; retryable: boolean; protected_roots?: string[]; remediation?: string };
  evidence: { kinds?: string[]; stages?: string[]; required_fields?: string[]; secret_free: boolean; freshness?: string };
  remediation?: string;
}
export interface CapabilityEvidence {
  kind: string; stage?: string; artifact_identity: string; source_generation?: string; checksum?: string;
  coverage?: string[]; observed_at: string; verified: boolean; remediation?: string;
  credential_ref?: { logical_id: string; field: string; version: string };
  target_id?: string; environment?: string; account_identity?: string; operation?: string;
  owner?: string; context_digest?: string; catalog_revision?: string; configuration_revision?: string; provider_adapter_version?: string;
  expires_at?: string; next_action?: string; effect_class?: string; effects_used?: number; cleanup_completed?: boolean;
}
export interface CapabilityStatus {
  descriptor: CapabilityDescriptor; state: CapabilityStateName; candidates?: CapabilityCandidate[];
  missing_inputs?: string[]; evidence?: CapabilityEvidence[]; remediation?: string; updated_at: string;
}
export interface CapabilityStatusResponse { capabilities: CapabilityStatus[]; count: number }
export interface CapabilityPreview {
  capability_id: string; plan_id: string; state: CapabilityStateName; mutations?: Array<{ id: string; summary: string; reversible: boolean }>;
  candidates?: CapabilityCandidate[]; remediation?: string; expires_at?: string;
}
export interface CapabilityResult {
  capability_id: string; state: CapabilityStateName; outcome: string; retryable: boolean; error_code?: string;
  retry_after_seconds?: number; next_action?: string; remediation?: string; evidence?: CapabilityEvidence[]; mutations?: Array<{ id: string; summary: string; reversible: boolean }>;
  completed_at?: string;
}
export interface CapabilityVerificationResult {
  capability_id: string; outcome: string; retryable: boolean; error_code?: string;
  retry_after_seconds?: number; next_action?: string; remediation?: string; evidence: CapabilityEvidence[];
}
export interface CapabilityActionRequest { capability_id: string; idempotency_key?: string; confirm: boolean; inputs: Record<string, unknown> }

// Keep the client source compatible with an older projected node_modules copy
// during a rolling proto refresh. The governed generated package is the runtime
// source; this narrow cast only lets TypeScript consume the additive fields.
type WireCapabilityInput = { declinable?: boolean; constraints?: { minLength?: number; maxLength?: number; minDuration?: string; maxDuration?: string } };
type WireCapabilityEvidenceContract = { stages?: string[] };
type WireCapabilityEvidence = ProtoCapabilityEvidence & {
  stage?: string;
  owner?: string;
  contextDigest?: string;
  catalogRevision?: string;
  configurationRevision?: string;
  providerAdapterVersion?: string;
};
type WireCapabilityDescriptor = NonNullable<ProtoCapabilityStatus["descriptor"]> & {
  scope?: string; purpose?: string; sensitivity?: string; disposition?: string; dispositionReason?: string; referenceUrl?: string;
  applicability?: { platforms: string[]; environments: string[]; targets: string[] };
  provenance?: { requester: string; scope: string; grantSource: string; revocationLimit: string };
  lifecycle?: { preview: boolean; apply: boolean; verify: boolean; revoke: boolean; recover: boolean; recovery: string };
};

export async function fetchCapabilities(target = "local"): Promise<CapabilityStatusResponse> {
  const response = await client.listCapabilities({ target }) as unknown as ListCapabilitiesResponse;
  return { capabilities: response.capabilities.map(statusFromProto), count: response.count };
}

export async function fetchCapabilityStatus(target = "local"): Promise<CapabilityStatusResponse> {
  const response = await client.getCapabilityStatus({ target }) as unknown as { statuses: ProtoCapabilityStatus[]; count: number };
  return { capabilities: response.statuses.map(statusFromProto), count: response.count };
}

export async function previewCapability(request: CapabilityActionRequest, target = "local"): Promise<CapabilityPreview> {
  const response = await client.previewCapability({ target, action: action(request) }) as unknown as PreviewCapabilityResponse;
  return previewFromProto(response);
}

export async function applyCapability(request: CapabilityActionRequest, target = "local"): Promise<CapabilityResult> {
  const response = await client.applyCapability({ target, action: action(request) }) as unknown as ApplyCapabilityResponse;
  return resultFromProto(response);
}

export async function verifyCapability(capabilityId: string, target = "local"): Promise<CapabilityVerificationResult> {
  // The generated workspace package can lag one governed projection during
  // a rolling proto refresh. Keep the additive RPC typed at this boundary so
  // the UI remains source-compatible while the canonical package updates.
  const verificationClient = client as unknown as { verifyCapability(request: unknown): Promise<{ capabilityId?: string; evidence: ProtoCapabilityEvidence[]; outcome?: string; errorCode?: string; retryable?: boolean; retryAfterSeconds?: bigint | number; nextAction?: string; remediation?: string }> };
  const response = await verificationClient.verifyCapability({
    target,
    verification: {
      capabilityId,
      targetId: target,
      environment: "",
      accountIdentity: "",
      operation: "readiness-check",
      context: {},
      effectClass: "read_only",
      maxOperations: 0,
      cleanupPolicy: "",
      timeoutSeconds: 30n,
    },
  });
  return {
    capability_id: response.capabilityId ?? capabilityId,
    outcome: response.outcome ?? (response.errorCode ? "verification_failed" : "verified"),
    retryable: response.retryable ?? !response.errorCode,
    error_code: response.errorCode || undefined,
    retry_after_seconds: response.retryAfterSeconds === undefined ? undefined : Number(response.retryAfterSeconds),
    next_action: response.nextAction || undefined,
    remediation: response.remediation || undefined,
    evidence: (response.evidence ?? []).map(evidenceFromProto),
  };
}

function action(request: CapabilityActionRequest) {
  return {
    capabilityId: request.capability_id,
    idempotencyKey: request.idempotency_key ?? "",
    confirm: request.confirm,
    inputs: request.inputs as JsonObject,
  };
}

function statusFromProto(status: ProtoCapabilityStatus): CapabilityStatus {
  const descriptor = status.descriptor;
  const extendedDescriptor = descriptor as unknown as WireCapabilityDescriptor | undefined;
  return {
    descriptor: {
      version: descriptor?.version ?? "", id: descriptor?.id ?? "", owner: descriptor?.owner ?? "", title: descriptor?.title ?? "",
      description: descriptor?.description, risk: descriptor?.risk,
      inputs: descriptor?.inputs.map((input) => { const extendedInput = input as WireCapabilityInput; return { id: input.id, kind: input.kind as CapabilityInputKind, label: input.label, description: input.description, required: input.required, declinable: extendedInput.declinable, options: input.options, default: input.defaultValue, candidates: input.candidates.map(candidateFromProto), validation: input.validation, constraints: extendedInput.constraints ? { min_length: extendedInput.constraints.minLength, max_length: extendedInput.constraints.maxLength, min_duration: extendedInput.constraints.minDuration, max_duration: extendedInput.constraints.maxDuration } : undefined }; }),
      scope: extendedDescriptor?.scope, purpose: extendedDescriptor?.purpose, sensitivity: extendedDescriptor?.sensitivity, disposition: extendedDescriptor?.disposition, disposition_reason: extendedDescriptor?.dispositionReason, reference_url: extendedDescriptor?.referenceUrl,
      applicability: extendedDescriptor?.applicability ? { platforms: extendedDescriptor.applicability.platforms, environments: extendedDescriptor.applicability.environments, targets: extendedDescriptor.applicability.targets } : undefined,
      provenance: extendedDescriptor?.provenance ? { requester: extendedDescriptor.provenance.requester, scope: extendedDescriptor.provenance.scope, grant_source: extendedDescriptor.provenance.grantSource, revocation_limit: extendedDescriptor.provenance.revocationLimit } : undefined,
      lifecycle: extendedDescriptor?.lifecycle ? { preview: extendedDescriptor.lifecycle.preview, apply: extendedDescriptor.lifecycle.apply, verify: extendedDescriptor.lifecycle.verify, revoke: extendedDescriptor.lifecycle.revoke, recover: extendedDescriptor.lifecycle.recover, recovery: extendedDescriptor.lifecycle.recovery } : undefined,
      prerequisites: descriptor?.prerequisites,
      policy: { requires_confirmation: descriptor?.policy?.requiresConfirmation ?? false, idempotent: descriptor?.policy?.idempotent ?? false, retryable: descriptor?.policy?.retryable ?? false, protected_roots: descriptor?.policy?.protectedRoots, remediation: descriptor?.policy?.remediation },
      evidence: { kinds: descriptor?.evidence?.kinds, stages: (descriptor?.evidence as WireCapabilityEvidenceContract | undefined)?.stages, required_fields: descriptor?.evidence?.requiredFields, secret_free: descriptor?.evidence?.secretFree ?? false, freshness: descriptor?.evidence?.freshness },
      remediation: descriptor?.remediation,
    },
    state: stateName(status.state), candidates: status.candidates.map(candidateFromProto), missing_inputs: status.missingInputs,
    evidence: status.evidence.map(evidenceFromProto), remediation: status.remediation, updated_at: status.updatedAt ? timestampDate(status.updatedAt).toISOString() : "",
  };
}

function candidateFromProto(candidate: ProtoCapabilityCandidate): CapabilityCandidate {
  return { id: candidate.id, kind: candidate.kind, label: candidate.label, location: candidate.location, stable_identity: candidate.stableIdentity, device_identity: candidate.deviceIdentity, writable: candidate.writable, physical_independence: candidate.physicalIndependence, status: candidate.status, risk: candidate.risk, remediation: candidate.remediation, metadata: candidate.metadata };
}

function evidenceFromProto(evidence: ProtoCapabilityEvidence): CapabilityEvidence {
  const wireEvidence = evidence as WireCapabilityEvidence;
  return {
    kind: evidence.kind, stage: wireEvidence.stage, artifact_identity: evidence.artifactIdentity, source_generation: evidence.sourceGeneration,
    checksum: evidence.checksum, coverage: evidence.coverage, observed_at: evidence.observedAt ? timestampDate(evidence.observedAt).toISOString() : "",
    verified: evidence.verified, remediation: evidence.remediation,
    credential_ref: evidence.credentialRef ? { logical_id: evidence.credentialRef.logicalId, field: evidence.credentialRef.field, version: evidence.credentialRef.version } : undefined,
    target_id: evidence.targetId, environment: evidence.environment, account_identity: evidence.accountIdentity, operation: evidence.operation,
    owner: wireEvidence.owner, context_digest: wireEvidence.contextDigest, catalog_revision: wireEvidence.catalogRevision, configuration_revision: wireEvidence.configurationRevision, provider_adapter_version: wireEvidence.providerAdapterVersion,
    expires_at: evidence.expiresAt ? timestampDate(evidence.expiresAt).toISOString() : undefined, next_action: evidence.nextAction,
    effect_class: evidence.effectClass, effects_used: evidence.effectsUsed, cleanup_completed: evidence.cleanupCompleted,
  };
}

function previewFromProto(response: PreviewCapabilityResponse): CapabilityPreview {
  return { capability_id: response.capabilityId, plan_id: response.planId, state: stateName(response.state), mutations: response.mutations.map((mutation) => ({ id: mutation.id, summary: mutation.summary, reversible: mutation.reversible })), candidates: response.candidates.map(candidateFromProto), remediation: response.remediation, expires_at: response.expiresAt ? timestampDate(response.expiresAt).toISOString() : undefined };
}

function resultFromProto(response: ApplyCapabilityResponse): CapabilityResult {
  const extended = response as ApplyCapabilityResponse & { retryAfterSeconds?: bigint | number; nextAction?: string };
  return { capability_id: response.capabilityId, state: stateName(response.state), outcome: response.outcome, retryable: response.retryable, error_code: response.errorCode, retry_after_seconds: extended.retryAfterSeconds === undefined ? undefined : Number(extended.retryAfterSeconds), next_action: extended.nextAction || undefined, remediation: response.remediation, evidence: response.evidence.map(evidenceFromProto), mutations: response.mutations.map((mutation) => ({ id: mutation.id, summary: mutation.summary, reversible: mutation.reversible })), completed_at: response.completedAt ? timestampDate(response.completedAt).toISOString() : undefined };
}

function stateName(state: CapabilityState): CapabilityStateName {
  switch (state) {
    case CapabilityState.DISCOVERED: return "discovered";
    case CapabilityState.NEEDS_OPERATOR_INPUT: return "needs_operator_input";
    case CapabilityState.READY_TO_PREVIEW: return "ready_to_preview";
    case CapabilityState.APPLYING: return "applying";
    case CapabilityState.VERIFYING: return "verifying";
    case CapabilityState.READY: return "ready";
    case CapabilityState.RETRYABLE_FAILURE: return "retryable_failure";
    case CapabilityState.DEGRADED: return "degraded";
    case CapabilityState.UNSUPPORTED: return "unsupported";
    default: return "discovered";
  }
}
