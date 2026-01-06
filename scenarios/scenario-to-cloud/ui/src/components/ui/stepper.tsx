import { Check } from "lucide-react";
import { cn } from "../../lib/utils";

export interface Step {
  id: string;
  label: string;
  description?: string;
}

export type StepperStatus = "pending" | "running" | "completed" | "failed" | "skipped";

interface StepperProps extends React.HTMLAttributes<HTMLElement> {
  steps: Step[];
  currentStep: number;
  onStepClick?: (stepIndex: number) => void;
  stepStates?: StepperStatus[];
  allowFutureClicks?: boolean;
}

export function Stepper({
  steps,
  currentStep,
  onStepClick,
  stepStates,
  allowFutureClicks,
  className,
  ...props
}: StepperProps) {
  return (
    <nav aria-label="Progress" className={cn("w-full", className)} {...props}>
      <ol className="flex items-center">
        {steps.map((step, index) => {
          const status = stepStates?.[index];
          const isCompleted = status ? status === "completed" : index < currentStep;
          const isCurrent = status ? status === "running" : index === currentStep;
          const isSkipped = status === "skipped";
          const isClickable = Boolean(
            onStepClick && (allowFutureClicks || index <= currentStep),
          );

          return (
            <li
              key={step.id}
              className={cn(
                "relative flex-1",
                index !== steps.length - 1 && "pr-8 sm:pr-12"
              )}
            >
              {/* Connector line */}
              {index !== steps.length - 1 && (
                <div
                  className="absolute top-4 left-0 -right-4 sm:-right-6 h-0.5 w-full"
                  aria-hidden="true"
                >
                  <div
                    className={cn(
                      "h-full transition-colors duration-300",
                      isCompleted ? "bg-emerald-500" : isSkipped ? "bg-slate-500/70" : "bg-slate-700"
                    )}
                  />
                </div>
              )}

              <button
                type="button"
                onClick={() => isClickable && onStepClick(index)}
                disabled={!isClickable}
                className={cn(
                  "group relative flex flex-col items-center",
                  isClickable && "cursor-pointer",
                  !isClickable && "cursor-default"
                )}
              >
                {/* Step circle */}
                <span
                  className={cn(
                    "relative z-10 flex h-8 w-8 items-center justify-center rounded-full border-2 transition-all duration-300",
                    isCompleted && "bg-emerald-500 border-emerald-500",
                    isCurrent && "bg-slate-900 border-blue-500 ring-4 ring-blue-500/20",
                    isSkipped && "bg-slate-800 border-slate-500",
                    !isCompleted && !isCurrent && !isSkipped && "bg-slate-900 border-slate-600",
                    isClickable && !isCurrent && "group-hover:border-slate-400"
                  )}
                >
                  {isCompleted ? (
                    <Check className="h-4 w-4 text-white" />
                  ) : (
                    <span
                      className={cn(
                        "text-sm font-medium",
                        isCurrent ? "text-blue-400" : isSkipped ? "text-slate-300" : "text-slate-400"
                      )}
                    >
                      {index + 1}
                    </span>
                  )}
                </span>

                {/* Step label */}
                <span
                  className={cn(
                    "mt-2 text-xs font-medium text-center transition-colors hidden sm:block",
                    isCompleted && "text-emerald-400",
                    isCurrent && "text-blue-400",
                    isSkipped && "text-slate-400",
                    !isCompleted && !isCurrent && !isSkipped && "text-slate-500"
                  )}
                >
                  {step.label}
                </span>
              </button>
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

interface VerticalStepperProps {
  steps: Step[];
  currentStep: number;
  onStepClick?: (stepIndex: number) => void;
  className?: string;
}

export function VerticalStepper({ steps, currentStep, onStepClick, className }: VerticalStepperProps) {
  return (
    <nav aria-label="Progress" className={cn("w-full", className)}>
      <ol className="space-y-4">
        {steps.map((step, index) => {
          const isCompleted = index < currentStep;
          const isCurrent = index === currentStep;
          const isClickable = onStepClick && index <= currentStep;

          return (
            <li key={step.id} className="relative">
              {/* Connector line */}
              {index !== steps.length - 1 && (
                <div
                  className="absolute left-4 top-8 -ml-px h-full w-0.5"
                  aria-hidden="true"
                >
                  <div
                    className={cn(
                      "h-full transition-colors duration-300",
                      isCompleted ? "bg-emerald-500" : "bg-slate-700"
                    )}
                  />
                </div>
              )}

              <button
                type="button"
                onClick={() => isClickable && onStepClick(index)}
                disabled={!isClickable}
                className={cn(
                  "group relative flex items-start gap-4 w-full text-left",
                  isClickable && "cursor-pointer",
                  !isClickable && "cursor-default"
                )}
              >
                {/* Step circle */}
                <span
                  className={cn(
                    "relative z-10 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full border-2 transition-all duration-300",
                    isCompleted && "bg-emerald-500 border-emerald-500",
                    isCurrent && "bg-slate-900 border-blue-500 ring-4 ring-blue-500/20",
                    !isCompleted && !isCurrent && "bg-slate-900 border-slate-600",
                    isClickable && !isCurrent && "group-hover:border-slate-400"
                  )}
                >
                  {isCompleted ? (
                    <Check className="h-4 w-4 text-white" />
                  ) : (
                    <span
                      className={cn(
                        "text-sm font-medium",
                        isCurrent ? "text-blue-400" : "text-slate-400"
                      )}
                    >
                      {index + 1}
                    </span>
                  )}
                </span>

                {/* Step content */}
                <div className="pt-0.5">
                  <span
                    className={cn(
                      "text-sm font-medium transition-colors",
                      isCompleted && "text-emerald-400",
                      isCurrent && "text-blue-400",
                      !isCompleted && !isCurrent && "text-slate-500"
                    )}
                  >
                    {step.label}
                  </span>
                  {step.description && (
                    <p className="mt-0.5 text-xs text-slate-500">
                      {step.description}
                    </p>
                  )}
                </div>
              </button>
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
