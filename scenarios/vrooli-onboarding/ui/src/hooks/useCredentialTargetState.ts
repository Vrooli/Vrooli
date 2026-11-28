import { useCallback, useSyncExternalStore } from "react";

export type ProvisionError = {
  kind: "operator" | "generic";
  logicalID: string;
  field: string;
  recoveryURL?: string;
  authSource?: string;
  authProviders?: string[];
};

export interface CredentialTargetState {
  values: Record<string, string>;
  questionSecrets: Record<string, string>;
  provisioning: string | null;
  provisionedCredential: string | null;
  provisionError: ProvisionError | null;
}

const emptyState: CredentialTargetState = {
  values: {},
  questionSecrets: {},
  provisioning: null,
  provisionedCredential: null,
  provisionError: null,
};

const targetStates = new Map<string, CredentialTargetState>();
const targetListeners = new Map<string, Set<() => void>>();

export function resetCredentialTargetState(target: string): void {
  targetStates.delete(target);
  targetListeners.get(target)?.forEach((listener) => listener());
}

function stateForTarget(target: string): CredentialTargetState {
  return targetStates.get(target) ?? emptyState;
}

function subscribeToTarget(target: string, listener: () => void) {
  const listeners = targetListeners.get(target) ?? new Set<() => void>();
  listeners.add(listener);
  targetListeners.set(target, listeners);
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) targetListeners.delete(target);
  };
}

/**
 * Keeps unsaved credential and secret-answer input scoped to its target for
 * this tab's lifetime. The module store survives route unmounts but never
 * writes secret text to browser storage; a reload clears it and requires
 * re-entry.
 */
export function useCredentialTargetState(target: string) {
  const updateTargetState = useCallback((targetId: string, update: (state: CredentialTargetState) => CredentialTargetState) => {
    targetStates.set(targetId, update(stateForTarget(targetId)));
    targetListeners.get(targetId)?.forEach((listener) => listener());
  }, []);
  const state = useSyncExternalStore(
    (listener) => subscribeToTarget(target, listener),
    () => stateForTarget(target),
    () => emptyState,
  );
  return {
    state,
    updateTargetState,
  };
}
