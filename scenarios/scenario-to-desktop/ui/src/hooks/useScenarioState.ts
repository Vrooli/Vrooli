/**
 * Hook for managing server-side scenario state persistence.
 * Replaces localStorage-based draft storage with server-side persistence.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchScenarioState,
  saveScenarioState,
  deleteScenarioState,
  checkStateStaleness,
  type FormState,
  type ScenarioState,
  type SaveStateResponse,
  type InputFingerprint,
  type StateChange,
  type ValidationStatus,
  type BuildArtifact,
} from "../lib/api";

const SAVE_DEBOUNCE_MS = 600;
const STALENESS_CHECK_INTERVAL_MS = 30000;

export interface UseScenarioStateOptions {
  scenarioName: string;
  enabled?: boolean;
  onStateLoaded?: (state: ScenarioState) => void;
  onStateCleared?: () => void;
  onConflict?: (serverState: ScenarioState) => void;
  onSaveError?: (error: Error) => void;
  onManifestChanged?: (currentHash: string, storedHash: string) => void;
}

export interface UseScenarioStateResult {
  // State data
  state: ScenarioState | null;
  formState: FormState | null;
  isLoading: boolean;
  isError: boolean;
  error: Error | null;

  // Save operations
  isSaving: boolean;
  saveError: Error | null;
  lastSavedAt: string | null;

  // Staleness detection
  isStale: boolean;
  pendingChanges: StateChange[];
  validationStatus: ValidationStatus | null;

  // Hashes for conflict detection
  serverHash: string | null;
  localHash: string | null;

  // Actions
  updateFormState: (updates: Partial<FormState>) => void;
  saveNow: () => Promise<void>;
  clearState: () => Promise<void>;
  refetch: () => void;
  resolveConflict: (resolution: "local" | "server") => void;
  checkStaleness: (config: InputFingerprint) => Promise<void>;

  // Timestamps
  timestamps: { createdAt: string; updatedAt: string } | null;

  // Build artifacts
  buildArtifacts: BuildArtifact[];
}

export function useScenarioState({
  scenarioName,
  enabled = true,
  onStateLoaded,
  onStateCleared,
  onConflict,
  onSaveError,
  onManifestChanged,
}: UseScenarioStateOptions): UseScenarioStateResult {
  const queryClient = useQueryClient();
  const saveTimeoutRef = useRef<number | null>(null);
  const pendingUpdatesRef = useRef<Partial<FormState>>({});
  const skipNextSaveRef = useRef(false);

  const [localFormState, setLocalFormState] = useState<FormState | null>(null);
  const [localHash, setLocalHash] = useState<string | null>(null);
  const [lastSavedAt, setLastSavedAt] = useState<string | null>(null);
  const [isStale, setIsStale] = useState(false);
  const [pendingChanges, setPendingChanges] = useState<StateChange[]>([]);
  const [validationStatus, setValidationStatus] = useState<ValidationStatus | null>(null);
  const [conflictState, setConflictState] = useState<ScenarioState | null>(null);

  // Query for fetching state
  const {
    data: serverData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["scenario-state", scenarioName],
    queryFn: () =>
      fetchScenarioState(scenarioName, {
        validateManifest: true,
      }),
    enabled: enabled && Boolean(scenarioName),
    staleTime: 60000,
    refetchOnWindowFocus: false,
  });

  // Mutation for saving state
  const saveMutation = useMutation({
    mutationFn: async (updates: Partial<FormState>) => {
      const mergedState: FormState = {
        ...(localFormState || {}),
        ...updates,
      };
      return saveScenarioState(scenarioName, mergedState, {
        computeHash: true,
        expectedHash: localHash || undefined,
      });
    },
    onSuccess: (response: SaveStateResponse) => {
      if (response.conflict && response.server_state) {
        setConflictState(response.server_state);
        onConflict?.(response.server_state);
      } else {
        setLocalHash(response.hash || null);
        setLastSavedAt(response.updated_at);
        setIsStale(false);
        // Update cache
        queryClient.setQueryData(["scenario-state", scenarioName], {
          state: {
            ...serverData?.state,
            form_state: { ...localFormState, ...pendingUpdatesRef.current },
            hash: response.hash,
            updated_at: response.updated_at,
          },
          found: true,
        });
      }
      pendingUpdatesRef.current = {};
    },
    onError: (err: Error) => {
      onSaveError?.(err);
    },
  });

  // Mutation for deleting state
  const deleteMutation = useMutation({
    mutationFn: () => deleteScenarioState(scenarioName),
    onSuccess: () => {
      setLocalFormState(null);
      setLocalHash(null);
      setLastSavedAt(null);
      setIsStale(false);
      setPendingChanges([]);
      setValidationStatus(null);
      queryClient.invalidateQueries({ queryKey: ["scenario-state", scenarioName] });
      onStateCleared?.();
    },
  });

  // Mutation for staleness check
  const stalenessMutation = useMutation({
    mutationFn: (config: InputFingerprint) => checkStateStaleness(scenarioName, config),
    onSuccess: (response) => {
      setIsStale(response.changed);
      setPendingChanges(response.pending_changes || []);
      if (response.status) {
        setValidationStatus(response.status);
      }
    },
  });

  // Initialize local state from server
  useEffect(() => {
    if (serverData?.state) {
      skipNextSaveRef.current = true;
      setLocalFormState(serverData.state.form_state);
      setLocalHash(serverData.state.hash || null);
      setLastSavedAt(serverData.state.updated_at);
      onStateLoaded?.(serverData.state);

      // Check if manifest changed
      if (serverData.manifest_changed && serverData.current_hash && serverData.stored_hash) {
        onManifestChanged?.(serverData.current_hash, serverData.stored_hash);
      }
    } else if (serverData && !serverData.state) {
      setLocalFormState(null);
      setLocalHash(null);
      onStateCleared?.();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverData]);

  // Staleness check interval
  useEffect(() => {
    if (!enabled || !scenarioName || !localHash) return;

    const interval = setInterval(() => {
      // Only check if we have a manifest path
      const manifestPath = localFormState?.bundle_manifest_path;
      if (manifestPath) {
        stalenessMutation.mutate({ manifest_path: manifestPath });
      }
    }, STALENESS_CHECK_INTERVAL_MS);

    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, scenarioName, localHash, localFormState?.bundle_manifest_path]);

  // Debounced save
  const debouncedSave = useCallback(() => {
    if (saveTimeoutRef.current) {
      window.clearTimeout(saveTimeoutRef.current);
    }

    saveTimeoutRef.current = window.setTimeout(() => {
      if (Object.keys(pendingUpdatesRef.current).length > 0) {
        saveMutation.mutate(pendingUpdatesRef.current);
      }
    }, SAVE_DEBOUNCE_MS);
  }, [saveMutation]);

  // Update form state function
  const updateFormState = useCallback(
    (updates: Partial<FormState>) => {
      if (skipNextSaveRef.current) {
        skipNextSaveRef.current = false;
        return;
      }

      setLocalFormState((prev) => (prev ? { ...prev, ...updates } : updates));
      pendingUpdatesRef.current = { ...pendingUpdatesRef.current, ...updates };
      debouncedSave();
    },
    [debouncedSave]
  );

  // Force save now
  const saveNow = useCallback(async () => {
    if (saveTimeoutRef.current) {
      window.clearTimeout(saveTimeoutRef.current);
    }
    if (Object.keys(pendingUpdatesRef.current).length > 0 || localFormState) {
      await saveMutation.mutateAsync(pendingUpdatesRef.current);
    }
  }, [saveMutation, localFormState]);

  // Clear state
  const clearState = useCallback(async () => {
    if (saveTimeoutRef.current) {
      window.clearTimeout(saveTimeoutRef.current);
    }
    pendingUpdatesRef.current = {};
    await deleteMutation.mutateAsync();
  }, [deleteMutation]);

  // Check staleness manually
  const checkStalenessManual = useCallback(
    async (config: InputFingerprint) => {
      await stalenessMutation.mutateAsync(config);
    },
    [stalenessMutation]
  );

  // Resolve conflict
  const resolveConflict = useCallback(
    (resolution: "local" | "server") => {
      if (resolution === "server" && conflictState) {
        skipNextSaveRef.current = true;
        setLocalFormState(conflictState.form_state);
        setLocalHash(conflictState.hash || null);
        onStateLoaded?.(conflictState);
      } else if (resolution === "local" && localFormState) {
        // Force save local state, ignoring hash check
        saveMutation.mutate({ ...localFormState, ...pendingUpdatesRef.current });
      }
      setConflictState(null);
    },
    [conflictState, localFormState, saveMutation, onStateLoaded]
  );

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        window.clearTimeout(saveTimeoutRef.current);
      }
    };
  }, []);

  const timestamps = useMemo(() => {
    if (!serverData?.state) return null;
    return {
      createdAt: serverData.state.created_at,
      updatedAt: serverData.state.updated_at,
    };
  }, [serverData?.state]);

  const buildArtifacts = useMemo(() => {
    return serverData?.state?.build_artifacts || [];
  }, [serverData?.state?.build_artifacts]);

  return {
    state: serverData?.state || null,
    formState: localFormState,
    isLoading,
    isError,
    error: error as Error | null,
    isSaving: saveMutation.isPending,
    saveError: saveMutation.error as Error | null,
    lastSavedAt,
    isStale,
    pendingChanges,
    validationStatus,
    serverHash: serverData?.state?.hash || null,
    localHash,
    updateFormState,
    saveNow,
    clearState,
    refetch,
    resolveConflict,
    checkStaleness: checkStalenessManual,
    timestamps,
    buildArtifacts,
  };
}
