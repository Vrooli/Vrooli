import { useCallback, useEffect, useRef, useState } from "react";
import {
  fetchOperatorState,
  saveOperatorStateAtRevision,
  type OperatorState,
  type OperatorStatePatch,
} from "../api/operatorstate";
import { advanceSessionStep, fetchSession, fetchStepModel } from "../api/session";
import { acceptRecommendation as acceptRecommendationRequest } from "../api/selection";
import type { Step } from "@vrooli/proto-types/vrooli-onboarding/v1/session/session_pb";

function stepForPath(pathname: string, steps: Step[]) {
  const index = steps.findIndex((step) => step.route === pathname);
  return index >= 0 ? index : 0;
}

function mergePatches(base: OperatorStatePatch | null, next: OperatorStatePatch): OperatorStatePatch {
  if (!base) return next;
  return {
    ...base,
    ...next,
    core: next.core ?? base.core,
    scenarios: next.scenarios ? { ...(base.scenarios ?? {}), ...next.scenarios } : base.scenarios,
    resources: next.resources ? { ...(base.resources ?? {}), ...next.resources } : base.resources,
    hostTools: next.hostTools ? { ...(base.hostTools ?? {}), ...next.hostTools } : base.hostTools,
    hostSafeguards: next.hostSafeguards ? { ...(base.hostSafeguards ?? {}), ...next.hostSafeguards } : base.hostSafeguards,
  };
}

export function useWizardState() {
  const [steps, setSteps] = useState<Step[]>([]);
  const [stepsLoading, setStepsLoading] = useState(true);
  const [stepsError, setStepsError] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [selectedScenarios, setSelectedScenarios] = useState<Set<string>>(
    new Set(),
  );
  const [operatorState, setOperatorState] = useState<OperatorState | null>(
    null,
  );
  const [planAccepted, setPlanAccepted] = useState(false);
  const [operatorStateError, setOperatorStateError] = useState<string | null>(null);
  const [operatorStateSaveState, setOperatorStateSaveState] = useState<
    "idle" | "saving" | "saved" | "failed" | "conflict"
  >("idle");
  const stepContentRef = useRef<HTMLDivElement>(null);
  const prevStepRef = useRef(currentStep);
  const operatorStateRef = useRef<OperatorState | null>(null);
  const pendingPatchRef = useRef<OperatorStatePatch | null>(null);
  const failedPatchRef = useRef<OperatorStatePatch | null>(null);
  const saveGenerationRef = useRef(0);

  // V2 re-entry loads durable operator choices, not database-backed progress.
  useEffect(() => {
    Promise.all([fetchStepModel(), fetchOperatorState()])
      .then(([model, state]) => {
        setSteps(model.steps.slice().sort((a, b) => a.ordinal - b.ordinal));
        setStepsError(null);
        setCurrentStep(stepForPath(window.location.pathname, model.steps));
        setOperatorState(state);
        operatorStateRef.current = state;
        const selected = new Set(
          Object.entries(state.scenarios ?? {})
            .filter(([, choice]) => choice.enabled)
            .map(([name]) => name),
        );
        setSelectedScenarios(selected);
      })
      .catch(() => {
        setStepsError("The onboarding step model could not be loaded.");
      })
      .finally(() => {
        setStepsLoading(false);
      });
  }, []);

  useEffect(() => {
    const onPopState = () =>
      setCurrentStep(stepForPath(window.location.pathname, steps));
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, [steps]);

  const moveToStep = useCallback(
    (step: number, replace = false) => {
      if (step < 0 || step >= steps.length) return;
      const path = steps[step]?.route;
      if (!path) return;
      if (window.location.pathname !== path) {
        const params = new URLSearchParams(window.location.search);
        const query = params.toString();
        window.history[replace ? "replaceState" : "pushState"]({}, "", query ? `${path}?${query}` : path);
      }
      setCurrentStep(step);
      const stepId = steps[step]?.id;
      if (stepId) void advanceSessionStep(stepId).catch(() => undefined);
    },
    [steps],
  );

  useEffect(() => {
    if (
      steps.length > 0 &&
      (window.location.pathname === "/" ||
        window.location.pathname === "/setup")
    ) {
      fetchSession()
        .then((session) => {
          if (
            Number.isInteger(session.firstUnsatisfiedStep) &&
            session.firstUnsatisfiedStep >= 0
          ) {
            moveToStep(session.firstUnsatisfiedStep, true);
          }
        })
        .catch(() => undefined);
    }
  }, [moveToStep, steps.length]);

  const mergeOperatorState = useCallback((base: OperatorState, patch: OperatorStatePatch): OperatorState => ({
    ...base,
    ...patch,
    core: patch.core ?? base.core,
    scenarios: patch.scenarios
      ? { ...(base.scenarios ?? {}), ...patch.scenarios }
      : base.scenarios,
    resources: patch.resources
      ? { ...(base.resources ?? {}), ...patch.resources }
      : base.resources,
    hostTools: patch.hostTools
      ? { ...(base.hostTools ?? {}), ...patch.hostTools }
      : base.hostTools,
    hostSafeguards: patch.hostSafeguards
      ? { ...(base.hostSafeguards ?? {}), ...patch.hostSafeguards }
      : base.hostSafeguards,
  }), []);

  const persistOperatorState = useCallback((patch: OperatorStatePatch) => {
    const base = operatorStateRef.current ?? { version: "1.0.0", updatedAt: "" };
    const optimistic = mergeOperatorState(base, patch);
    const expectedRevision = base.updatedAt ?? "";
    const requestPatch = mergePatches(pendingPatchRef.current, patch);
    const generation = ++saveGenerationRef.current;
    pendingPatchRef.current = requestPatch;
    operatorStateRef.current = optimistic;
    setOperatorState(optimistic);
    setOperatorStateSaveState("saving");
    setOperatorStateError(null);
    saveOperatorStateAtRevision(requestPatch, expectedRevision)
      .then((state) => {
        // A late response must not erase a newer optimistic edit. The durable
        // revision check rejects the competing write; the local edit remains
        // visible for an explicit retry/rebase.
        if (generation !== saveGenerationRef.current) return;
        operatorStateRef.current = state;
        setOperatorState(state);
        pendingPatchRef.current = null;
        failedPatchRef.current = null;
        setOperatorStateSaveState("saved");
        setOperatorStateError(null);
      })
      .catch((error: unknown) => {
        if (generation !== saveGenerationRef.current) return;
        const message = error instanceof Error ? error.message : String(error);
        const conflict = message.toLowerCase().includes("conflict") || message.toLowerCase().includes("aborted");
        failedPatchRef.current = requestPatch;
        setOperatorStateSaveState(conflict ? "conflict" : "failed");
        setOperatorStateError(conflict
          ? "Another client changed these preferences. Your edit is retained; reload and retry to rebase it."
          : "The latest choice could not be saved. Your edit is retained; retry when the connection recovers.");
      });
  }, [mergeOperatorState]);

  const retryOperatorStateSave = useCallback(async () => {
    const patch = failedPatchRef.current;
    if (!patch) return;
    let latest: OperatorState;
    try {
      latest = await fetchOperatorState();
    } catch {
      setOperatorStateSaveState("failed");
      setOperatorStateError("The latest server state could not be loaded. Your edit remains available for retry.");
      return;
    }
    const rebased = mergeOperatorState(latest, patch);
    operatorStateRef.current = rebased;
    pendingPatchRef.current = patch;
    setOperatorState(rebased);
    setOperatorStateSaveState("saving");
    setOperatorStateError(null);
    try {
      const saved = await saveOperatorStateAtRevision(patch, latest.updatedAt ?? "");
      operatorStateRef.current = saved;
      setOperatorState(saved);
      pendingPatchRef.current = null;
      failedPatchRef.current = null;
      setOperatorStateSaveState("saved");
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : String(error);
      setOperatorStateSaveState(message.toLowerCase().includes("conflict") ? "conflict" : "failed");
      setOperatorStateError("The edit is still not saved. It remains available for another retry.");
    }
  }, [mergeOperatorState]);

  const toggleScenario = useCallback(
    (name: string) => {
      setSelectedScenarios((prev) => {
        const next = new Set(prev);
        const enabled = !next.has(name);
        if (enabled) next.add(name);
        else next.delete(name);
        persistOperatorState({
          scenarios: {
            [name]: { ...(operatorState?.scenarios?.[name] ?? {}), enabled },
          },
        });
        return next;
      });
    },
    [operatorState, persistOperatorState],
  );

  const setScenarioAutoRestart = useCallback(
    (name: string, autoRestart: boolean) => {
      persistOperatorState({
        scenarios: {
          [name]: {
            ...(operatorState?.scenarios?.[name] ?? {}),
            autoRestart,
          },
        },
      });
    },
    [operatorState, persistOperatorState],
  );

  const setCoreSeed = useCallback(
    (seed: string[]) => {
      persistOperatorState({
        core: {
          seed: Array.from(new Set(seed)).sort(),
          trustedBase: operatorState?.core?.trustedBase ?? [],
        },
      });
    },
    [operatorState, persistOperatorState],
  );
  const setHostOptIn = useCallback(
    (
      kind: "host_tools" | "host_safeguards",
      name: string,
      optedIn: boolean,
    ) => {
      const field = kind === "host_tools" ? "hostTools" : "hostSafeguards";
      persistOperatorState({ [field]: { [name]: { optedIn } } });
    },
    [operatorState, persistOperatorState],
  );

  const setHostConfig = useCallback(
    (
      kind: "host_tools" | "host_safeguards",
      name: string,
      config: Record<string, unknown>,
    ) => {
      const field = kind === "host_tools" ? "hostTools" : "hostSafeguards";
      persistOperatorState({ [field]: { [name]: { config } } });
    },
    [persistOperatorState],
  );

  const setResourceEnabled = useCallback(
    (name: string, enabled: boolean) => {
      persistOperatorState({ resources: { [name]: { enabled } } });
    },
    [persistOperatorState],
  );

  const goNext = useCallback(() => {
    moveToStep(Math.min(currentStep + 1, steps.length - 1));
  }, [currentStep, moveToStep, steps.length]);

  const acceptRecommendation = useCallback(async (profile?: string, scenarios?: string[]) => {
    const accepted = await acceptRecommendationRequest("local", profile, scenarios);
    const session = await fetchSession();
    setPlanAccepted(true);
    moveToStep(accepted.firstUnsatisfiedStep || session.firstUnsatisfiedStep, true);
  }, [moveToStep]);

  const goPrev = useCallback(() => {
    moveToStep(Math.max(currentStep - 1, 0));
  }, [currentStep, moveToStep]);

  const goToStep = useCallback(
    (step: number) => {
      if (step >= 0 && step < steps.length) {
        moveToStep(step);
      }
    },
    [moveToStep, steps.length],
  );

  const startOver = useCallback(() => {
    moveToStep(0, true);
    setSelectedScenarios(new Set());
  }, [moveToStep]);

  // Move focus to step content when step changes (accessibility)
  useEffect(() => {
    if (prevStepRef.current !== currentStep) {
      prevStepRef.current = currentStep;
      requestAnimationFrame(() => {
        const heading = stepContentRef.current?.querySelector("h1");
        if (heading) {
          heading.setAttribute("tabindex", "-1");
          heading.focus();
        }
      });
    }
  }, [currentStep]);

  const nextLabel =
    currentStep === 0
      ? "Get Started"
      : currentStep === steps.length - 2
        ? "Review readiness"
        : "Next";
  const isLastStep = steps.length > 0 && currentStep === steps.length - 1;

  return {
    currentStep,
    steps,
    stepsLoading,
    stepsError,
    selectedScenarios,
    operatorState,
    stepContentRef,
    toggleScenario,
    setCoreSeed,
    setScenarioAutoRestart,
    setHostOptIn,
    setHostConfig,
    setResourceEnabled,
    goNext,
    goPrev,
    goToStep,
    startOver,
    nextLabel,
    isLastStep,
    totalSteps: steps.length,
    planAccepted,
    operatorStateError,
    operatorStateSaveState,
    retryOperatorStateSave,
    acceptRecommendation,
  };
}
