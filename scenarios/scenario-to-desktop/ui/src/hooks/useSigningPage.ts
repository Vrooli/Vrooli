/**
 * Hook for Signing page state management.
 * Extracts business logic from SigningPage.tsx for testability.
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useEffect, useCallback } from "react";
import {
  fetchSigningConfig,
  saveSigningConfig,
  validateSigningConfig,
  checkSigningReadiness,
  fetchSigningPrerequisites,
  deleteSigningConfig,
  discoverCertificates,
  generateLinuxSigningKey,
  fetchScenarioDesktopStatus,
  type SigningConfig,
  type SigningReadinessResponse,
  type SigningValidationResult,
  type ToolDetectionResult,
  type DiscoveredCertificate,
} from "../lib/api";
import type { ScenarioDesktopStatus, ScenariosResponse } from "../components/scenario-inventory/types";
import { applyCertificateToConfig, type SigningPlatform } from "../services/signing.service";

// ============================================================================
// Types
// ============================================================================

export interface UseSigningPageProps {
  initialScenario?: string;
  onScenarioChange?: (name: string) => void;
}

export interface UseSigningPageReturn {
  // Scenarios
  scenarios: ScenarioDesktopStatus[];
  selectedScenario: string;
  setSelectedScenario: (name: string) => void;

  // Config state
  localConfig: SigningConfig;
  hasUnsavedChanges: boolean;
  configLoading: boolean;
  serverConfig: SigningConfig | null;

  // Readiness
  readinessData: SigningReadinessResponse | undefined;

  // Prerequisites
  prerequisitesData: ToolDetectionResult[];
  prerequisitesLoading: boolean;
  refetchPrerequisites: () => void;

  // Certificate discovery
  discoverPlatform: SigningPlatform;
  setDiscoverPlatform: (platform: SigningPlatform) => void;
  discovered: DiscoveredCertificate[];
  discoverPending: boolean;
  onDiscover: () => void;

  // Key generation
  keygenMessage: string | undefined;
  keygenPending: boolean;

  // Validation
  validationResult: SigningValidationResult | undefined;
  validatePending: boolean;

  // Mutations
  savePending: boolean;
  saveError: Error | null;
  deletePending: boolean;
  deleteError: Error | null;
  validateError: Error | null;

  // Actions
  handleConfigChange: (updates: Partial<SigningConfig>) => void;
  handleSave: () => void;
  handleValidate: () => void;
  handleDelete: () => void;
  handleGenerateKey: () => Promise<void>;
  applyCertificate: (cert: DiscoveredCertificate) => void;
  refetchConfig: () => void;
  refetchReadiness: () => void;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useSigningPage(props: UseSigningPageProps): UseSigningPageReturn {
  const { initialScenario, onScenarioChange } = props;
  const queryClient = useQueryClient();

  // ========== Local State ==========
  const [selectedScenario, setSelectedScenarioInternal] = useState<string>("");
  const [localConfig, setLocalConfig] = useState<SigningConfig>({ enabled: false });
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [discoverPlatform, setDiscoverPlatform] = useState<SigningPlatform>("windows");
  const [discovered, setDiscovered] = useState<DiscoveredCertificate[]>([]);
  const [keygenMessage, setKeygenMessage] = useState<string | undefined>();

  // ========== Queries ==========

  const { data: scenariosData } = useQuery<ScenariosResponse>({
    queryKey: ["scenarios-desktop-status"],
    queryFn: fetchScenarioDesktopStatus,
    refetchInterval: 30000,
  });

  const {
    data: configData,
    isLoading: configLoading,
    refetch: refetchConfig,
  } = useQuery({
    queryKey: ["signing-config", selectedScenario],
    queryFn: () => fetchSigningConfig(selectedScenario),
    enabled: !!selectedScenario,
  });

  const { data: readinessData, refetch: refetchReadiness } = useQuery<SigningReadinessResponse>({
    queryKey: ["signing-readiness", selectedScenario],
    queryFn: () => checkSigningReadiness(selectedScenario),
    enabled: !!selectedScenario,
  });

  const {
    data: prerequisitesData,
    refetch: refetchPrerequisites,
    isFetching: prerequisitesLoading,
  } = useQuery<{ tools: ToolDetectionResult[] }>({
    queryKey: ["signing-prerequisites"],
    queryFn: fetchSigningPrerequisites,
  });

  // ========== Mutations ==========

  const validateMutation = useMutation({
    mutationFn: () => validateSigningConfig(selectedScenario),
  });

  const generateKeyMutation = useMutation({
    mutationFn: (payload: Parameters<typeof generateLinuxSigningKey>[1]) =>
      generateLinuxSigningKey(selectedScenario, payload),
    onSuccess: (resp) => {
      setHasUnsavedChanges(false);
      setKeygenMessage(`Generated key ${resp.fingerprint} in ${resp.homedir}`);
      queryClient.invalidateQueries({ queryKey: ["signing-config", selectedScenario] });
      queryClient.invalidateQueries({ queryKey: ["signing-readiness", selectedScenario] });
    },
    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : String(err);
      setKeygenMessage(message);
    },
  });

  const saveMutation = useMutation({
    mutationFn: (config: SigningConfig) => saveSigningConfig(selectedScenario, config),
    onSuccess: () => {
      setHasUnsavedChanges(false);
      queryClient.invalidateQueries({ queryKey: ["signing-config", selectedScenario] });
      queryClient.invalidateQueries({ queryKey: ["signing-readiness", selectedScenario] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteSigningConfig(selectedScenario),
    onSuccess: () => {
      setLocalConfig({ enabled: false });
      setHasUnsavedChanges(false);
      queryClient.invalidateQueries({ queryKey: ["signing-config", selectedScenario] });
      queryClient.invalidateQueries({ queryKey: ["signing-readiness", selectedScenario] });
    },
  });

  const discoverMutation = useMutation({
    mutationFn: () => discoverCertificates(discoverPlatform),
    onSuccess: (resp) => {
      setDiscovered(resp.certificates || []);
      if (typeof window !== "undefined") {
        const soonest = (resp.certificates || []).find(
          (c) => !c.is_expired && typeof c.days_to_expiry === "number"
        );
        if (soonest) {
          const warning = `Signing certificate expires in ${soonest.days_to_expiry} days (${soonest.expires_at || "date unknown"}).`;
          window.localStorage.setItem("std_signing_expiry_warning", warning);
        }
      }
    },
  });

  // ========== Effects ==========

  // Sync local state when config data changes
  useEffect(() => {
    if (configData?.config) {
      setLocalConfig(configData.config);
      setHasUnsavedChanges(false);
    } else if (configData && !configData.config) {
      setLocalConfig({ enabled: false });
      setHasUnsavedChanges(false);
    }
  }, [configData]);

  // Set initial scenario
  useEffect(() => {
    if (initialScenario) {
      setSelectedScenarioInternal(initialScenario);
    }
  }, [initialScenario]);

  // Clear keygen message when scenario changes
  useEffect(() => {
    setKeygenMessage(undefined);
  }, [selectedScenario]);

  // ========== Handlers ==========

  const setSelectedScenario = useCallback(
    (name: string) => {
      setSelectedScenarioInternal(name);
      onScenarioChange?.(name);
    },
    [onScenarioChange]
  );

  const handleConfigChange = useCallback((updates: Partial<SigningConfig>) => {
    setLocalConfig((prev) => ({ ...prev, ...updates }));
    setHasUnsavedChanges(true);
  }, []);

  const handleGenerateKey = useCallback(async () => {
    if (!selectedScenario) return;
    const name =
      typeof window !== "undefined" ? window.prompt("Name for GPG UID (required)", selectedScenario) : "";
    if (name === null) return;
    const email = typeof window !== "undefined" ? window.prompt("Email for GPG UID (optional)", "") : "";
    if (email === null) return;
    const passphrase =
      typeof window !== "undefined" ? window.prompt("Passphrase (optional, leave blank for none)", "") : "";

    try {
      await generateKeyMutation.mutateAsync({
        name: name || undefined,
        email: email || undefined,
        passphrase: passphrase || undefined,
        passphrase_env: "GPG_PASSPHRASE",
        expiry: "1y",
      });
      await refetchConfig();
      await refetchReadiness();
    } catch (err) {
      console.error(err);
    }
  }, [selectedScenario, generateKeyMutation, refetchConfig, refetchReadiness]);

  const applyCertificate = useCallback(
    (cert: DiscoveredCertificate) => {
      setHasUnsavedChanges(true);
      setLocalConfig((prev) => applyCertificateToConfig(discoverPlatform, cert, prev));
    },
    [discoverPlatform]
  );

  const handleSave = useCallback(() => {
    saveMutation.mutate(localConfig);
  }, [saveMutation, localConfig]);

  const handleValidate = useCallback(() => {
    validateMutation.mutate();
  }, [validateMutation]);

  const handleDelete = useCallback(() => {
    if (confirm("Are you sure you want to delete this signing configuration?")) {
      deleteMutation.mutate();
    }
  }, [deleteMutation]);

  // ========== Return ==========

  return {
    // Scenarios
    scenarios: scenariosData?.scenarios || [],
    selectedScenario,
    setSelectedScenario,

    // Config state
    localConfig,
    hasUnsavedChanges,
    configLoading,
    serverConfig: configData?.config ?? null,

    // Readiness
    readinessData,

    // Prerequisites
    prerequisitesData: prerequisitesData?.tools || [],
    prerequisitesLoading,
    refetchPrerequisites,

    // Certificate discovery
    discoverPlatform,
    setDiscoverPlatform,
    discovered,
    discoverPending: discoverMutation.isPending,
    onDiscover: () => discoverMutation.mutate(),

    // Key generation
    keygenMessage,
    keygenPending: generateKeyMutation.isPending,

    // Validation
    validationResult: validateMutation.data,
    validatePending: validateMutation.isPending,

    // Mutations
    savePending: saveMutation.isPending,
    saveError: saveMutation.error as Error | null,
    deletePending: deleteMutation.isPending,
    deleteError: deleteMutation.error as Error | null,
    validateError: validateMutation.error as Error | null,

    // Actions
    handleConfigChange,
    handleSave,
    handleValidate,
    handleDelete,
    handleGenerateKey,
    applyCertificate,
    refetchConfig,
    refetchReadiness,
  };
}
