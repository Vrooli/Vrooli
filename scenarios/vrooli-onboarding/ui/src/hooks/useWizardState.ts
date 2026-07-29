import { useCallback, useEffect, useRef, useState } from "react";
import { fetchOperatorState, saveOperatorState } from "../lib/api";
import type { OperatorState } from "../types";
import { TOTAL_STEPS } from "../types";

export function useWizardState() {
  const [currentStep, setCurrentStep] = useState(0);
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

  const persistOperatorState = useCallback((next: OperatorState) => {
    setOperatorState(next);
    saveOperatorState(next).then(setOperatorState).catch(() => {
      // Keep the in-memory choice visible. The next operator action retries it.
    });
  }, []);

  const toggleScenario = useCallback((name: string) => {
    setSelectedScenarios((prev) => {
      const next = new Set(prev);
      const enabled = !next.has(name);
      if (enabled) next.add(name); else next.delete(name);
      const nextState: OperatorState = {
        ...(operatorState ?? { version: "1.0.0", updated_at: "" }),
        scenarios: { ...(operatorState?.scenarios ?? {}), [name]: { ...(operatorState?.scenarios?.[name] ?? {}), enabled } },
      };
      persistOperatorState(nextState);
      return next;
    });
  }, [operatorState, persistOperatorState]);

  const setScenarioAutoRestart = useCallback((name: string, autoRestart: boolean) => {
    const next: OperatorState = {
      ...(operatorState ?? { version: "1.0.0", updated_at: "" }),
      scenarios: { ...(operatorState?.scenarios ?? {}), [name]: { ...(operatorState?.scenarios?.[name] ?? {}), auto_restart: autoRestart } },
    };
    persistOperatorState(next);
  }, [operatorState, persistOperatorState]);
  const setHostOptIn = useCallback((kind: "host_tools" | "host_safeguards", name: string, optedIn: boolean) => {
    const next: OperatorState = { ...(operatorState ?? { version: "1.0.0", updated_at: "" }), [kind]: { ...(operatorState?.[kind] ?? {}), [name]: { opted_in: optedIn } } };
    persistOperatorState(next);
  }, [operatorState, persistOperatorState]);

  const goNext = useCallback(() => {
    setCurrentStep((prev) => Math.min(prev + 1, TOTAL_STEPS - 1));
  }, []);

  const goPrev = useCallback(() => {
    setCurrentStep((prev) => Math.max(prev - 1, 0));
  }, []);

  const goToStep = useCallback((step: number) => {
    if (step >= 0 && step < TOTAL_STEPS) {
      setCurrentStep(step);
    }
  }, []);

  const startOver = useCallback(() => {
    setCurrentStep(0);
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
    goNext,
    goPrev,
    goToStep,
    startOver,
    nextLabel,
    isLastStep,
    totalSteps: TOTAL_STEPS,
  };
}
