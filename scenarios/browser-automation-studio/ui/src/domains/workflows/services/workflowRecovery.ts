import { newGatewayRootOperationID } from '@/api/ai';

type OperationKind = 'create' | 'modify';

interface PendingOperation {
  fingerprint: string;
  id: string;
}

const pendingByKind: Partial<Record<OperationKind, PendingOperation>> = {};

const stableValue = (value: unknown): unknown => {
  if (Array.isArray(value)) return value.map(stableValue);
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, item]) => [key, stableValue(item)]),
    );
  }
  if (typeof value === 'bigint') return value.toString();
  return value;
};

/**
 * Keep the same operation ID across uncertain retries of an identical request.
 * A changed payload immediately gets a fresh ID; success retires the ID.
 */
export const beginWorkflowRecoveryOperation = (
  kind: OperationKind,
  payload: unknown,
): string => {
  const fingerprint = JSON.stringify(stableValue(payload));
  const pending = pendingByKind[kind];
  if (pending?.fingerprint === fingerprint) return pending.id;

  const id = newGatewayRootOperationID();
  pendingByKind[kind] = { fingerprint, id };
  return id;
};

export const completeWorkflowRecoveryOperation = (
  kind: OperationKind,
  operationID: string,
): void => {
  if (pendingByKind[kind]?.id === operationID) delete pendingByKind[kind];
};

/** Test seam to keep cases independent. */
export const resetWorkflowRecoveryOperations = (): void => {
  delete pendingByKind.create;
  delete pendingByKind.modify;
};
