import { useState, useCallback, useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  generateDeploymentManifest,
  fetchScenarioTierOverrides,
  setScenarioOverride,
  deleteScenarioOverride,
  copyOverridesFromTier,
  type DeploymentManifestResponse,
  type DeploymentManifestRequest,
  type SetOverridePayload
} from "../../lib/api";
import type { FilterMode, PendingOverrideEdit, OverrideFields } from "./types";
import { secretIdToString } from "./types";
import {
  groupSecretsByResource,
  filterResourceGroups,
  computeManifestSummary,
  createExportManifest
} from "./utils";

const DEFAULT_TIERS = [
  { value: "tier-1-local", label: "Tier 1 - Local" },
  { value: "tier-2-desktop", label: "Tier 2 - Desktop" },
  { value: "tier-3-mobile", label: "Tier 3 - Mobile" },
  { value: "tier-4-saas", label: "Tier 4 - SaaS" },
  { value: "tier-5-enterprise", label: "Tier 5 - Enterprise" }
];

interface UseManifestWorkspaceOptions {
  initialScenario?: string;
  initialTier?: string;
  availableScenarios?: Array<{ name: string; display_name?: string }>;
  onScenarioChange?: (scenario: string) => void;
}

export function useManifestWorkspace({
  initialScenario = "",
  initialTier = "tier-2-desktop",
  availableScenarios = [],
  onScenarioChange
}: UseManifestWorkspaceOptions = {}) {
  const queryClient = useQueryClient();

  // Core selection state
  const [scenario, setScenarioInternal] = useState(initialScenario);
  const [tier, setTierInternal] = useState(initialTier);

  // Debounced scenario for API calls
  const [debouncedScenario, setDebouncedScenario] = useState(scenario);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // UI state
  const [filter, setFilter] = useState<FilterMode>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [expandedResources, setExpandedResources] = useState<Set<string>>(new Set());
  const [selectedSecret, setSelectedSecret] = useState<{ resource: string; key: string } | null>(null);

  // Session-level exclusions (ephemeral)
  const [excludedResources, setExcludedResources] = useState<Set<string>>(new Set());
  const [excludedSecrets, setExcludedSecrets] = useState<Set<string>>(new Set());

  // Pending edits (not yet saved)
  const [pendingOverrides, setPendingOverrides] = useState<Map<string, PendingOverrideEdit>>(new Map());

  // Confirm dialog state
  const [confirmDialog, setConfirmDialog] = useState<{
    open: boolean;
    title: string;
    message: string;
    onConfirm: () => void;
  } | null>(null);

  // JSON panel visibility
  const [jsonPanelOpen, setJsonPanelOpen] = useState(true);

  // Sync external scenario changes
  useEffect(() => {
    if (initialScenario && initialScenario !== scenario) {
      setScenarioInternal(initialScenario);
    }
  }, [initialScenario]);

  // Debounce scenario input to prevent excessive API calls
  useEffect(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      setDebouncedScenario(scenario);
    }, 400);
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [scenario]);

  // Manifest query - auto-fetches when scenario/tier changes
  const manifestQuery = useQuery({
    queryKey: ["deployment-manifest", debouncedScenario, tier],
    queryFn: () =>
      generateDeploymentManifest({
        scenario: debouncedScenario,
        tier,
        include_optional: false
      }),
    enabled: !!debouncedScenario && !!tier,
    staleTime: 30000 // 30 seconds
  });

  const manifest = manifestQuery.data;

  // Fetch existing overrides for this scenario/tier
  const overridesQuery = useQuery({
    queryKey: ["scenario-overrides", debouncedScenario, tier],
    queryFn: () => fetchScenarioTierOverrides(debouncedScenario, tier),
    enabled: !!debouncedScenario && !!tier
  });

  // Build set of overridden secrets from server data
  const overriddenSecrets = new Set<string>();
  if (overridesQuery.data?.overrides) {
    for (const override of overridesQuery.data.overrides) {
      overriddenSecrets.add(secretIdToString({ resource: override.resource_name, key: override.secret_key }));
    }
  }

  // Auto-expand resources when manifest loads
  useEffect(() => {
    if (manifest?.resources) {
      setExpandedResources(new Set(manifest.resources));
    }
  }, [manifest?.resources]);

  // Group secrets by resource
  const resourceGroups = manifest?.secrets
    ? groupSecretsByResource(manifest.secrets, excludedResources, excludedSecrets, overriddenSecrets)
    : [];

  // Filter resource groups
  const filteredGroups = filterResourceGroups(
    resourceGroups,
    filter,
    excludedResources,
    excludedSecrets,
    overriddenSecrets,
    searchQuery
  );

  // Compute summary statistics
  const summary = manifest
    ? computeManifestSummary(manifest, excludedResources, excludedSecrets, overriddenSecrets)
    : { totalSecrets: 0, strategizedSecrets: 0, blockingSecrets: 0, excludedSecrets: 0, overriddenSecrets: 0, resourceCount: 0 };

  // Get the selected secret object
  const selectedSecretData = selectedSecret && manifest?.secrets
    ? manifest.secrets.find(
        (s) => s.resource_name === selectedSecret.resource && s.secret_key === selectedSecret.key
      ) ?? null
    : null;

  // Has pending changes
  const hasPendingChanges = Array.from(pendingOverrides.values()).some((p) => p.isDirty);

  // Clear session state when scenario/tier changes
  const clearSessionState = useCallback(() => {
    setExcludedResources(new Set());
    setExcludedSecrets(new Set());
    setPendingOverrides(new Map());
    setSelectedSecret(null);
    setSearchQuery("");
    setFilter("all");
  }, []);

  // Scenario change with dirty check
  const setScenario = useCallback(
    (newScenario: string) => {
      if (newScenario === scenario) return;

      const performChange = () => {
        clearSessionState();
        setScenarioInternal(newScenario);
        onScenarioChange?.(newScenario);
      };

      if (hasPendingChanges) {
        setConfirmDialog({
          open: true,
          title: "Unsaved changes",
          message: `You have unsaved changes. Switching to "${newScenario}" will discard them. Continue?`,
          onConfirm: () => {
            performChange();
            setConfirmDialog(null);
          }
        });
      } else {
        performChange();
      }
    },
    [scenario, hasPendingChanges, clearSessionState, onScenarioChange]
  );

  // Tier change with dirty check
  const setTier = useCallback(
    (newTier: string) => {
      if (newTier === tier) return;

      const performChange = () => {
        clearSessionState();
        setTierInternal(newTier);
      };

      if (hasPendingChanges) {
        setConfirmDialog({
          open: true,
          title: "Unsaved changes",
          message: `You have unsaved changes. Switching to "${newTier}" will discard them. Continue?`,
          onConfirm: () => {
            performChange();
            setConfirmDialog(null);
          }
        });
      } else {
        performChange();
      }
    },
    [tier, hasPendingChanges, clearSessionState]
  );

  // Close confirm dialog
  const closeConfirmDialog = useCallback(() => {
    setConfirmDialog(null);
  }, []);

  // Check if secret is overridden
  const isSecretOverridden = useCallback(
    (resource: string, key: string) => {
      return overriddenSecrets.has(secretIdToString({ resource, key }));
    },
    [overriddenSecrets]
  );

  // Tree navigation
  const toggleResource = useCallback((resourceName: string) => {
    setExpandedResources((prev) => {
      const next = new Set(prev);
      if (next.has(resourceName)) {
        next.delete(resourceName);
      } else {
        next.add(resourceName);
      }
      return next;
    });
  }, []);

  const selectSecret = useCallback((resource: string, key: string) => {
    setSelectedSecret({ resource, key });
    setExpandedResources((prev) => {
      if (prev.has(resource)) return prev;
      return new Set([...prev, resource]);
    });
  }, []);

  // Exclusions (session-only)
  const isExcluded = useCallback(
    (resource: string, key?: string): boolean => {
      if (excludedResources.has(resource)) return true;
      if (key) {
        return excludedSecrets.has(secretIdToString({ resource, key }));
      }
      return false;
    },
    [excludedResources, excludedSecrets]
  );

  const toggleSecretExclusion = useCallback(
    (resource: string, key: string) => {
      const id = secretIdToString({ resource, key });
      setExcludedSecrets((prev) => {
        const next = new Set(prev);
        if (next.has(id)) {
          next.delete(id);
        } else {
          next.add(id);
        }
        return next;
      });
    },
    []
  );

  const toggleResourceExclusion = useCallback((resourceName: string) => {
    setExcludedResources((prev) => {
      const next = new Set(prev);
      if (next.has(resourceName)) {
        next.delete(resourceName);
      } else {
        next.add(resourceName);
      }
      return next;
    });
  }, []);

  // Strategy editing - update pending changes
  const updatePendingChange = useCallback(
    (resource: string, key: string, changes: Partial<OverrideFields>) => {
      if (!manifest?.secrets) return;

      const id = secretIdToString({ resource, key });
      setPendingOverrides((prev) => {
        const next = new Map(prev);
        const existing = next.get(id);
        const original = manifest.secrets.find(
          (s) => s.resource_name === resource && s.secret_key === key
        );
        if (!original) return prev;

        next.set(id, {
          original,
          changes: { ...(existing?.changes ?? {}), ...changes },
          isDirty: true
        });
        return next;
      });
    },
    [manifest?.secrets]
  );

  // Save override mutation
  const saveOverrideMutation = useMutation({
    mutationFn: async ({
      resource,
      key,
      payload
    }: {
      resource: string;
      key: string;
      payload: SetOverridePayload;
    }) => {
      return setScenarioOverride(scenario, tier, resource, key, payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scenario-overrides", debouncedScenario, tier] });
      queryClient.invalidateQueries({ queryKey: ["deployment-manifest", debouncedScenario, tier] });
    }
  });

  // Delete override mutation
  const deleteOverrideMutation = useMutation({
    mutationFn: async ({ resource, key }: { resource: string; key: string }) => {
      return deleteScenarioOverride(scenario, tier, resource, key);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scenario-overrides", debouncedScenario, tier] });
      queryClient.invalidateQueries({ queryKey: ["deployment-manifest", debouncedScenario, tier] });
    }
  });

  // Copy from tier mutation
  const copyFromTierMutation = useMutation({
    mutationFn: async ({ sourceTier, overwrite }: { sourceTier: string; overwrite?: boolean }) => {
      return copyOverridesFromTier(scenario, {
        source_tier: sourceTier,
        target_tier: tier,
        overwrite
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scenario-overrides", debouncedScenario, tier] });
      queryClient.invalidateQueries({ queryKey: ["deployment-manifest", debouncedScenario, tier] });
    }
  });

  const saveOverride = useCallback(
    async (resource: string, key: string) => {
      const id = secretIdToString({ resource, key });
      const pending = pendingOverrides.get(id);
      if (!pending?.isDirty) return;

      const payload: SetOverridePayload = {
        handling_strategy: pending.changes.handling_strategy,
        fallback_strategy: pending.changes.fallback_strategy,
        requires_user_input: pending.changes.requires_user_input,
        prompt_label: pending.changes.prompt_label,
        prompt_description: pending.changes.prompt_description,
        generator_template: pending.changes.generator_template,
        bundle_hints: pending.changes.bundle_hints
      };

      await saveOverrideMutation.mutateAsync({ resource, key, payload });

      setPendingOverrides((prev) => {
        const next = new Map(prev);
        next.delete(id);
        return next;
      });
    },
    [pendingOverrides, saveOverrideMutation]
  );

  const resetOverride = useCallback(
    async (resource: string, key: string) => {
      await deleteOverrideMutation.mutateAsync({ resource, key });

      const id = secretIdToString({ resource, key });
      setPendingOverrides((prev) => {
        const next = new Map(prev);
        next.delete(id);
        return next;
      });
    },
    [deleteOverrideMutation]
  );

  const saveAllPending = useCallback(async () => {
    const entries = Array.from(pendingOverrides.entries());
    for (const [id, pending] of entries) {
      if (!pending.isDirty) continue;
      const [resource, key] = id.split(":");
      if (resource && key) {
        await saveOverride(resource, key);
      }
    }
  }, [pendingOverrides, saveOverride]);

  // Export functionality
  const getExportPreview = useCallback(() => {
    if (!manifest) return null;
    return createExportManifest(manifest, excludedResources, excludedSecrets);
  }, [manifest, excludedResources, excludedSecrets]);

  const exportManifest = useCallback(() => {
    const exportData = getExportPreview();
    if (!exportData) return;

    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${scenario}-manifest-${tier}-custom.json`;
    link.click();
    URL.revokeObjectURL(url);
  }, [getExportPreview, scenario, tier]);

  const copyFromTier = useCallback(
    async (sourceTier: string, overwrite?: boolean) => {
      await copyFromTierMutation.mutateAsync({ sourceTier, overwrite });
    },
    [copyFromTierMutation]
  );

  const refreshManifest = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["deployment-manifest", debouncedScenario, tier] });
  }, [queryClient, debouncedScenario, tier]);

  return {
    // Selection state
    scenario,
    tier,
    setScenario,
    setTier,
    availableTiers: DEFAULT_TIERS,
    availableScenarios,

    // Manifest data
    manifest,
    manifestIsLoading: manifestQuery.isLoading,
    manifestIsError: manifestQuery.isError,
    manifestError: manifestQuery.error,
    refreshManifest,

    // Derived data
    resources: filteredGroups,
    allResources: resourceGroups,
    selectedSecret: selectedSecretData,
    summary,

    // Filters & search
    filter,
    setFilter,
    searchQuery,
    setSearchQuery,

    // Tree navigation
    expandedResources,
    toggleResource,
    selectSecret,
    selectedSecretId: selectedSecret,

    // Exclusions (session-only)
    isExcluded,
    toggleSecretExclusion,
    toggleResourceExclusion,
    excludedResources,
    excludedSecrets,

    // Override status
    overriddenSecrets,
    isSecretOverridden,
    overridesQuery,

    // Strategy editing
    pendingOverrides,
    updatePendingChange,
    saveOverride,
    resetOverride,
    saveAllPending,
    hasPendingChanges,
    isSaving: saveOverrideMutation.isPending,
    isDeleting: deleteOverrideMutation.isPending,

    // Copy from tier
    copyFromTier,
    isCopying: copyFromTierMutation.isPending,
    copyError: copyFromTierMutation.error?.message ?? null,

    // Export
    exportManifest,
    getExportPreview,

    // JSON panel
    jsonPanelOpen,
    setJsonPanelOpen,

    // Confirm dialog
    confirmDialog,
    closeConfirmDialog
  };
}

export type UseManifestWorkspaceReturn = ReturnType<typeof useManifestWorkspace>;
