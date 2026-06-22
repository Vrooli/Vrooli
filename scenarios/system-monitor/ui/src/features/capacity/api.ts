import { protoFetch } from '../../shared/api/apiFetch';
import {
  parseGetCapacityOverviewResponse,
  parseReconcileCapacityResponse,
  parseGetCapacityPolicyResponse,
  parseSetCapacityPolicyResponse,
} from '../../shared/api/proto-contracts';
import type {
  CapacityOverview,
  CapacityReconciliation,
  PolicyLever,
} from './types';

/** Fetch the per-GPU contention picture plus the active claim table. */
export async function fetchCapacityOverview(signal?: AbortSignal): Promise<CapacityOverview> {
  return protoFetch('/api/v1/capacity/overview', parseGetCapacityOverviewResponse, { signal });
}

/** Fetch the reconciliation findings (unclaimed-consumer warnings). */
export async function fetchCapacityReconciliation(signal?: AbortSignal): Promise<CapacityReconciliation> {
  return protoFetch('/api/v1/capacity/reconcile', parseReconcileCapacityResponse, { signal });
}

/** Fetch the tunable policy levers. */
export async function fetchCapacityPolicy(signal?: AbortSignal): Promise<PolicyLever[]> {
  const resp = await protoFetch('/api/v1/capacity/policy', parseGetCapacityPolicyResponse, { signal });
  return resp.levers;
}

/** Persist a single policy lever and return the full updated lever set. */
export async function setCapacityPolicy(key: string, value: string, signal?: AbortSignal): Promise<PolicyLever[]> {
  const resp = await protoFetch('/api/v1/capacity/policy', parseSetCapacityPolicyResponse, {
    method: 'POST',
    body: JSON.stringify({ key, value }),
    signal,
  });
  return resp.levers;
}
