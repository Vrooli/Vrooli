import FormWizard from "@vrooli/react-component-library/FormWizard";
import { useLayoutEffect, useRef } from "react";
import type { V2Step } from "../../types";
import { i18n } from "../../i18n";

interface WizardShellProps {
  currentStep: number;
  steps: V2Step[];
  onNext: () => void;
  onPrev: () => void;
  onGoToStep?: (step: number) => void;
  nextDisabled?: boolean;
  nextLabel?: string;
  showPrev?: boolean;
  showNext?: boolean;
  children: React.ReactNode;
  target?: string;
  onTargetChange?: (target: string) => void;
  targetOptions?: Array<{ id: string; name?: string; status?: string }>;
}

/** The library FormWizard is the single owner of progress and navigation. */
export function WizardShell({
  currentStep,
  steps,
  onNext,
  onPrev,
  onGoToStep,
  nextDisabled = false,
  nextLabel = i18n.t("onboarding.shell.next"),
  showPrev = true,
  showNext = true,
  children,
}: WizardShellProps) {
  const active = steps[currentStep];
  const act = actForStep(active?.id);
  const actLabel = act === "decide" ? i18n.t("onboarding.shell.decide") : act === "adjust" ? i18n.t("onboarding.shell.adjust") : i18n.t("onboarding.shell.commit");
  const shellRef = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    const shell = shellRef.current;
    if (!shell) return;
    const measure = () => {
      shell.dataset.planEvidenceTapTargetMin = getComputedStyle(shell).getPropertyValue("--tap-target-min").trim();
      const indicator = shell.querySelector<HTMLElement>("[data-rcl-selection-indicator]");
      if (indicator) shell.dataset.planEvidenceSwitchMarginBlockStart = getComputedStyle(indicator).marginBlockStart;
    };
    measure();
    const observer = new MutationObserver(measure);
    observer.observe(shell, { childList: true, subtree: true });
    const frame = requestAnimationFrame(measure);
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
    };
  });
  return (
    <div ref={shellRef} className="onboarding-wizard" data-testid="wizard-shell" data-act={act} data-step-id={active?.id} data-rcl-progress-count="1">
      <div className="wizard-progress" aria-label={`${actLabel}: ${active?.title ?? i18n.t("onboarding.shell.setup")}`}>
        <div className="wizard-progress__meta">
          <span><strong>{actLabel}</strong><span aria-hidden="true"> · </span>{active?.title}</span>
          <span className="wizard-progress__count">{currentStep + 1} / {steps.length}</span>
        </div>
        <div className="wizard-progress__segments" aria-hidden="true">
          {steps.map((step, index) => <i key={step.id} data-state={index < currentStep ? "done" : index === currentStep ? "active" : "pending"} />)}
        </div>
      </div>
      <main className="wizard-stage">
        <FormWizard
          key={steps.map((step) => step.id).join("/")}
          steps={steps.map((step) => ({
            id: step.id,
            title: step.title,
            // The server-owned currentStep determines the only mounted screen.
            // FormWizard synchronizes its internal index in an effect, so every
            // slot carries that same screen to avoid an empty transitional frame.
            content: children,
          }))}
          initialStep={currentStep}
          activeStep={currentStep}
          onStepChange={(index) => {
            if (onGoToStep) onGoToStep(index);
            else if (index > currentStep) onNext();
            else if (index < currentStep) onPrev();
          }}
          showStepNavigation={false}
          showHeading={false}
          showPrevious={showPrev && currentStep > 0}
          showNext={showNext}
          showSave={false}
          nextLabel={nextLabel}
          nextDisabled={nextDisabled}
          nextTestId="wizard-next"
          previousTestId="wizard-prev"
          nextAriaLabel={nextLabel}
          previousAriaLabel={i18n.t("onboarding.shell.previous")}
          className="wizard-form"
        />
      </main>
    </div>
  );
}

export function actForStep(stepID?: string): "decide" | "adjust" | "commit" {
  if (stepID === "welcome" || stepID === "plan") return "decide";
  if (stepID === "scenarios" || stepID === "core-set" || stepID === "resources" || stepID === "operating-mode" || stepID === "host") return "adjust";
  return "commit";
}
