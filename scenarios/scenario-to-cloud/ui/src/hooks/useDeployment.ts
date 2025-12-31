import { useState, useMemo, useCallback } from "react";
import {
  validateManifest,
  buildBundle,
  runPreflight as runPreflightApi,
  createDeployment,
  executeDeployment,
  type ValidationIssue,
  type BundleArtifact,
  type PreflightCheck,
  type SSHConnectionStatus,
} from "../lib/api";
import {
  type DeploymentManifest,
  type WizardStep,
  DEFAULT_MANIFEST_JSON,
  WIZARD_STEPS,
} from "../types/deployment";

const STORAGE_KEY = "scenario-to-cloud:deployment";

type SavedDeployment = {
  manifestJson: string;
  currentStep: number;
  timestamp: number;
  sshKeyPath?: string | null;
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
  const [bundleArtifact, setBundleArtifact] = useState<BundleArtifact | null>(null);
  const [bundleError, setBundleError] = useState<string | null>(null);
  const [isBuildingBundle, setIsBuildingBundle] = useState(false);

  // Preflight state
  const [preflightPassed, setPreflightPassed] = useState<boolean | null>(null);
  const [preflightChecks, setPreflightChecks] = useState<PreflightCheck[] | null>(null);
  const [preflightError, setPreflightError] = useState<string | null>(null);
  const [isRunningPreflight, setIsRunningPreflight] = useState(false);
  const [preflightOverride, setPreflightOverride] = useState(false);

  // Deploy state
  const [deploymentStatus, setDeploymentStatus] = useState<"idle" | "deploying" | "success" | "failed">("idle");
  const [deploymentError, setDeploymentError] = useState<string | null>(null);
  const [deploymentId, setDeploymentId] = useState<string | null>(null);

  // SSH state
  const [sshKeyPath, setSSHKeyPath] = useState<string | null>(saved?.sshKeyPath ?? null);
  const [sshConnectionStatus, setSSHConnectionStatus] = useState<SSHConnectionStatus>("untested");

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

  // Save on manifest/step change
  const updateManifestJson = useCallback((json: string) => {
    setManifestJson(json);
    saveDeployment({
      manifestJson: json,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath,
    });
  }, [currentStepIndex, sshKeyPath]);

  // Navigation
  const goToStep = useCallback((index: number) => {
    if (index >= 0 && index < WIZARD_STEPS.length) {
      setCurrentStepIndex(index);
      saveDeployment({
        manifestJson,
        currentStep: index,
        timestamp: Date.now(),
        sshKeyPath,
      });
    }
  }, [manifestJson, sshKeyPath]);

  // Update SSH key path with persistence
  const updateSSHKeyPath = useCallback((keyPath: string | null) => {
    setSSHKeyPath(keyPath);
    setSSHConnectionStatus("untested"); // Reset connection status when key changes
    saveDeployment({
      manifestJson,
      currentStep: currentStepIndex,
      timestamp: Date.now(),
      sshKeyPath: keyPath,
    });
  }, [manifestJson, currentStepIndex]);

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
      });
      // Clear validation state to trigger re-validation
      setValidationIssues(null);
      setNormalizedManifest(null);
    }
  }, [normalizedManifest, currentStepIndex, sshKeyPath]);

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
  }, [parsedManifest]);

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

  const deploy = useCallback(async () => {
    setDeploymentStatus("deploying");
    setDeploymentError(null);
    setDeploymentId(null);

    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid manifest: ${parsedManifest.error}`);
      }

      // Step 1: Create or get existing deployment record
      const createRes = await createDeployment(parsedManifest.value);
      setDeploymentId(createRes.deployment.id);

      // Step 2: Execute deployment (setup + deploy)
      // This is now non-blocking - it returns immediately.
      // Progress is tracked via SSE on /deployments/{id}/progress
      // and status updates come via the onDeploymentComplete callback
      await executeDeployment(createRes.deployment.id);

      // Status remains "deploying" - will be updated via SSE progress events
    } catch (e) {
      setDeploymentError(e instanceof Error ? e.message : String(e));
      setDeploymentStatus("failed");
    }
  }, [parsedManifest]);

  // Called when SSE progress stream reports completion or error
  const onDeploymentComplete = useCallback((success: boolean, error?: string) => {
    if (success) {
      setDeploymentStatus("success");
      setDeploymentError(null);
    } else {
      setDeploymentStatus("failed");
      setDeploymentError(error || "Deployment failed");
    }
  }, []);

  const reset = useCallback(() => {
    clearSavedDeployment();
    setCurrentStepIndex(0);
    setManifestJson(DEFAULT_MANIFEST_JSON);
    setValidationIssues(null);
    setValidationError(null);
    setNormalizedManifest(null);
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
  }, [currentStep.id, parsedManifest, validationIssues, bundleArtifact, preflightPassed, preflightOverride, preflightChecks, deploymentStatus]);

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

    // Validation
    validationIssues,
    validationError,
    isValidating,
    validate,
    normalizedManifest,
    applyAllFixes,

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

    // SSH
    sshKeyPath,
    setSSHKeyPath: updateSSHKeyPath,
    sshConnectionStatus,
    setSSHConnectionStatus,

    // Actions
    reset,
  };
}
