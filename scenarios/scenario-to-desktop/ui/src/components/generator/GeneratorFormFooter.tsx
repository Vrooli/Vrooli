/**
 * Footer section for GeneratorForm.
 * Contains validation errors display and submit button.
 */

import { Button } from "../ui/button";
import { ValidationErrors, type ValidationError } from ".";

export interface GeneratorFormFooterProps {
  validationErrors: ValidationError[];
  onDismissErrors: () => void;
  isPending: boolean;
  isError: boolean;
  errorMessage: string | null;
  isUpdateMode: boolean;
}

export function GeneratorFormFooter({
  validationErrors,
  onDismissErrors,
  isPending,
  isError,
  errorMessage,
  isUpdateMode,
}: GeneratorFormFooterProps) {
  return (
    <>
      {/* Validation errors - shown above submit button */}
      <ValidationErrors
        errors={validationErrors}
        onDismiss={onDismissErrors}
      />

      <Button
        type="submit"
        className="w-full"
        disabled={isPending || validationErrors.length > 0}
      >
        {isPending
          ? "Generating..."
          : isUpdateMode
            ? "Update Desktop Application"
            : "Generate Desktop Application"}
      </Button>

      {isError && errorMessage && (
        <div className="rounded-lg bg-red-900/20 p-3 text-sm text-red-300">
          <strong>Error:</strong> {errorMessage}
        </div>
      )}
    </>
  );
}
