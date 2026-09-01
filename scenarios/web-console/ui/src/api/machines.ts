// The screens this contract serves are specified in the repository-level design
// proposal `docs/reference/cross-platform-effort/machine-linking-ux-2026-08-26.html`.
// It is not a `DOC:` marker because that reference kind resolves inside this
// scenario's own docs tree, and the proposal spans several scenarios.
import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import {
  MachineService,
  FleetState,
} from "@vrooli/proto-types/web-console/v1/machines/machines_pb";
import type {
  Machine as MachineMessage,
  JoinRequest as JoinRequestMessage,
  PermissionPreset as PresetMessage,
} from "@vrooli/proto-types/web-console/v1/machines/machines_pb";
import type { PreflightOnboardingResponse, StartOnboardingResponse, GetOnboardingResponse, WaitOnboardingResponse } from "@vrooli/proto-types/vrooli-bridge/v1/onboard/onboard_pb";
import { AnswerSecretRequestSchema, CreateGrantRequestSchema, RevokeGrantRequestSchema, type CredentialGrant, type ListGrantsResponse, type CreateGrantRequest, type RevokeGrantRequest } from "@vrooli/proto-types/vrooli-bridge/v1/credentialgrant/credentialgrant_pb";

import { transport } from "./client";
import { decodeTarget, type TerminalTarget } from "./targets";

export const machineClient = createClient(MachineService, transport);

type OnboardingClient = {
  preflightOnboarding(request: { host: string; port: number; user: string; machineId: string }): Promise<PreflightOnboardingResponse>;
  startOnboarding(request: { machineId: string; host: string; port: number; user: string; sshPassword: string; nodeName: string; setupPreset: string }): Promise<StartOnboardingResponse>;
  getOnboarding(request: { onboardingOpId: string }): Promise<GetOnboardingResponse>;
  waitOnboarding(request: { onboardingOpId: string; timeoutSeconds: number }): Promise<WaitOnboardingResponse>;
  getConfiguration(request: { machineId: string }): Promise<{ targetId: string; questionsJson: Uint8Array; readinessJson: Uint8Array; machineDetailJson: Uint8Array }>;
  resolveConfiguration(request: { machineId: string; answersJson: Uint8Array }): Promise<{ targetId: string; resultJson: Uint8Array }>;
  listCredentialGrants(request: { machineId: string }): Promise<ListGrantsResponse>;
  createCredentialGrant(request: CreateGrantRequest): Promise<CredentialGrant>;
  revokeCredentialGrant(request: RevokeGrantRequest): Promise<CredentialGrant>;
  answerSecret(request: { nodeId: string; logicalId: string; field: string; value: string }): Promise<CredentialGrant>;
  reapplyConfiguration(request: { machineId: string }): Promise<{ targetId: string; resultJson: Uint8Array }>;
  getConfigurationApplyStatus(request: { machineId: string; runId: string }): Promise<{ targetId: string; resultJson: Uint8Array }>;
};
const onboardingClient = machineClient as typeof machineClient & OnboardingClient;

export function preflightOnboarding(request: { host: string; port?: number; user: string; machineId?: string }): Promise<PreflightOnboardingResponse> {
  return onboardingClient.preflightOnboarding({ host: request.host, port: request.port ?? 22, user: request.user, machineId: request.machineId ?? "" });
}

export function startOnboarding(request: { machineId: string; host: string; port?: number; user: string; sshPassword: string; nodeName?: string; setupPreset?: string }): Promise<StartOnboardingResponse> {
  return onboardingClient.startOnboarding({ machineId: request.machineId, host: request.host, port: request.port ?? 22, user: request.user, sshPassword: request.sshPassword, nodeName: request.nodeName ?? "", setupPreset: request.setupPreset ?? "minimal" });
}

export function getOnboarding(onboardingOpId: string): Promise<GetOnboardingResponse> {
  return onboardingClient.getOnboarding({ onboardingOpId });
}

export function waitOnboarding(onboardingOpId: string, timeoutSeconds = 30): Promise<WaitOnboardingResponse> {
  return onboardingClient.waitOnboarding({ onboardingOpId, timeoutSeconds });
}

export interface ConfigurationQuestion {
  id: string;
  kind: "secret" | "choice" | "confirm" | "path" | "enum" | "boolean" | "duration" | "confirmation";
  title: string;
  description?: string;
  default?: string;
  options?: string[];
  required: boolean;
  validation?: string;
  owner?: string;
  input_id?: string;
  candidates?: Array<{
    id: string;
    label: string;
    status?: string;
    risk?: string;
    remediation?: string;
  }>;
}

export interface MachineConfigurationDetail {
  machine?: { desiredProfileId?: string; desiredProfileVersion?: string; desiredSelectionJson?: string; appliedProfileId?: string; appliedProfileVersion?: string; appliedSelectionJson?: string; appliedAt?: string };
  auditEvents?: Array<{ actor?: string; action?: string; detail?: string; createdAt?: string }>;
  drift?: Array<{ kind?: string; name?: string; reason?: string }>;
  readiness?: { ready?: boolean; reasons?: string[] };
  effectivePolicy?: unknown;
}

export async function getConfiguration(machineId: string): Promise<{ targetId: string; questions: ConfigurationQuestion[]; readiness: unknown; detail: MachineConfigurationDetail | null }> {
  const response = await onboardingClient.getConfiguration({ machineId });
  const decode = (value: Uint8Array) => JSON.parse(new TextDecoder().decode(value)) as unknown;
  const questions = decode(response.questionsJson) as { requests?: ConfigurationQuestion[] };
  const detail = response.machineDetailJson.length ? decode(response.machineDetailJson) as MachineConfigurationDetail : null;
  return { targetId: response.targetId, questions: questions.requests ?? [], readiness: decode(response.readinessJson), detail };
}

export async function resolveConfiguration(machineId: string, answers: Array<{ request_id: string; value: string }>): Promise<{ targetId: string; result: unknown }> {
  const response = await onboardingClient.resolveConfiguration({ machineId, answersJson: new TextEncoder().encode(JSON.stringify(answers)) });
  return { targetId: response.targetId, result: JSON.parse(new TextDecoder().decode(response.resultJson)) as unknown };
}

export function listCredentialGrants(machineId: string): Promise<ListGrantsResponse> {
  return onboardingClient.listCredentialGrants({ machineId });
}

export function createCredentialGrant(request: { nodeId: string; logicalId: string; field: string; class: string; retention: string }): Promise<CredentialGrant> {
  return onboardingClient.createCredentialGrant(create(CreateGrantRequestSchema, { nodeId: request.nodeId, logicalId: request.logicalId, field: request.field, class: request.class, retention: request.retention, generation: 1n }));
}

export function revokeCredentialGrant(id: string): Promise<CredentialGrant> {
  return onboardingClient.revokeCredentialGrant(create(RevokeGrantRequestSchema, { id }));
}

export function answerSecret(request: { nodeId: string; logicalId: string; field: string; value: string }): Promise<CredentialGrant> {
  return onboardingClient.answerSecret(create(AnswerSecretRequestSchema, { nodeId: request.nodeId, logicalId: request.logicalId, field: request.field, value: request.value, class: "user_prompt", retention: "durable" }));
}

export async function reapplyConfiguration(machineId: string): Promise<{ targetId: string; result: unknown }> {
  const response = await onboardingClient.reapplyConfiguration({ machineId });
  return { targetId: response.targetId, result: JSON.parse(new TextDecoder().decode(response.resultJson)) as unknown };
}

export async function getConfigurationApplyStatus(machineId: string, runId: string): Promise<{ targetId: string; result: unknown }> {
  const response = await onboardingClient.getConfigurationApplyStatus({ machineId, runId });
  return { targetId: response.targetId, result: JSON.parse(new TextDecoder().decode(response.resultJson)) as unknown };
}

export type FleetStatus = "ready" | "empty" | "unenrolled" | "unreachable";

/** The effect families a grant confers, least to most consequential. */
export type GrantEffect = "read" | "write" | "destructive";

export interface Grant {
  /** The grant as a sentence. Nobody should read a scope string to understand it. */
  summary: string;
  effects: GrantEffect[];
  /** How many apps the grant names. Meaningless when coversAllApps is true. */
  appCount: number;
  coversAllApps: boolean;
  /** The concrete scopes, kept for audit rather than for the primary reading. */
  scopes: string[];
  /** The preset this grant exactly matches, or "" when it was customized. */
  preset: string;
}

export interface Machine {
  target: TerminalTarget;
  grant: Grant;
  drift?: MachineDrift[];
  /** Seconds since the control plane last heard from this machine. */
  heartbeatAgeSeconds: number;
  /** False for the computer the console runs on. */
  manageable: boolean;
}

export interface MachineDrift {
  kind: string;
  name: string;
  reason: string;
}

export interface JoinRequest {
  id: string;
  name: string;
  os: string;
  arch: string;
  endpoint: string;
  /** Derived from both machines' keys — the one field the sender cannot choose. */
  confirmationWords: string[];
  keyFingerprint: string;
  requestedAgeSeconds: number;
}

export interface PermissionPreset {
  name: string;
  title: string;
  description: string;
  scopes: string[];
  withholds: string[];
  summary: string;
  effects: GrantEffect[];
  appCount: number;
}

export interface ControlPlane {
  reachable: boolean;
  /**
   * The API base the server dials. Identity and diagnostics only — never an
   * href. It is resolved server-side against loopback and the API port, so a
   * browser that opens it reaches a Connect endpoint, or the wrong computer.
   */
  endpoint: string;
  detail: string;
  /**
   * The control plane's interface, resolved by the server against this
   * browser's own origin. Empty when it could not be located, in which case
   * call sites hide the affordance rather than render a dead link.
   */
  consoleUrl: string;
}

export interface Fleet {
  status: FleetStatus;
  machines: Machine[];
  joinRequests: JoinRequest[];
  presets: PermissionPreset[];
  message: string;
  recoveryAction: string;
  controlPlane: ControlPlane;
}

export interface IssuedCode {
  code: string;
  expiresInSeconds: number;
}

function fleetStatus(state: FleetState): FleetStatus {
  switch (state) {
    case FleetState.EMPTY:
      return "empty";
    case FleetState.UNENROLLED:
      return "unenrolled";
    case FleetState.UNREACHABLE:
      return "unreachable";
    default:
      return "ready";
  }
}

function effects(values: string[]): GrantEffect[] {
  const allowed: GrantEffect[] = ["read", "write", "destructive"];
  return allowed.filter((effect) => values.includes(effect));
}

function decodeGrant(grant: MachineMessage["grant"]): Grant {
  return {
    summary: grant?.summary ?? "",
    effects: effects(grant?.effects ?? []),
    appCount: grant?.appCount ?? 0,
    coversAllApps: grant?.coversAllApps ?? false,
    scopes: grant?.scopes ?? [],
    preset: grant?.preset ?? "",
  };
}

export function decodeMachine(machine: MachineMessage): Machine {
  return {
    target: machine.target ? decodeTarget(machine.target) : { id: "", kind: "bridge-node", label: "", available: false },
    grant: decodeGrant(machine.grant),
    drift: (machine.drift ?? []).map((item) => ({ kind: item.kind, name: item.name, reason: item.reason })),
    heartbeatAgeSeconds: Number(machine.heartbeatAgeSeconds),
    manageable: machine.manageable,
  };
}

function decodeJoinRequest(request: JoinRequestMessage): JoinRequest {
  return {
    id: request.id,
    name: request.name,
    os: request.os,
    arch: request.arch,
    endpoint: request.endpoint,
    confirmationWords: request.confirmationWords,
    keyFingerprint: request.keyFingerprint,
    requestedAgeSeconds: Number(request.requestedAgeSeconds),
  };
}

function decodePreset(preset: PresetMessage): PermissionPreset {
  return {
    name: preset.name,
    title: preset.title || preset.name,
    description: preset.description,
    scopes: preset.scopes,
    withholds: preset.withholds,
    summary: preset.summary,
    effects: effects(preset.effects),
    appCount: preset.appCount,
  };
}

export async function listFleet(): Promise<Fleet> {
  const response = await machineClient.list({});
  return {
    status: fleetStatus(response.state),
    machines: response.machines.map(decodeMachine),
    joinRequests: response.joinRequests.map(decodeJoinRequest),
    presets: response.presets.map(decodePreset),
    message: response.message,
    recoveryAction: response.recoveryAction,
    controlPlane: {
      reachable: response.controlPlane?.reachable ?? false,
      endpoint: response.controlPlane?.endpoint ?? "",
      detail: response.controlPlane?.detail ?? "",
      consoleUrl: response.controlPlane?.consoleUrl ?? "",
    },
  };
}

export async function issueJoinCode(label = ""): Promise<IssuedCode> {
  const response = await machineClient.issueCode({ label });
  return { code: response.code, expiresInSeconds: Number(response.expiresInSeconds) };
}

export interface DecideInput {
  requestId: string;
  approve: boolean;
  confirmationWords: string[];
  preset: string;
}

export async function decideJoinRequest(input: DecideInput): Promise<string> {
  const response = await machineClient.decide(input);
  return response.message;
}

export async function setMachineGrant(machineId: string, preset: string): Promise<Machine | null> {
  const response = await machineClient.setGrant({ machineId, preset });
  return response.machine ? decodeMachine(response.machine) : null;
}

export async function forgetMachine(machineId: string): Promise<void> {
  await machineClient.forget({ machineId });
}
