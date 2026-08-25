import { useCallback, useEffect, useRef, useState } from "react";
import {
  fetchOperatorState,
  fetchV2Recommendation,
  fetchV2Session,
  fetchV2Steps,
  saveOperatorState,
  saveV2SessionStep,
} from "../lib/api";
import type { OperatorState, OperatorStatePatch, V2Step } from "../types";

function stepForPath(pathname: string, steps: V2Step[]) {
  const index = steps.findIndex((step) => step.route === pathname);
  return index >= 0 ? index : 0;
}

export function useWizardState() {
  const [steps, setSteps] = useState<V2Step[]>([]);
  const [stepsLoading, setStepsLoading] = useState(true);
  const [stepsError, setStepsError] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [selectedScenarios, setSelectedScenarios] = useState<Set<string>>(
    new Set(),
  );
  const [operatorState, setOperatorState] = useState<OperatorState | null>(
    null,
  );
  const stepContentRef = useRef<HTMLDivElement>(null);
  const prevStepRef = useRef(currentStep);

  // V2 re-entry loads durable operator choices, not database-backed progress.
  useEffect(() => {
    Promise.all([fetchV2Steps(), fetchOperatorState(), fetchV2Recommendation()])
      .then(([model, state, recommendation]) => {
        setSteps(model.steps.slice().sort((a, b) => a.ordinal - b.ordinal));
        setStepsError(null);
        setCurrentStep(stepForPath(window.location.pathname, model.steps));
        setOperatorState(state);
        const selected = new Set(
          Object.entries(state.scenarios ?? {})
            .filter(([, choice]) => choice.enabled)
            .map(([name]) => name),
        );
        const recommendedScenarios = recommendation?.scenarios ?? [];
        if (selected.size === 0 && recommendedScenarios.length > 0) {
          recommendedScenarios.forEach((name) => selected.add(name));
          const patch = {
            active_profile: recommendation?.profile ?? "starter",
            scenarios: Object.fromEntries(
              recommendedScenarios.map((name) => [name, { enabled: true }]),
            ) as Record<string, { enabled: boolean }>,
          };
          saveOperatorState(patch)
            .then(setOperatorState)
            .catch(() => undefined);
        }
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
        window.history[replace ? "replaceState" : "pushState"]({}, "", path);
      }
      setCurrentStep(step);
      void saveV2SessionStep(step).catch(() => undefined);
    },
    [steps],
  );

  useEffect(() => {
    if (
      steps.length > 0 &&
      (window.location.pathname === "/" ||
        window.location.pathname === "/setup")
    ) {
      fetchV2Session()
        .then((session) => {
          if (
            Number.isInteger(session.first_unsatisfied_step) &&
            session.first_unsatisfied_step >= 0
          ) {
            moveToStep(session.first_unsatisfied_step, true);
          }
        })
        .catch(() => undefined);
    }
  }, [moveToStep, steps.length]);

  const persistOperatorState = useCallback((patch: OperatorStatePatch) => {
    setOperatorState((previous) => {
      const base = previous ?? { version: "1.0.0", updated_at: "" };
      return {
        ...base,
        ...patch,
        scenarios: patch.scenarios
          ? { ...(base.scenarios ?? {}), ...patch.scenarios }
          : base.scenarios,
        resources: patch.resources
          ? { ...(base.resources ?? {}), ...patch.resources }
          : base.resources,
        host_tools: patch.host_tools
          ? { ...(base.host_tools ?? {}), ...patch.host_tools }
          : base.host_tools,
        host_safeguards: patch.host_safeguards
          ? { ...(base.host_safeguards ?? {}), ...patch.host_safeguards }
          : base.host_safeguards,
      };
    });
    saveOperatorState(patch)
      .then(setOperatorState)
      .catch(() => {
        // Keep the in-memory choice visible. The next operator action retries it.
      });
  }, []);

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
            auto_restart: autoRestart,
          },
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
      persistOperatorState({ [kind]: { [name]: { opted_in: optedIn } } });
    },
    [operatorState, persistOperatorState],
  );

  const setHostConfig = useCallback(
    (
      kind: "host_tools" | "host_safeguards",
      name: string,
      config: Record<string, unknown>,
    ) => {
      persistOperatorState({ [kind]: { [name]: { config } } });
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
  };
}
