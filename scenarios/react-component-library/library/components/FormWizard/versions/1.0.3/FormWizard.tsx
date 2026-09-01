/**
 * @libraryId react-component-library:FormWizard
 * @displayName Form Wizard
 * @description Resumable step-based form controller with validation boundaries
 * @version 1.0.3
 * @tags ["forms","wizard","setup"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";

export interface WizardStep {
  id: string;
  title: string;
  content: React.ReactNode;
  validate?: () => boolean | Promise<boolean>;
}
export interface FormWizardProps {
  steps?: WizardStep[];
  initialStep?: number;
  activeStep?: number;
  onComplete?: () => void;
  onStepChange?: (index: number) => void;
  draftKey?: string;
  className?: string;
  showStepNavigation?: boolean;
  showHeading?: boolean;
  showFooter?: boolean;
  showPrevious?: boolean;
  showNext?: boolean;
  showSave?: boolean;
  nextLabel?: string;
  nextDisabled?: boolean;
  nextTestId?: string;
  previousTestId?: string;
  nextAriaLabel?: string;
  previousAriaLabel?: string;
}

export default function FormWizard({
  steps = [],
  initialStep = 0,
  activeStep,
  onComplete,
  onStepChange,
  draftKey,
  className = "",
  showStepNavigation = true,
  showHeading = true,
  showFooter = true,
  showPrevious = true,
  showNext = true,
  showSave = true,
  nextLabel = "Continue",
  nextDisabled = false,
  nextTestId,
  previousTestId,
  nextAriaLabel,
  previousAriaLabel,
}: FormWizardProps) {
  const [index, setIndex] = React.useState(() => {
    if (!draftKey) return initialStep;
    const saved = Number(sessionStorage.getItem(`rcl-wizard:${draftKey}`));
    return Number.isFinite(saved) && saved >= 0 && saved < steps.length ? saved : initialStep;
  });
  React.useEffect(() => {
    if (activeStep === undefined || activeStep === index) return;
    if (activeStep >= 0 && activeStep < steps.length) setIndex(activeStep);
  }, [activeStep, index, steps.length]);
  const step = steps[index];
  const move = async (next: number) => {
    if (next > index && step?.validate && !(await step.validate())) return;
    const bounded = Math.max(0, Math.min(next, steps.length - 1));
    setIndex(bounded);
    onStepChange?.(bounded);
    if (draftKey) sessionStorage.setItem(`rcl-wizard:${draftKey}`, String(bounded));
    if (bounded === steps.length - 1 && next > index) onComplete?.();
  };
  return (
    <section className={`rcl-component form-wizard ${className}`.trim()} aria-label="Form wizard">
      {showStepNavigation && (
        <nav aria-label="Wizard steps">
          {steps.map((item, i) => (
            <button
              type="button"
              key={item.id}
              aria-current={i === index ? "step" : undefined}
              onClick={() => {
                if (i < index) void move(i);
              }}
            >
              {i + 1}. {item.title}
            </button>
          ))}
        </nav>
      )}
      {step && (
        <>
          {showHeading && <h2>{step.title}</h2>}
          <div>{step.content}</div>
          {showFooter && (
            <footer>
              {showPrevious && (
                <button
                  type="button"
                  data-testid={previousTestId}
                  aria-label={previousAriaLabel}
                  onClick={() => {
                    void move(index - 1);
                  }}
                  disabled={index === 0}
                >
                  Back
                </button>
              )}
              {index < steps.length - 1 && showNext ? (
                <button
                  type="button"
                  data-testid={nextTestId}
                  aria-label={nextAriaLabel}
                  disabled={nextDisabled}
                  onClick={() => {
                    void move(index + 1);
                  }}
                >
                  {nextLabel}
                </button>
              ) : showSave ? (
                <button
                  type="button"
                  data-testid={nextTestId}
                  aria-label={nextAriaLabel}
                  onClick={() => {
                    onComplete?.();
                  }}
                >
                  Save
                </button>
              ) : null}
            </footer>
          )}
        </>
      )}
    </section>
  );
}
