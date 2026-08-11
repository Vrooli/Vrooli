import { useCallback, useEffect, useRef, useState } from "react";
import { fetchOperatorState, saveOperatorState } from "../lib/api";
import type { OperatorState, OperatorStatePatch } from "../types";
import { STEP_PATHS, TOTAL_STEPS } from "../types";

function stepForPath(pathname: string) {
  // Apply is a durable deep-link alias for the validation step. Keeping it as
  // an alias lets evidence and deployment surfaces name the commitment phase
  // without creating a second source of wizard state.
  if (pathname === "/setup/apply") return TOTAL_STEPS - 1;
  const index = STEP_PATHS.indexOf(pathname as (typeof STEP_PATHS)[number]);
  return index >= 0 ? index : 0;
}

export function useWizardState() {
  const [currentStep, setCurrentStep] = useState(() => stepForPath(window.location.pathname));
  const [selectedScenarios, setSelectedScenarios] = useState<Set<string>>(new Set());
  const [operatorState, setOperatorState] = useState<OperatorState | null>(null);
  const stepContentRef = useRef<HTMLDivElement>(null);
  const prevStepRef = useRef(currentStep);

  // V2 re-entry loads durable operator choices, not database-backed progress.
  useEffect(() => {
    fetchOperatorState()
      .then((state) => {
        setOperatorState(state);
        setSelectedScenarios(new Set(Object.entries(state.scenarios ?? {}).filter(([, choice]) => choice.enabled).map(([name]) => name)));
      })
      .catch(() => {
        // An unconfigured installation has no operator-state document yet.
      });
  }, []);

  useEffect(() => {
    const onPopState = () => setCurrentStep(stepForPath(window.location.pathname));
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  const moveToStep = useCallback((step: number, replace = false) => {
    if (step < 0 || step >= TOTAL_STEPS) return;
    const path = STEP_PATHS[step];
    if (window.location.pathname !== path) {
      window.history[replace ? "replaceState" : "pushState"]({}, "", path);
    }
    setCurrentStep(step);
  }, []);

  const persistOperatorState = useCallback((patch: OperatorStatePatch) => {
    setOperatorState((previous) => {
      const base = previous ?? { version: "1.0.0", updated_at: "" };
      return {
        ...base,
        ...patch,
        scenarios: patch.scenarios ? { ...(base.scenarios ?? {}), ...patch.scenarios } : base.scenarios,
        resources: patch.resources ? { ...(base.resources ?? {}), ...patch.resources } : base.resources,
        host_tools: patch.host_tools ? { ...(base.host_tools ?? {}), ...patch.host_tools } : base.host_tools,
        host_safeguards: patch.host_safeguards ? { ...(base.host_safeguards ?? {}), ...patch.host_safeguards } : base.host_safeguards,
      };
    });
    saveOperatorState(patch).then(setOperatorState).catch(() => {
      // Keep the in-memory choice visible. The next operator action retries it.
    });
  }, []);

  const toggleScenario = useCallback((name: string) => {
    setSelectedScenarios((prev) => {
      const next = new Set(prev);
      const enabled = !next.has(name);
      if (enabled) next.add(name); else next.delete(name);
      persistOperatorState({ scenarios: { [name]: { ...(operatorState?.scenarios?.[name] ?? {}), enabled } } });
      return next;
    });
  }, [operatorState, persistOperatorState]);

  const setScenarioAutoRestart = useCallback((name: string, autoRestart: boolean) => {
    persistOperatorState({ scenarios: { [name]: { ...(operatorState?.scenarios?.[name] ?? {}), auto_restart: autoRestart } } });
  }, [operatorState, persistOperatorState]);
  const setHostOptIn = useCallback((kind: "host_tools" | "host_safeguards", name: string, optedIn: boolean) => {
    persistOperatorState({ [kind]: { [name]: { opted_in: optedIn } } });
  }, [operatorState, persistOperatorState]);

  const setResourceEnabled = useCallback((name: string, enabled: boolean) => {
    persistOperatorState({ resources: { [name]: { enabled } } });
  }, [persistOperatorState]);

  const goNext = useCallback(() => {
    moveToStep(Math.min(currentStep + 1, TOTAL_STEPS - 1));
  }, [currentStep, moveToStep]);

  const goPrev = useCallback(() => {
    moveToStep(Math.max(currentStep - 1, 0));
  }, [currentStep, moveToStep]);

  const goToStep = useCallback((step: number) => {
    if (step >= 0 && step < TOTAL_STEPS) {
      moveToStep(step);
    }
  }, [moveToStep]);

  const startOver = useCallback(() => {
    moveToStep(0, true);
    setSelectedScenarios(new Set());
  }, []);

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

  const nextLabel = currentStep === 0 ? "Get Started" : currentStep === TOTAL_STEPS - 2 ? "Review readiness" : "Next";
  const isLastStep = currentStep === TOTAL_STEPS - 1;

  return {
    currentStep,
    selectedScenarios,
    operatorState,
    stepContentRef,
    toggleScenario,
    setScenarioAutoRestart,
    setHostOptIn,
    setResourceEnabled,
    goNext,
    goPrev,
    goToStep,
    startOver,
    nextLabel,
    isLastStep,
    totalSteps: TOTAL_STEPS,
  };
}
