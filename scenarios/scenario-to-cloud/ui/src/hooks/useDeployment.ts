import { useState, useMemo, useCallback, useRef, useEffect } from "react";
import {
  validateManifest,
  buildBundle,
  runPreflight as runPreflightApi,
  createDeployment,
  executeDeployment,
  getDeployment,
  findInProgressDeployment,
  fetchSecretsManifest,
  type ValidationIssue,
  type BundleArtifact,
  type PreflightCheck,
  type SSHConnectionStatus,
  type DeploymentStatus,
  type SecretsManifest,
  type ProvidedSecrets,
} from "../lib/api";
import {
  type CustomSecret,
  isValidSecretKey,
  isReservedKey,
} from "../types/secrets";
import {
  type DeploymentManifest,
  type WizardStep,
  DEFAULT_MANIFEST,
  DEFAULT_MANIFEST_JSON,
  WIZARD_STEPS,
} from "../types/deployment";

const MAX_HISTORY_SIZE = 50;

const STORAGE_KEY = "scenario-to-cloud:deployment";

type SavedDeployment = {
  manifestJson: string;
  currentStep: number;
  timestamp: number;
  sshKeyPath?: string | null;
  deploymentId?: string | null;
  deploymentStatus?: "idle" | "deploying" | "success" | "failed" | null;
};

function loadSavedDeployment(): SavedDeployment | null {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      return JSON.parse(saved);
    }
  } catch {
    // Ignore parse errors
  }
  return null;
}

function saveDeployment(data: SavedDeployment): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch {
    // Ignore storage errors
  }
}

function clearSavedDeployment(): void {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Ignore storage errors
  }
}

export function useDeployment() {
  const saved = useMemo(() => loadSavedDeployment(), []);

  // Wizard navigation
  const [currentStepIndex, setCurrentStepIndex] = useState(saved?.currentStep ?? 0);

  // Manifest state
  const [manifestJson, setManifestJson] = useState(saved?.manifestJson ?? DEFAULT_MANIFEST_JSON);

  // Validation state
  const [validationIssues, setValidationIssues] = useState<ValidationIssue[] | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const [normalizedManifest, setNormalizedManifest] = useState<DeploymentManifest | null>(null);

  // Bundle state
  const [bundleArtifact, setBundleArtifactRaw] = useState<BundleArtifact | null>(null);
  const [bundleError, setBundleError] = useState<string | null>(null);
  const [isBuildingBundle, setIsBuildingBundle] = useState(false);

  // Preflight state
  const [preflightPassed, setPreflightPassed] = useState<boolean | null>(null);
  const [preflightChecks, setPreflightChecks] = useState<PreflightCheck[] | null>(null);
  const [preflightError, setPreflightError] = useState<string | null>(null);
  const [isRunningPreflight, setIsRunningPreflight] = useState(false);
  const [preflightOverride, setPreflightOverride] = useState(false);

  // Deploy state - initialize from saved state if available
  const [deploymentStatus, setDeploymentStatus] = useState<"idle" | "deploying" | "success" | "failed">(
    saved?.deploymentStatus ?? "idle"
  );
  const [deploymentError, setDeploymentError] = useState<string | null>(null);
  const [deploymentId, setDeploymentId] = useState<string | null>(saved?.deploymentId ?? null);
  const [isReconnecting, setIsReconnecting] = useState(false);

  // SSH state
  const [sshKeyPath, setSSHKeyPath] = useState<string | null>(saved?.sshKeyPath ?? null);
  const [sshConnectionStatus, setSSHConnectionStatus] = useState<SSHConnectionStatus>("untested");

  // Secrets state
  const [secretsManifest, setSecretsManifest] = useState<SecretsManifest | null>(null);
  const [secretsError, setSecretsError] = useState<string | null>(null);
  const [isFetchingSecrets, setIsFetchingSecrets] = useState(false);
  const [secretsFetched, setSecretsFetched] = useState(false); // Track if fetch was attempted
  const [providedSecrets, setProvidedSecrets] = useState<ProvidedSecrets>({});

  // Custom secrets state (manually added secrets not detected by secrets-manager)
  const [customSecrets, setCustomSecrets] = useState<CustomSecret[]>([]);

  // Undo/redo history
  const historyRef = useRef<string[]>([]);
  const futureRef = useRef<string[]>([]);
  const isUndoRedoRef = useRef(false);

  // Parse manifest
  const parsedManifest = useMemo((): { ok: true; value: DeploymentManifest } | { ok: false; error: string } => {
    try {
      const value = JSON.parse(manifestJson) as DeploymentManifest;
      return { ok: true, value };
    } catch (e) {
      return { ok: false, error: e instanceof Error ? e.message : String(e) };
    }
  }, [manifestJson]);

  const currentStep = WIZARD_STEPS[currentStepIndex];

  // Wrapper to reset deployment state when bundle changes
  const setBundleArtifact = useCallback((artifact: BundleArtifact | null) => {
    setBundleArtifactRaw(artifact);
    // Reset deployment state when a new bundle is selected
    // This ensures "Retry Deployment" uses the new bundle
    if (artifact !== null) {
      setDeploymentStatus("idle");
      setDeploymentError(null);
      setDeploymentId(null);
    }
  }, []);

  // Reconnection logic: Check for in-progress deployment on mount
  useEffect(() => {
    const checkForInProgressDeployment = async () => {
      // Case 1: Same-device refresh - we have a saved deploymentId
      if (saved?.deploymentId && saved?.deploymentStatus === "deploying") {
        setIsReconnecting(true);
        try {
          const res = await getDeployment(saved.deploymentId);
          const status = res.deployment.status;

          if (status === "setup_running" || status === "deploying") {
            // Still in progress - state already initialized from saved, just verify
            setDeploymentId(saved.deploymentId);
            setDeploymentStatus("deploying");
            // Navigate to deploy step if not already there
            if (currentStepIndex !== 3) {
              setCurrentStepIndex(3);
            }
          } else if (status === "deployed") {
            // Completed successfully
            setDeploymentStatus("success");
            setDeploymentId(saved.deploymentId);
          } else if (status === "failed" || status === "stopped") {
            // Failed or stopped
            setDeploymentStatus("failed");
            setDeploymentId(saved.deploymentId);
            setDeploymentError(res.deployment.error_message ?? "Deployment failed");
          }
        } catch {
          // Deployment no longer exists - clear saved state
          clearSavedDeployment();
          setDeploymentId(null);
          setDeploymentStatus("idle");
        } finally {
          setIsReconnecting(false);
        }
        return;
      }

      // Case 2: Cross-device - check if there's an in-progress deployment for this scenario
      if (saved?.manifestJson && !saved?.deploymentId) {
        try {
          const manifest = JSON.parse(saved.manifestJson) as DeploymentManifest;
          const scenarioId = manifest.scenario?.id;
          if (scenarioId) {
            const inProgress = await findInProgressDeployment(scenarioId);
            if (inProgress) {
              // Found an in-progress deployment - resume it
              setDeploymentId(inProgress.id);
              setDeploymentStatus("deploying");
              setCurrentStepIndex(3); // Navigate to deploy step
              // Save the discovered deployment to localStorage
              saveDeployment({
                manifestJson: saved.manifestJson,
                currentStep: 3,
                timestamp: Date.now(),
                sshKeyPath: saved.sshKeyPath,
                deploymentId: inProgress.id,
                deploymentStatus: "deploying",
              });
            }
          }
        } catch {
          // Ignore errors during cross-device detection
        }
      }
    };

    checkForInProgressDeployment();
    // Only run on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Save on manifest/step change (with undo history tracking)
  // Accepts either a string or a functional update (prev => newValue)
  const updateManifestJson = useCallback((jsonOrUpdater: string | ((prev: string) => string)) => {
    // Use functional update to ensure we always have the latest state
    // This is critical for async operations that may complete after other state changes
    setManifestJson(prevManifest => {
      const newJson = typeof jsonOrUpdater === "function" ? jsonOrUpdater(prevManifest) : jsonOrUpdater;

      // Skip history tracking during undo/redo operations
      if (!isUndoRedoRef.current) {
        // Push current state to history before changing
        historyRef.current = [...historyRef.current, prevManifest].slice(-MAX_HISTORY_SIZE);
        // Clear future on new change
        futureRef.current = [];
      }
      isUndoRedoRef.current = false;

      // Save to localStorage (async, doesn't affect return value)
      saveDeployment({
        manifestJson: newJson,
        currentStep: currentStepIndex,
        timestamp: Date.now(),
        sshKeyPath,
        deploymentId,
        deploymentStatus,
      });

      return newJson;
    });
  }, [currentStepIndex, sshKeyPath, deploymentId, deploymentStatus]);

  // Undo last manifest change
  const undo = useCallback(() => {
    if (historyRef.current.length === 0) return;

    const previousState = historyRef.current[historyRef.current.length - 1];
    historyRef.current = historyRef.current.slice(0, -1);
    futureRef.current = [manifestJson, ...futureRef.current];

    isUndoRedoRef.current = true;
    setManifestJson(previousState);
    saveDeployment({
      manifestJson: previousState,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath,
      deploymentId,
      deploymentStatus,
    });
  }, [manifestJson, currentStepIndex, sshKeyPath, deploymentId, deploymentStatus]);

  // Redo last undone change
  const redo = useCallback(() => {
    if (futureRef.current.length === 0) return;

    const nextState = futureRef.current[0];
    futureRef.current = futureRef.current.slice(1);
    historyRef.current = [...historyRef.current, manifestJson];

    isUndoRedoRef.current = true;
    setManifestJson(nextState);
    saveDeployment({
      manifestJson: nextState,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath,
      deploymentId,
      deploymentStatus,
    });
  }, [manifestJson, currentStepIndex, sshKeyPath, deploymentId, deploymentStatus]);

  // Check if undo/redo is available (for UI state)
  const canUndo = historyRef.current.length > 0;
  const canRedo = futureRef.current.length > 0;

  // Navigation
  const goToStep = useCallback((index: number) => {
    if (index >= 0 && index < WIZARD_STEPS.length) {
      setCurrentStepIndex(index);
      saveDeployment({
        manifestJson,
        currentStep: index,
        timestamp: Date.now(),
        sshKeyPath,
        deploymentId,
        deploymentStatus,
      });
    }
  }, [manifestJson, sshKeyPath, deploymentId, deploymentStatus]);

  // Update SSH key path with persistence
  const updateSSHKeyPath = useCallback((keyPath: string | null) => {
    setSSHKeyPath(keyPath);
    setSSHConnectionStatus("untested"); // Reset connection status when key changes
    saveDeployment({
      manifestJson,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath: keyPath,
      deploymentId,
      deploymentStatus,
    });
  }, [manifestJson, currentStepIndex, deploymentId, deploymentStatus]);

  const goNext = useCallback(() => {
    goToStep(currentStepIndex + 1);
  }, [currentStepIndex, goToStep]);

  const goBack = useCallback(() => {
    goToStep(currentStepIndex - 1);
  }, [currentStepIndex, goToStep]);

  // Actions
  const validate = useCallback(async () => {
    setValidationError(null);
    setValidationIssues(null);
    setNormalizedManifest(null);
    setIsValidating(true);

    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await validateManifest(parsedManifest.value);
      setValidationIssues(res.issues ?? []);
      // Store the normalized manifest for auto-fix functionality
      if (res.manifest) {
        setNormalizedManifest(res.manifest as DeploymentManifest);
      }
    } catch (e) {
      setValidationError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsValidating(false);
    }
  }, [parsedManifest]);

  // Apply all auto-fixes by replacing manifest with normalized version
  const applyAllFixes = useCallback(() => {
    if (normalizedManifest) {
      const json = JSON.stringify(normalizedManifest, null, 2);
      setManifestJson(json);
      saveDeployment({
        manifestJson: json,
        currentStep: currentStepIndex,
        timestamp: Date.now(),
        sshKeyPath,
        deploymentId,
        deploymentStatus,
      });
      // Clear validation state to trigger re-validation
      setValidationIssues(null);
      setNormalizedManifest(null);
    }
  }, [normalizedManifest, currentStepIndex, sshKeyPath, deploymentId, deploymentStatus]);

  const build = useCallback(async () => {
    setBundleArtifact(null);
    setBundleError(null);
    setIsBuildingBundle(true);

    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await buildBundle(parsedManifest.value);
      setBundleArtifact(res.artifact);
    } catch (e) {
      setBundleError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsBuildingBundle(false);
    }
  }, [parsedManifest, setBundleArtifact]);

  const runPreflight = useCallback(async () => {
    setPreflightPassed(null);
    setPreflightChecks(null);
    setPreflightError(null);
    setPreflightOverride(false); // Reset override when re-running checks
    setIsRunningPreflight(true);

    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await runPreflightApi(parsedManifest.value);
      setPreflightChecks(res.checks);
      setPreflightPassed(res.ok);
    } catch (e) {
      setPreflightError(e instanceof Error ? e.message : String(e));
      setPreflightPassed(false);
    } finally {
      setIsRunningPreflight(false);
    }
  }, [parsedManifest]);

  // Fetch secrets requirements for the current scenario
  const fetchSecrets = useCallback(async () => {
    if (!parsedManifest.ok) return;
    const scenarioId = parsedManifest.value.scenario?.id;
    if (!scenarioId) {
      setSecretsError("No scenario selected");
      setSecretsFetched(true);
      return;
    }

    setIsFetchingSecrets(true);
    setSecretsError(null);

    try {
      const res = await fetchSecretsManifest(
        scenarioId,
        "tier-4-saas", // Cloud deployments use tier-4
        parsedManifest.value.dependencies?.resources
      );
      setSecretsManifest(res.secrets);
    } catch (e) {
      setSecretsError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsFetchingSecrets(false);
      setSecretsFetched(true);
    }
  }, [parsedManifest]);

  // Update a single provided secret
  const updateProvidedSecret = useCallback((key: string, value: string) => {
    setProvidedSecrets(prev => ({ ...prev, [key]: value }));
  }, []);

  // Custom secrets management
  const addCustomSecret = useCallback(() => {
    const id = `custom-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    setCustomSecrets(prev => [...prev, { id, key: "", value: "", description: "" }]);
  }, []);

  const removeCustomSecret = useCallback((id: string) => {
    setCustomSecrets(prev => prev.filter(s => s.id !== id));
  }, []);

  const updateCustomSecret = useCallback((id: string, field: keyof Omit<CustomSecret, "id">, value: string) => {
    setCustomSecrets(prev => prev.map(s =>
      s.id === id ? { ...s, [field]: value } : s
    ));
  }, []);

  // Validate all custom secrets
  const customSecretsValidation = useMemo(() => {
    const errors: Record<string, string> = {};
    const keys = new Set<string>();

    // Also collect keys from auto-detected secrets
    const autoDetectedKeys = new Set(
      secretsManifest?.bundle_secrets.map(s => s.target.name || s.id) ?? []
    );

    for (const secret of customSecrets) {
      if (!secret.key.trim()) {
        errors[secret.id] = "Key is required";
      } else if (!isValidSecretKey(secret.key)) {
        errors[secret.id] = "Key must be uppercase letters, numbers, and underscores (e.g., MY_API_KEY)";
      } else if (isReservedKey(secret.key)) {
        errors[secret.id] = "This key prefix is reserved";
      } else if (keys.has(secret.key)) {
        errors[secret.id] = "Duplicate key";
      } else if (autoDetectedKeys.has(secret.key)) {
        errors[secret.id] = "This key conflicts with an auto-detected secret";
      } else if (!secret.value.trim()) {
        errors[secret.id] = "Value is required";
      }
      keys.add(secret.key);
    }

    return {
      errors,
      isValid: Object.keys(errors).length === 0,
    };
  }, [customSecrets, secretsManifest]);

  // Merge custom secrets into provided secrets for deployment
  const mergedProvidedSecrets = useMemo((): ProvidedSecrets => {
    const merged = { ...providedSecrets };
    for (const secret of customSecrets) {
      if (secret.key.trim() && secret.value.trim()) {
        merged[secret.key] = secret.value;
      }
    }
    return merged;
  }, [providedSecrets, customSecrets]);

  const deploy = useCallback(async () => {
    setDeploymentStatus("deploying");
    setDeploymentError(null);
    setDeploymentId(null);

    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid manifest: ${parsedManifest.error}`);
      }

      // Step 1: Create or get existing deployment record with bundle info
      const createRes = await createDeployment(parsedManifest.value, {
        bundlePath: bundleArtifact?.path,
        bundleSha256: bundleArtifact?.sha256,
        bundleSizeBytes: bundleArtifact?.size_bytes,
      });
      const newDeploymentId = createRes.deployment.id;
      setDeploymentId(newDeploymentId);

      // Save deployment ID to localStorage for reconnection on refresh
      saveDeployment({
        manifestJson,
        currentStep: currentStepIndex,
        timestamp: Date.now(),
        sshKeyPath,
        deploymentId: newDeploymentId,
        deploymentStatus: "deploying",
      });

      // Step 2: Execute deployment (setup + deploy)
      // This is now non-blocking - it returns immediately.
      // Progress is tracked via SSE on /deployments/{id}/progress
      // and status updates come via the onDeploymentComplete callback
      await executeDeployment(newDeploymentId, { providedSecrets: mergedProvidedSecrets });

      // Status remains "deploying" - will be updated via SSE progress events
    } catch (e) {
      setDeploymentError(e instanceof Error ? e.message : String(e));
      setDeploymentStatus("failed");
    }
  }, [parsedManifest, manifestJson, currentStepIndex, sshKeyPath, bundleArtifact, mergedProvidedSecrets]);

  // Called when SSE progress stream reports completion or error
  const onDeploymentComplete = useCallback((success: boolean, error?: string) => {
    const newStatus = success ? "success" : "failed";
    if (success) {
      setDeploymentStatus("success");
      setDeploymentError(null);
    } else {
      setDeploymentStatus("failed");
      setDeploymentError(error || "Deployment failed");
    }
    // Save final status to localStorage
    saveDeployment({
      manifestJson,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath,
      deploymentId,
      deploymentStatus: newStatus,
    });
  }, [manifestJson, currentStepIndex, sshKeyPath, deploymentId]);

  // Reset manifest to defaults (keeps current step)
  const resetManifestToDefaults = useCallback(() => {
    // Push current state to history so user can undo
    historyRef.current = [...historyRef.current, manifestJson].slice(-MAX_HISTORY_SIZE);
    futureRef.current = [];

    setManifestJson(DEFAULT_MANIFEST_JSON);
    setValidationIssues(null);
    setValidationError(null);
    setNormalizedManifest(null);
    saveDeployment({
      manifestJson: DEFAULT_MANIFEST_JSON,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath,
      deploymentId,
      deploymentStatus,
    });
  }, [manifestJson, currentStepIndex, sshKeyPath, deploymentId, deploymentStatus]);

  // Reset manifest but preserve scenario selection and auto-populate its ports
  const resetManifestWithScenario = useCallback((scenarioId: string, scenarioPorts: Record<string, number>) => {
    // Push current state to history so user can undo
    historyRef.current = [...historyRef.current, manifestJson].slice(-MAX_HISTORY_SIZE);
    futureRef.current = [];

    const newManifest: DeploymentManifest = {
      ...DEFAULT_MANIFEST,
      scenario: { id: scenarioId },
      dependencies: { ...DEFAULT_MANIFEST.dependencies, scenarios: [scenarioId] },
      ports: scenarioPorts,
    };
    const json = JSON.stringify(newManifest, null, 2);

    setManifestJson(json);
    setValidationIssues(null);
    setValidationError(null);
    setNormalizedManifest(null);
    saveDeployment({
      manifestJson: json,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath,
      deploymentId,
      deploymentStatus,
    });
  }, [manifestJson, currentStepIndex, sshKeyPath, deploymentId, deploymentStatus]);

  // Full reset (returns to step 0, clears everything)
  const reset = useCallback(() => {
    clearSavedDeployment();
    historyRef.current = [];
    futureRef.current = [];
    setCurrentStepIndex(0);
    setManifestJson(DEFAULT_MANIFEST_JSON);
    setValidationIssues(null);
    setValidationError(null);
    setNormalizedManifest(null);
    setSecretsManifest(null);
    setSecretsError(null);
    setSecretsFetched(false);
    setProvidedSecrets({});
    setCustomSecrets([]);
    setBundleArtifact(null);
    setBundleError(null);
    setPreflightPassed(null);
    setPreflightChecks(null);
    setPreflightError(null);
    setPreflightOverride(false);
    setDeploymentStatus("idle");
    setDeploymentError(null);
    setDeploymentId(null);
    setSSHKeyPath(null);
    setSSHConnectionStatus("untested");
  }, []);

  // Computed state
  const canProceed = useMemo(() => {
    switch (currentStep.id) {
      case "manifest":
        // Merged manifest + validate: require valid JSON AND no blocking validation errors
        return (
          parsedManifest.ok &&
          validationIssues !== null &&
          validationIssues.filter((i) => i.severity === "error").length === 0
        );
      case "secrets":
        // Must have completed fetch attempt
        if (!secretsFetched) return false;
        // Validate custom secrets if any are defined
        if (customSecrets.length > 0 && !customSecretsValidation.isValid) return false;
        // If no secrets manifest (null response), no secrets required - can proceed
        if (!secretsManifest) return true;
        // Check all required user_prompt secrets are provided
        const requiredUserPrompt = secretsManifest.bundle_secrets.filter(
          (s) => s.class === "user_prompt" && s.required
        );
        return requiredUserPrompt.every((s) => {
          const key = s.target.name || s.id;
          return providedSecrets[key]?.trim();
        });
      case "build":
        return bundleArtifact !== null;
      case "preflight":
        // Allow proceeding if passed, OR if user explicitly overrides (checks must have run)
        return preflightPassed === true || (preflightOverride && preflightChecks !== null);
      case "deploy":
        return deploymentStatus === "success";
      default:
        return false;
    }
  }, [currentStep.id, parsedManifest, validationIssues, secretsFetched, secretsManifest, providedSecrets, bundleArtifact, preflightPassed, preflightOverride, preflightChecks, deploymentStatus]);

  const hasSavedProgress = saved !== null && saved.manifestJson !== DEFAULT_MANIFEST_JSON;

  return {
    // Navigation
    currentStepIndex,
    currentStep,
    steps: WIZARD_STEPS,
    goToStep,
    goNext,
    goBack,
    canProceed,
    hasSavedProgress,

    // Manifest
    manifestJson,
    setManifestJson: updateManifestJson,
    parsedManifest,

    // Undo/redo
    undo,
    redo,
    canUndo,
    canRedo,

    // Reset options
    resetManifestToDefaults,
    resetManifestWithScenario,

    // Validation
    validationIssues,
    validationError,
    isValidating,
    validate,
    normalizedManifest,
    applyAllFixes,

    // Secrets
    secretsManifest,
    secretsError,
    isFetchingSecrets,
    secretsFetched,
    fetchSecrets,
    providedSecrets,
    setProvidedSecrets: updateProvidedSecret,

    // Custom secrets (manually added)
    customSecrets,
    addCustomSecret,
    removeCustomSecret,
    updateCustomSecret,
    customSecretsValidation,

    // Bundle
    bundleArtifact,
    setBundleArtifact,
    bundleError,
    isBuildingBundle,
    build,

    // Preflight
    preflightPassed,
    preflightChecks,
    preflightError,
    isRunningPreflight,
    preflightOverride,
    setPreflightOverride,
    runPreflight,

    // Deploy
    deploymentStatus,
    deploymentError,
    deploymentId,
    deploy,
    onDeploymentComplete,
    isReconnecting,

    // SSH
    sshKeyPath,
    setSSHKeyPath: updateSSHKeyPath,
    sshConnectionStatus,
    setSSHConnectionStatus,

    // Actions
    reset,
  };
}
