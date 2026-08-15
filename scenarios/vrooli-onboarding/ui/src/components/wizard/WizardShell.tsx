import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import { STEP_LABELS } from "../../types";

interface WizardShellProps {
  currentStep: number;
  onNext: () => void;
  onPrev: () => void;
  onGoToStep?: (step: number) => void;
  nextDisabled?: boolean;
  nextLabel?: string;
  showPrev?: boolean;
  showNext?: boolean;
  children: React.ReactNode;
}

export function WizardShell({
  currentStep,
  onNext,
  onPrev,
  onGoToStep,
  nextDisabled = false,
  nextLabel = "Next",
  showPrev = true,
  showNext = true,
  children,
}: WizardShellProps) {
  return (
    <div className="flex min-h-full flex-col bg-surface text-foreground" data-testid="wizard-shell">
      {/* Step indicator */}
      <section className="border-b border-muted bg-surface-muted px-2 py-2 sm:px-6 sm:py-4" aria-label="Wizard progress">
        <div className="mx-auto max-w-3xl">
          {/* Mobile: compact current-step label + dot indicators */}
          <div className="flex items-center justify-between sm:hidden mb-2">
            <span className="text-xs font-medium text-foreground">
              Step {currentStep + 1}: {STEP_LABELS[currentStep]}
            </span>
            <span className="text-xs text-muted">
              {currentStep + 1}/{STEP_LABELS.length}
            </span>
          </div>
          <div className="flex max-w-full flex-wrap items-center justify-center gap-2 sm:hidden mb-2" aria-label="Step progress" role="list">
            {STEP_LABELS.map((label, i) => {
              const isCompleted = i < currentStep;
              const isClickable = isCompleted && onGoToStep;
              return isClickable ? (
                <Button variant="ghost"
                  key={label}
                  type="button"
                                  className="h-11 w-11 shrink-0 rounded-full bg-transparent p-0 transition-colors cursor-pointer hover:bg-surface-subtle focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus/50"
                                  aria-label={`Go back to ${label} (completed)`}
                                  onClick={() => onGoToStep(i)}
                                ><span aria-hidden="true" className="h-2 w-2 rounded-full bg-primary" /></Button>
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
                        : "bg-border-muted"
                  )}
                  aria-label={`${label}${isCompleted ? " (completed)" : i === currentStep ? " (current)" : ""}`}
                />
              );
            })}
          </div>

          {/* Desktop: full step labels with numbers */}
		  <ol className="hidden sm:flex items-center justify-between mb-3" aria-label="Wizard steps" data-testid="wizard-steps-desktop">
            {STEP_LABELS.map((label, i) => {
              const isCompleted = i < currentStep;
              const isClickable = isCompleted && onGoToStep;
              return (
                <li key={label} className="flex items-center gap-2" aria-current={i === currentStep ? "step" : undefined}>
                  <Button variant="ghost"
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
                        : "cursor-default"
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
					{label}{isCompleted ? " (completed)" : i === currentStep ? " (current)" : ""}
				  </span>
				  {i < STEP_LABELS.length - 1 && (
					<div className="mx-1 h-px w-4 bg-surface-subtle lg:mx-2 lg:w-6" aria-hidden="true" />
                  )}
                </li>
              );
            })}
          </ol>
          {/* Progress bar */}
          <div className="h-1 w-full rounded-full bg-surface-subtle" role="progressbar" aria-valuenow={currentStep} aria-valuemin={0} aria-valuemax={STEP_LABELS.length - 1} aria-label="Wizard progress">
            <div
              className="h-1 rounded-full bg-primary transition-all duration-300"
              style={{ width: `${(currentStep / (STEP_LABELS.length - 1)) * 100}%` }}
              data-testid="progress-bar"
            />
          </div>
        </div>
      </section>

      {/* Content - reduced padding on mobile, extra bottom padding for sticky nav */}
      <div className="flex-1 overflow-auto px-3 py-4 sm:px-6 sm:py-8 pb-20 sm:pb-8">
        <div className="mx-auto max-w-3xl">{children}</div>
      </div>

      {/* Navigation - sticky on mobile for thumb access */}
      <div className="sticky bottom-0 border-t border-muted bg-surface/95 backdrop-blur-sm px-3 py-2.5 sm:static sm:bg-surface-muted sm:px-6 sm:py-4" style={{ paddingBottom: "max(0.625rem, env(safe-area-inset-bottom))" }}>
        <div className="mx-auto flex max-w-3xl items-center justify-between">
          <div>
            {showPrev && currentStep > 0 && (
              <Button variant="outline" onClick={onPrev} data-testid="wizard-prev" aria-label="Go to previous step">
                <ChevronLeft className="mr-1 h-4 w-4" aria-hidden="true" />
                Back
              </Button>
            )}
          </div>
          <div>
            {showNext && (
              <Button onClick={onNext} disabled={nextDisabled} data-testid="wizard-next" aria-label={nextLabel}>
                {nextLabel}
                <ChevronRight className="ml-1 h-4 w-4" aria-hidden="true" />
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
