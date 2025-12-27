import { useState, useMemo, useCallback } from "react";
import {
  validateManifest,
  buildPlan,
  buildBundle,
  runPreflight as runPreflightApi,
  type ValidationIssue,
  type PlanStep,
  type BundleArtifact,
  type PreflightCheck,
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

  // Plan state
  const [plan, setPlan] = useState<PlanStep[] | null>(null);
  const [planError, setPlanError] = useState<string | null>(null);
  const [isPlanning, setIsPlanning] = useState(false);

  // Bundle state
  const [bundleArtifact, setBundleArtifact] = useState<BundleArtifact | null>(null);
  const [bundleError, setBundleError] = useState<string | null>(null);
  const [isBuildingBundle, setIsBuildingBundle] = useState(false);

  // Preflight state
  const [preflightPassed, setPreflightPassed] = useState<boolean | null>(null);
  const [preflightChecks, setPreflightChecks] = useState<PreflightCheck[] | null>(null);
  const [preflightError, setPreflightError] = useState<string | null>(null);
  const [isRunningPreflight, setIsRunningPreflight] = useState(false);

  // Deploy state (placeholder)
  const [deploymentStatus, setDeploymentStatus] = useState<"idle" | "deploying" | "success" | "failed">("idle");
  const [deploymentError, setDeploymentError] = useState<string | null>(null);

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
    });
  }, [currentStepIndex]);

  // Navigation
  const goToStep = useCallback((index: number) => {
    if (index >= 0 && index < WIZARD_STEPS.length) {
      setCurrentStepIndex(index);
      saveDeployment({
        manifestJson,
        currentStep: index,
        timestamp: Date.now(),
      });
    }
  }, [manifestJson]);

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
    setIsValidating(true);

    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await validateManifest(parsedManifest.value);
      setValidationIssues(res.issues ?? []);
    } catch (e) {
      setValidationError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsValidating(false);
    }
  }, [parsedManifest]);

  const generatePlan = useCallback(async () => {
    setPlan(null);
    setPlanError(null);
    setIsPlanning(true);

    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await buildPlan(parsedManifest.value);
      setPlan(res.plan ?? []);
    } catch (e) {
      setPlanError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsPlanning(false);
    }
  }, [parsedManifest]);

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

    try {
      // Placeholder - will be implemented when deploy API is ready
      await new Promise((resolve) => setTimeout(resolve, 2000));
      setDeploymentStatus("success");
    } catch (e) {
      setDeploymentError(e instanceof Error ? e.message : String(e));
      setDeploymentStatus("failed");
    }
  }, []);

  const reset = useCallback(() => {
    clearSavedDeployment();
    setCurrentStepIndex(0);
    setManifestJson(DEFAULT_MANIFEST_JSON);
    setValidationIssues(null);
    setValidationError(null);
    setPlan(null);
    setPlanError(null);
    setBundleArtifact(null);
    setBundleError(null);
    setPreflightPassed(null);
    setPreflightChecks(null);
    setPreflightError(null);
    setDeploymentStatus("idle");
    setDeploymentError(null);
  }, []);

  // Computed state
  const canProceed = useMemo(() => {
    switch (currentStep.id) {
      case "manifest":
        return parsedManifest.ok;
      case "validate":
        return validationIssues !== null && validationIssues.filter((i) => i.severity === "error").length === 0;
      case "plan":
        return plan !== null && plan.length > 0;
      case "build":
        return bundleArtifact !== null;
      case "preflight":
        return preflightPassed === true;
      case "deploy":
        return deploymentStatus === "success";
      default:
        return false;
    }
  }, [currentStep.id, parsedManifest, validationIssues, plan, bundleArtifact, preflightPassed, deploymentStatus]);

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

    // Plan
    plan,
    planError,
    isPlanning,
    generatePlan,

    // Bundle
    bundleArtifact,
    bundleError,
    isBuildingBundle,
    build,

    // Preflight
    preflightPassed,
    preflightChecks,
    preflightError,
    isRunningPreflight,
    runPreflight,

    // Deploy
    deploymentStatus,
    deploymentError,
    deploy,

    // Actions
    reset,
  };
}
