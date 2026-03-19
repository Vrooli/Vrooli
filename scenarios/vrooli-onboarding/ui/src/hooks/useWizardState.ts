import { useCallback, useEffect, useRef, useState } from "react";
import { updateProgress, fetchProgress } from "../lib/api";
import { TOTAL_STEPS } from "../types";

export function useWizardState() {
  const [currentStep, setCurrentStep] = useState(0);
  const [selectedResources, setSelectedResources] = useState<Set<string>>(new Set());
  const [resumeAvailable, setResumeAvailable] = useState(false);
  const [resumeStep, setResumeStep] = useState(0);
  const stepContentRef = useRef<HTMLDivElement>(null);
  const prevStepRef = useRef(currentStep);

  // Check for saved progress on mount
  useEffect(() => {
    fetchProgress()
      .then((progress) => {
        if (progress.current_step > 0) {
          setResumeAvailable(true);
          setResumeStep(progress.current_step);
          const configData = progress.config_data;
          if (configData && Array.isArray(configData.resources)) {
            const resources = configData.resources;
            if (Array.isArray(resources) && resources.every((r): r is string => typeof r === "string")) {
              setSelectedResources(new Set(resources));
            }
          }
        }
      })
      .catch(() => {
        // No saved progress - start fresh
      });
  }, []);

  const handleResume = useCallback(() => {
    setCurrentStep(resumeStep);
    setResumeAvailable(false);
  }, [resumeStep]);

  const toggleResource = useCallback((name: string) => {
    setSelectedResources((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }, []);

  const goNext = useCallback(() => {
    setCurrentStep((prev) => {
      const next = Math.min(prev + 1, TOTAL_STEPS - 1);
      updateProgress({
        current_step: next,
        completed_steps: Array.from({ length: next }, (_, i) => i),
        config_data: { resources: Array.from(selectedResources) },
      }).catch(() => {
        // Silently ignore progress save failures
      });
      return next;
    });
  }, [selectedResources]);

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
    setSelectedResources(new Set());
    setResumeAvailable(false);
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

  const nextLabel = currentStep === 0 ? "Get Started" : currentStep === TOTAL_STEPS - 2 ? "Generate Config" : "Next";
  const isLastStep = currentStep === TOTAL_STEPS - 1;

  return {
    currentStep,
    selectedResources,
    resumeAvailable,
    resumeStep,
    stepContentRef,
    handleResume,
    toggleResource,
    goNext,
    goPrev,
    goToStep,
    startOver,
    nextLabel,
    isLastStep,
    totalSteps: TOTAL_STEPS,
  };
}
