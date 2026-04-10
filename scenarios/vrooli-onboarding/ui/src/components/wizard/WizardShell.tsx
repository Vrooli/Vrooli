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
    <div className="flex min-h-[calc(100vh-3.5rem)] flex-col bg-slate-950 text-slate-50" data-testid="wizard-shell">
      {/* Step indicator */}
      <section className="border-b border-white/10 bg-white/5 px-2 py-2 sm:px-6 sm:py-4" aria-label="Wizard progress">
        <div className="mx-auto max-w-3xl">
          {/* Mobile: compact current-step label + dot indicators */}
          <div className="flex items-center justify-between sm:hidden mb-2">
            <span className="text-xs font-medium text-slate-50">
              Step {currentStep + 1}: {STEP_LABELS[currentStep]}
            </span>
            <span className="text-xs text-slate-300">
              {currentStep + 1}/{STEP_LABELS.length}
            </span>
          </div>
          <div className="flex items-center justify-center gap-2 sm:hidden mb-2" aria-label="Step progress" role="list">
            {STEP_LABELS.map((label, i) => {
              const isCompleted = i < currentStep;
              const isClickable = isCompleted && onGoToStep;
              return isClickable ? (
                <button
                  key={label}
                  type="button"
                  role="listitem"
                  className="h-2 w-2 rounded-full bg-emerald-500 transition-colors cursor-pointer hover:bg-emerald-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50"
                  aria-label={`Go back to ${label} (completed)`}
                  onClick={() => onGoToStep(i)}
                />
              ) : (
                <div
                  key={label}
                  role="listitem"
                  className={cn(
                    "h-2 w-2 rounded-full transition-colors",
                    isCompleted
                      ? "bg-emerald-500"
                      : i === currentStep
                        ? "bg-slate-50"
                        : "bg-white/20"
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
                  <button
                    type="button"
                    className={cn(
                      "flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors",
                      isCompleted
                        ? "bg-emerald-500 text-white"
                        : i === currentStep
                          ? "bg-slate-50 text-slate-900"
                          : "bg-white/10 text-slate-300",
                      isClickable
                        ? "cursor-pointer hover:bg-emerald-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50"
                        : "cursor-default"
                    )}
                    data-testid={`step-indicator-${i}`}
                    onClick={() => isClickable && onGoToStep(i)}
                    tabIndex={isClickable ? 0 : -1}
                    aria-label={isClickable ? `Go back to ${label}` : undefined}
                    aria-hidden={!isClickable ? "true" : undefined}
                  >
                    {isCompleted ? "\u2713" : i + 1}
                  </button>
                  <span
                    className={cn(
                      "text-sm",
                      i === currentStep ? "text-slate-50 font-medium" : "text-slate-300"
                    )}
                  >
                    {label}
                    <span className="sr-only">
                      {isCompleted ? " (completed)" : i === currentStep ? " (current)" : ""}
                    </span>
                  </span>
                  {i < STEP_LABELS.length - 1 && (
                    <div className="mx-2 h-px w-8 bg-white/10" aria-hidden="true" />
                  )}
                </li>
              );
            })}
          </ol>
          {/* Progress bar */}
          <div className="h-1 w-full rounded-full bg-white/10" role="progressbar" aria-valuenow={currentStep} aria-valuemin={0} aria-valuemax={STEP_LABELS.length - 1} aria-label="Wizard progress">
            <div
              className="h-1 rounded-full bg-emerald-500 transition-all duration-300"
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
      <div className="sticky bottom-0 border-t border-white/10 bg-slate-950/95 backdrop-blur-sm px-3 py-2.5 sm:static sm:bg-white/5 sm:px-6 sm:py-4" style={{ paddingBottom: "max(0.625rem, env(safe-area-inset-bottom))" }}>
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
