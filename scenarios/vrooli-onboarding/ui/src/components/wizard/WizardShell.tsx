import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import FormWizard from "@vrooli/react-component-library/FormWizard";
import { cn } from "../../lib/utils";
import type { V2Step } from "../../types";

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
  stepContents?: React.ReactNode[];
  target?: string;
  onTargetChange?: (target: string) => void;
  targetOptions?: Array<{ id: string; name?: string; status?: string }>;
}

export function WizardShell({
  currentStep,
  steps,
  onNext,
  onPrev,
  onGoToStep,
  nextDisabled = false,
  nextLabel = "Next",
  showPrev = true,
  showNext = true,
  children,
  stepContents,
  target = "local",
  onTargetChange,
  targetOptions = [],
}: WizardShellProps) {
  return (
    <div
      className="flex min-h-full flex-col bg-surface text-foreground"
      data-testid="wizard-shell"
    >
      <div className="border-b border-muted bg-surface px-3 py-3 sm:px-6" data-testid="wizard-target-chrome">
        <div className="mx-auto flex max-w-3xl flex-wrap items-center gap-3">
          <label htmlFor="wizard-target" className="text-sm font-medium">Configuring target</label>
          <Input id="wizard-target" value={target} onChange={(event) => onTargetChange?.(event.target.value)} list="wizard-target-options" aria-describedby="wizard-target-help" data-testid="wizard-target" className="min-w-0 flex-1" />
          <datalist id="wizard-target-options">{targetOptions.map((option) => <option key={option.id} value={option.id}>{option.name || option.id}</option>)}</datalist>
          <span id="wizard-target-help" className="text-xs text-muted">Use <code>local</code> or a registered node id.</span>
        </div>
      </div>
      {/* Step indicator */}
      <section
        className="border-b border-muted bg-surface-muted px-2 py-2 sm:px-6 sm:py-4"
        aria-label="Wizard progress"
      >
        <div className="mx-auto max-w-3xl">
          {/* Mobile: compact current-step label + dot indicators */}
          <div className="flex items-center justify-between sm:hidden mb-2">
            <span className="text-xs font-medium text-foreground">
              Step {currentStep + 1}: {steps[currentStep]?.title ?? "Loading"}
            </span>
            <span className="text-xs text-muted">
              {currentStep + 1}/{steps.length}
            </span>
          </div>
          <div
            className="flex max-w-full flex-wrap items-center justify-center gap-2 sm:hidden mb-2"
            aria-label="Step progress"
            role="list"
          >
            {steps.map((step, i) => {
              const label = step.title;
              const isCompleted = i < currentStep;
              const isClickable = isCompleted && onGoToStep;
              return isClickable ? (
                <Button
                  variant="ghost"
                  key={label}
                  type="button"
                  className="h-11 w-11 shrink-0 rounded-full bg-transparent p-0 transition-colors cursor-pointer hover:bg-surface-subtle focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus/50"
                  aria-label={`Go back to ${label} (completed)`}
                  onClick={() => onGoToStep(i)}
                >
                  <span
                    aria-hidden="true"
                    className="h-2 w-2 rounded-full bg-primary"
                  />
                </Button>
              ) : (
                <div
                  key={label}
                  role="listitem"
                  className={cn(
                    "h-2 w-2 rounded-full transition-colors",
                    isCompleted
                      ? "bg-primary"
                      : i === currentStep
                        ? "bg-foreground"
                        : "bg-border-muted",
                  )}
                  aria-label={`${label}${isCompleted ? " (completed)" : i === currentStep ? " (current)" : ""}`}
                />
              );
            })}
          </div>

          {/* Desktop: full step labels with numbers */}
          <ol
            className="hidden sm:flex items-center justify-between mb-3"
            aria-label="Wizard steps"
            data-testid="wizard-steps-desktop"
          >
            {steps.map((step, i) => {
              const label = step.title;
              const isCompleted = i < currentStep;
              const isClickable = isCompleted && onGoToStep;
              return (
                <li
                  key={label}
                  className="flex items-center gap-2"
                  aria-current={i === currentStep ? "step" : undefined}
                >
                  <Button
                    variant="ghost"
                    type="button"
                    className={cn(
                      "flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors",
                      isCompleted
                        ? "bg-primary text-foreground"
                        : i === currentStep
                          ? "bg-foreground text-foreground-strong"
                          : "bg-surface-subtle text-muted",
                      isClickable
                        ? "cursor-pointer hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus/50"
                        : "cursor-default",
                    )}
                    data-testid={`step-indicator-${i}`}
                    onClick={() => isClickable && onGoToStep(i)}
                    tabIndex={isClickable ? 0 : -1}
                    aria-label={isClickable ? `Go back to ${label}` : undefined}
                    aria-hidden={!isClickable ? "true" : undefined}
                  >
                    {isCompleted ? "\u2713" : i + 1}
                  </Button>
                  <span className="sr-only">
                    {label}
                    {isCompleted
                      ? " (completed)"
                      : i === currentStep
                        ? " (current)"
                        : ""}
                  </span>
                  {i < steps.length - 1 && (
                    <div
                      className="mx-1 h-px w-4 bg-surface-subtle lg:mx-2 lg:w-6"
                      aria-hidden="true"
                    />
                  )}
                </li>
              );
            })}
          </ol>
          {/* Progress bar */}
          <div
            className="h-1 w-full rounded-full bg-surface-subtle"
            role="progressbar"
            aria-valuenow={currentStep}
            aria-valuemin={0}
            aria-valuemax={Math.max(steps.length - 1, 0)}
            aria-label="Wizard progress"
          >
            <div
              className="h-1 rounded-full bg-primary transition-all duration-300"
              style={{
                width: `${steps.length > 1 ? (currentStep / (steps.length - 1)) * 100 : 0}%`,
              }}
              data-testid="progress-bar"
            />
          </div>
        </div>
      </section>

      {/* Content - reduced padding on mobile, extra bottom padding for sticky nav */}
      <div className="flex-1 overflow-auto px-3 py-4 sm:px-6 sm:py-8 pb-20 sm:pb-8">
        <div className="mx-auto max-w-3xl">
          <FormWizard
            key={steps.map((step) => step.id).join("/")}
            steps={steps.map((step, index) => ({
              id: step.id,
              title: step.title,
              content: stepContents?.[index] ?? (index === currentStep ? children : null),
            }))}
            initialStep={currentStep}
            activeStep={currentStep}
            onStepChange={(index) => {
              if (onGoToStep) {
                onGoToStep(index);
              } else if (index > currentStep) {
                onNext();
              } else if (index < currentStep) {
                onPrev();
              }
            }}
            draftKey="vrooli-onboarding"
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
            previousAriaLabel="Go to previous step"
            className="min-h-0"
          />
        </div>
      </div>
    </div>
  );
}
