/**
 * Validation errors display component for the generator form.
 *
 * This is a pure presentation component. Validation logic lives in the domain layer
 * at `domain/generator.ts`. This component only handles display and user interaction.
 *
 * NOTE: For `validateFormInputs`, `ValidationError`, and `ValidateFormInputsParams`,
 * import directly from the domain layer (`../../domain/generator`) or from the barrel
 * export in the generator index file.
 */

import { AlertTriangle, X } from "lucide-react";
import { Button } from "../ui/button";
import type { ValidationError } from "../../domain/generator";

export interface ValidationErrorsProps {
  errors: ValidationError[];
  onDismiss?: () => void;
  className?: string;
}

export function ValidationErrors({ errors, onDismiss, className = "" }: ValidationErrorsProps) {
  if (errors.length === 0) return null;

  return (
    <div className={`rounded-lg border border-red-800/60 bg-red-950/30 p-4 ${className}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div className="space-y-2">
            <p className="text-sm font-medium text-red-200">
              Please fix the following {errors.length === 1 ? "issue" : "issues"} before generating:
            </p>
            <ul className="space-y-1 text-sm text-red-300">
              {errors.map((error) => (
                <li key={error.id} className="flex items-start gap-2">
                  <span className="text-red-400">•</span>
                  <span>{error.message}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
        {onDismiss && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onDismiss}
            className="text-red-400 hover:text-red-300 hover:bg-red-900/30 -mt-1 -mr-1"
          >
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  );
}
