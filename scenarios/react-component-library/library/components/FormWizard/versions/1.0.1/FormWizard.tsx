/**
 * @libraryId react-component-library:FormWizard
 * @displayName Form Wizard
 * @description Resumable step-based form controller with validation boundaries
 * @version 1.0.1
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
  onComplete?: () => void;
  onStepChange?: (index: number) => void;
  draftKey?: string;
  className?: string;
}

export default function FormWizard({
  steps = [],
  initialStep = 0,
  onComplete,
  onStepChange,
  draftKey,
  className = "",
}: FormWizardProps) {
  const [index, setIndex] = React.useState(() => {
    if (!draftKey) return initialStep;
    const saved = Number(sessionStorage.getItem(`rcl-wizard:${draftKey}`));
    return Number.isFinite(saved) && saved >= 0 && saved < steps.length ? saved : initialStep;
  });
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
      <nav aria-label="Wizard steps">
        {steps.map((item, i) => (
          <button
            type="button"
            key={item.id}
            aria-current={i === index ? "step" : undefined}
            onClick={() => i < index && move(i)}
          >
            {i + 1}. {item.title}
          </button>
        ))}
      </nav>
      {step && (
        <>
          <h2>{step.title}</h2>
          <div>{step.content}</div>
          <footer>
            <button type="button" onClick={() => move(index - 1)} disabled={index === 0}>
              Back
            </button>
            {index < steps.length - 1 ? (
              <button type="button" onClick={() => move(index + 1)}>
                Continue
              </button>
            ) : (
              <button type="button" onClick={onComplete}>
                Save
              </button>
            )}
          </footer>
        </>
      )}
    </section>
  );
}
