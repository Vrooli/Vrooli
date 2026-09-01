/**
 * @libraryId react-component-library:FormWizard
 * @displayName Form Wizard
 * @description Resumable step-based form controller with validation boundaries
 * @version 1.1.0
 * @tags ["forms","wizard","setup"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:FormWizard */
import * as React from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { Button } from "@vrooli/react-component-library/Button/2";

/**
 * 1.1.0 — the wizard stops emitting naked buttons.
 *
 * Every control this component rendered was a bare `<button>` with no class,
 * no token and no layout: the step strip, and a footer holding `Back` and
 * `Save` as adjacent inline elements with no gap between them. In an adopting
 * app that has no global button styling — which is every app built on this
 * library, because the library is where button styling lives — the footer
 * rendered as the single run-on word `BackSave` under the form. It was
 * reported from a screenshot rather than from a test, because nothing here
 * asserted that the footer was legible.
 *
 * The fix is composition, not CSS: the footer is `Button`, so it inherits the
 * variant, the size, the pending spinner and the `nowrap` that stops a
 * two-word label breaking across lines. What is left in this file's own sheet
 * is only the arrangement — the gap, the alignment, the divider — which is
 * genuinely this component's to own.
 *
 * `showFooter` now defaults to `steps.length > 1` rather than `true`.
 *
 * A one-step wizard is not a wizard, and its footer was actively misleading: a
 * permanently disabled `Back`, beside a `Save` that only called `onComplete`
 * and could not advance anything. Both consumers found in adoption wrapped a
 * single form in a wizard purely to get the draft-resume behaviour, and then
 * had to look at a footer they never wanted. The default now matches what the
 * step count already says; a caller that wants a footer on one step passes
 * `showFooter` and gets it, unchanged.
 *
 * `footerActions` is the escape hatch for the case underneath that one: a form
 * whose primary action is its own — "Start onboarding", "Grant and push" — and
 * which wants the wizard's arrangement without the wizard's verbs.
 */

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
  /** Defaults to `steps.length > 1`: one step is a form, not a wizard. */
  showFooter?: boolean;
  showPrevious?: boolean;
  showNext?: boolean;
  showSave?: boolean;
  nextLabel?: string;
  nextDisabled?: boolean;
  nextTestId?: string;
  previousTestId?: string;
  previousLabel?: string;
  saveLabel?: string;
  saveDisabled?: boolean;
  /** Drives the footer primary's spinner and blocks a double submit. */
  pending?: boolean;
  nextAriaLabel?: string;
  previousAriaLabel?: string;
  /**
   * Replaces the generated primary. The `Back` control is still rendered when
   * `showPrevious` is set, so a caller supplying its own submit does not also
   * have to re-implement stepping backwards.
   */
  footerActions?: React.ReactNode;
  /** Sits at the footer's leading edge — a validation reason, a draft note. */
  footerNote?: React.ReactNode;
  testId?: string;
}

const styles = `
[data-rcl-wizard] { display: grid; gap: var(--space-md, 24px); min-inline-size: 0; }
[data-rcl-wizard-steps] {
  display: flex; flex-wrap: wrap; align-items: center;
  gap: var(--space-2xs, 8px); min-inline-size: 0;
}
[data-rcl-wizard-step] {
  display: inline-flex; align-items: center; gap: var(--space-2xs, 8px);
  min-block-size: var(--control-size-sm, 36px);
  padding-inline: var(--space-xs, 12px);
  border: var(--border-hairline, 1px) solid var(--color-border);
  border-radius: var(--radius-pill, 9999px);
  background: var(--color-surface);
  color: var(--color-muted-foreground);
  font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans));
  white-space: nowrap; cursor: pointer;
  transition: background var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2,0,0,1)), border-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2,0,0,1)), color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2,0,0,1));
}
[data-rcl-wizard-step]:disabled { cursor: default; opacity: var(--opacity-muted, .64); }
[data-rcl-wizard-step]:not(:disabled):hover { border-color: var(--color-primary); color: var(--color-foreground); }
[data-rcl-wizard-step]:focus-visible { outline: var(--border-strong, 2px) solid var(--color-focus); outline-offset: var(--space-3xs, 4px); }
[data-rcl-wizard-step][aria-current="step"] {
  border-color: color-mix(in srgb, var(--color-primary) 45%, var(--color-border));
  background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
  color: var(--color-foreground);
}
[data-rcl-wizard-step][data-complete="true"] { color: var(--color-foreground); }
[data-rcl-wizard-step-index] {
  display: inline-grid; place-items: center; flex: 0 0 auto;
  inline-size: 1.25rem; block-size: 1.25rem;
  border-radius: var(--radius-pill, 9999px);
  background: color-mix(in srgb, currentColor 14%, transparent);
  font-variant-numeric: tabular-nums; font-size: .75em;
}
[data-rcl-wizard-heading] {
  margin: 0;
  color: var(--color-foreground);
  font: var(--text-heading-sm, 600 var(--text-heading-sm-size) / var(--text-heading-sm-line) var(--font-sans));
  letter-spacing: -.01em;
}
[data-rcl-wizard-panel] { min-inline-size: 0; }
[data-rcl-wizard-footer] {
  display: flex; align-items: center; flex-wrap: wrap;
  gap: var(--space-2xs, 8px);
  padding-block-start: var(--space-sm, 16px);
  border-block-start: var(--border-hairline, 1px) solid var(--color-border);
}
[data-rcl-wizard-footer-note] {
  min-inline-size: 0; margin-inline-end: auto;
  color: var(--color-muted-foreground);
  font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans));
}
/* With no note, the leading edge still has to push the actions to the end. */
[data-rcl-wizard-footer] > [data-rcl-wizard-footer-spacer] { margin-inline-end: auto; }
@media (prefers-reduced-motion: reduce) { [data-rcl-wizard-step] { transition: none; } }
`;

export function FormWizard({
  steps = [],
  initialStep = 0,
  activeStep,
  onComplete,
  onStepChange,
  draftKey,
  className = "",
  showStepNavigation = true,
  showHeading = true,
  showFooter,
  showPrevious = true,
  showNext = true,
  showSave = true,
  nextLabel = "Continue",
  nextDisabled = false,
  nextTestId,
  previousTestId,
  previousLabel = "Back",
  saveLabel = "Save",
  saveDisabled = false,
  pending = false,
  nextAriaLabel,
  previousAriaLabel,
  footerActions,
  footerNote,
  testId,
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

  const isLastStep = index >= steps.length - 1;
  const footerVisible = showFooter ?? steps.length > 1;

  return (
    <section
      className={`rcl-component form-wizard ${className}`.trim()}
      aria-label="Form wizard"
      data-rcl-wizard="true"
      data-testid={testId ?? "forms.form-wizard"}
    >
      <StyleSheet name="form-wizard-1-1" css={styles} />
      {showStepNavigation && steps.length > 1 && (
        <nav aria-label="Wizard steps" data-rcl-wizard-steps>
          {steps.map((item, i) => (
            <button
              type="button"
              key={item.id}
              data-rcl-wizard-step
              data-complete={i < index ? "true" : undefined}
              aria-current={i === index ? "step" : undefined}
              // Only completed steps are reachable. Jumping forward would skip
              // the `validate` boundary the step behind it declared.
              disabled={i >= index}
              onClick={() => {
                if (i < index) void move(i);
              }}
            >
              <span data-rcl-wizard-step-index aria-hidden="true">
                {i + 1}
              </span>
              {item.title}
            </button>
          ))}
        </nav>
      )}
      {step && (
        <>
          {showHeading && <h2 data-rcl-wizard-heading>{step.title}</h2>}
          <div data-rcl-wizard-panel>{step.content}</div>
          {footerVisible && (
            <footer data-rcl-wizard-footer>
              {footerNote ? (
                <span data-rcl-wizard-footer-note>{footerNote}</span>
              ) : (
                <span data-rcl-wizard-footer-spacer aria-hidden="true" />
              )}
              {showPrevious && (
                <Button
                  variant="secondary"
                  data-testid={previousTestId}
                  aria-label={previousAriaLabel}
                  onClick={() => {
                    void move(index - 1);
                  }}
                  disabled={index === 0 || pending}
                >
                  {previousLabel}
                </Button>
              )}
              {footerActions ??
                (!isLastStep && showNext ? (
                  <Button
                    data-testid={nextTestId}
                    aria-label={nextAriaLabel}
                    disabled={nextDisabled}
                    pending={pending}
                    onClick={() => {
                      void move(index + 1);
                    }}
                  >
                    {nextLabel}
                  </Button>
                ) : isLastStep && showSave ? (
                  <Button
                    data-testid={nextTestId}
                    aria-label={nextAriaLabel}
                    disabled={saveDisabled}
                    pending={pending}
                    onClick={() => {
                      onComplete?.();
                    }}
                  >
                    {saveLabel}
                  </Button>
                ) : null)}
            </footer>
          )}
        </>
      )}
    </section>
  );
}

export default FormWizard;
