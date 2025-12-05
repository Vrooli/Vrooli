import { useState, useMemo, useCallback } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { DeploymentManifestResponse, DeploymentManifestSecret, SetOverridePayload } from "../../lib/api";
import {
  fetchScenarioTierOverrides,
  setScenarioOverride,
  deleteScenarioOverride,
  copyOverridesFromTier
} from "../../lib/api";
import type { FilterMode, ResourceGroup, ManifestSummary, PendingOverrideEdit, OverrideFields } from "./types";
import { secretIdToString } from "./types";
import {
  groupSecretsByResource,
  filterResourceGroups,
  computeManifestSummary,
  createExportManifest
} from "./utils";

interface UseManifestEditorOptions {
  scenario: string;
  tier: string;
  initialManifest: DeploymentManifestResponse;
}

export function useManifestEditor({ scenario, tier, initialManifest }: UseManifestEditorOptions) {
  const queryClient = useQueryClient();

  // UI state
  const [filter, setFilter] = useState<FilterMode>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [expandedResources, setExpandedResources] = useState<Set<string>>(() => {
    // Start with all resources expanded
    return new Set(initialManifest.resources);
  });
  const [selectedSecret, setSelectedSecret] = useState<{ resource: string; key: string } | null>(null);

  // Session-level exclusions (ephemeral)
  const [excludedResources, setExcludedResources] = useState<Set<string>>(new Set());
  const [excludedSecrets, setExcludedSecrets] = useState<Set<string>>(new Set());

  // Pending edits (not yet saved)
  const [pendingOverrides, setPendingOverrides] = useState<Map<string, PendingOverrideEdit>>(new Map());

  // Fetch existing overrides for this scenario/tier
  const overridesQuery = useQuery({
    queryKey: ["scenario-overrides", scenario, tier],
    queryFn: () => fetchScenarioTierOverrides(scenario, tier),
    enabled: !!scenario && !!tier
  });

  // Build set of overridden secrets from server data
  const overriddenSecrets = useMemo(() => {
    const set = new Set<string>();
    if (overridesQuery.data?.overrides) {
      for (const override of overridesQuery.data.overrides) {
        set.add(secretIdToString({ resource: override.resource_name, key: override.secret_key }));
      }
    }
    return set;
  }, [overridesQuery.data]);

  // Group secrets by resource
  const resourceGroups = useMemo(() => {
    return groupSecretsByResource(
      initialManifest.secrets,
      excludedResources,
      excludedSecrets,
      overriddenSecrets
    );
  }, [initialManifest.secrets, excludedResources, excludedSecrets, overriddenSecrets]);

  // Filter resource groups
  const filteredGroups = useMemo(() => {
    return filterResourceGroups(
      resourceGroups,
      filter,
      excludedResources,
      excludedSecrets,
      overriddenSecrets,
      searchQuery
    );
  }, [resourceGroups, filter, excludedResources, excludedSecrets, overriddenSecrets, searchQuery]);

  // Compute summary statistics
  const summary = useMemo(() => {
    return computeManifestSummary(initialManifest, excludedResources, excludedSecrets, overriddenSecrets);
  }, [initialManifest, excludedResources, excludedSecrets, overriddenSecrets]);

  // Get the selected secret object
  const selectedSecretData = useMemo(() => {
    if (!selectedSecret) return null;
    return initialManifest.secrets.find(
      (s) => s.resource_name === selectedSecret.resource && s.secret_key === selectedSecret.key
    ) ?? null;
  }, [selectedSecret, initialManifest.secrets]);

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
    // Auto-expand the resource if collapsed
    setExpandedResources((prev) => {
      if (prev.has(resource)) return prev;
      return new Set([...prev, resource]);
    });
  }, []);

  // Exclusions (session-only)
  const excludeResource = useCallback((resourceName: string) => {
    setExcludedResources((prev) => new Set([...prev, resourceName]));
  }, []);

  const includeResource = useCallback((resourceName: string) => {
    setExcludedResources((prev) => {
      const next = new Set(prev);
      next.delete(resourceName);
      return next;
    });
  }, []);

  const excludeSecret = useCallback((resource: string, key: string) => {
    const id = secretIdToString({ resource, key });
    setExcludedSecrets((prev) => new Set([...prev, id]));
  }, []);

  const includeSecret = useCallback((resource: string, key: string) => {
    const id = secretIdToString({ resource, key });
    setExcludedSecrets((prev) => {
      const next = new Set(prev);
      next.delete(id);
      return next;
    });
  }, []);

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
      if (isExcluded(resource, key)) {
        includeSecret(resource, key);
      } else {
        excludeSecret(resource, key);
      }
    },
    [isExcluded, includeSecret, excludeSecret]
  );

  const toggleResourceExclusion = useCallback(
    (resourceName: string) => {
      if (excludedResources.has(resourceName)) {
        includeResource(resourceName);
      } else {
        excludeResource(resourceName);
      }
    },
    [excludedResources, includeResource, excludeResource]
  );

  // Strategy editing - update pending changes
  const updatePendingChange = useCallback(
    (resource: string, key: string, changes: Partial<OverrideFields>) => {
      const id = secretIdToString({ resource, key });
      setPendingOverrides((prev) => {
        const next = new Map(prev);
        const existing = next.get(id);
        const original = initialManifest.secrets.find(
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
    [initialManifest.secrets]
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
      queryClient.invalidateQueries({ queryKey: ["scenario-overrides", scenario, tier] });
    }
  });

  // Delete override mutation
  const deleteOverrideMutation = useMutation({
    mutationFn: async ({ resource, key }: { resource: string; key: string }) => {
      return deleteScenarioOverride(scenario, tier, resource, key);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scenario-overrides", scenario, tier] });
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
      queryClient.invalidateQueries({ queryKey: ["scenario-overrides", scenario, tier] });
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

      // Clear pending after save
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

      // Clear any pending changes
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

  const hasPendingChanges = useMemo(() => {
    return Array.from(pendingOverrides.values()).some((p) => p.isDirty);
  }, [pendingOverrides]);

  // Export functionality
  const getExportPreview = useCallback(() => {
    return createExportManifest(initialManifest, excludedResources, excludedSecrets);
  }, [initialManifest, excludedResources, excludedSecrets]);

  const exportManifest = useCallback(() => {
    const exportData = getExportPreview();
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

  return {
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
    excludeResource,
    includeResource,
    excludeSecret,
    includeSecret,
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

    // Export
    exportManifest,
    getExportPreview
  };
}

export type UseManifestEditorReturn = ReturnType<typeof useManifestEditor>;
