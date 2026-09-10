import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  OperatorStateService,
  type OperatorState as WireOperatorState,
  type GetOperatorStateResponse,
  type PatchOperatorStateResponse,
  type PatchOperatorStateRequest,
} from "@vrooli/proto-types/vrooli-onboarding/v1/operatorstate/operatorstate_pb";

const client = createClient(
  OperatorStateService,
  onboardingTransport(),
);

export interface OperatorState {
  schema?: string;
  version?: string;
  updatedAt?: string;
  trustPosture?: string;
  hostWorkloadPosture?: string;
  capacityPosture?: string;
  accelPreference?: string;
  updateControl?: string;
  operatorId?: string;
  identityProvider?: string;
  sessionMode?: string;
  enrollmentReference?: string;
  enrolledAt?: string;
  core?: { seed: string[]; trustedBase: string[] };
  activeProfile?: unknown;
  capacity?: { transientHeadroomReserveBytes?: bigint };
  scenarios?: Record<string, { enabled?: boolean; autoRestart?: boolean }>;
  resources?: Record<string, { enabled?: boolean; capacity?: unknown }>;
  hostTools?: Record<string, { optedIn?: boolean }>;
  hostSafeguards?: Record<string, { optedIn?: boolean; config?: Record<string, unknown> }>;
  completion?: {
    selectionDigest?: string;
    appliedAt?: string;
    degradedAcknowledgement?: { readinessDigest?: string; acknowledgedAt?: string };
  };
  session?: { step?: number; stepId?: string };
  notifications?: { recipient?: string };
}

export type OperatorStatePatch = Partial<OperatorState>;

export async function fetchOperatorState(target = "local"): Promise<OperatorState> {
  const response = await client.getOperatorState({ target }) as unknown as GetOperatorStateResponse;
  return (response.state ?? {}) as OperatorState;
}

export async function saveOperatorState(patch: OperatorStatePatch, target = "local"): Promise<OperatorState> {
  const request = {
    target,
    state: patch,
    updateMask: { paths: Object.keys(patch) },
  } as unknown as PatchOperatorStateRequest;
  const response = await client.patchOperatorState(request) as unknown as PatchOperatorStateResponse;
  return (response.state ?? {}) as OperatorState;
}

export async function saveOperatorStateAtRevision(
  patch: OperatorStatePatch,
  expectedRevision: string,
  target = "local",
): Promise<OperatorState> {
  const request = {
    target,
    state: patch,
    updateMask: { paths: Object.keys(patch) },
    expectedRevision,
  } as unknown as PatchOperatorStateRequest;
  const response = await client.patchOperatorState(request) as unknown as PatchOperatorStateResponse;
  return (response.state ?? {}) as OperatorState;
}

export type { WireOperatorState };
