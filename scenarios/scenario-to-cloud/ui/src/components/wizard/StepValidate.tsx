import { useEffect } from "react";
import { CheckCircle2, AlertTriangle, AlertCircle, Play } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { LoadingState } from "../ui/spinner";
import { selectors } from "../../consts/selectors";
import type { useDeployment } from "../../hooks/useDeployment";

interface StepValidateProps {
  deployment: ReturnType<typeof useDeployment>;
}

export function StepValidate({ deployment }: StepValidateProps) {
  const {
    validationIssues,
    validationError,
    isValidating,
    validate,
    parsedManifest,
  } = deployment;

  // Auto-validate when entering this step
  useEffect(() => {
    if (validationIssues === null && !validationError && !isValidating) {
      validate();
    }
  }, []); // Only on mount

  const errorCount = validationIssues?.filter((i) => i.severity === "error").length ?? 0;
  const warnCount = validationIssues?.filter((i) => i.severity === "warn").length ?? 0;

  const hasNoIssues = validationIssues !== null && validationIssues.length === 0;
  const hasOnlyWarnings = validationIssues !== null && errorCount === 0 && warnCount > 0;
  const hasErrors = errorCount > 0;

  return (
    <div className="space-y-6">
      {/* Validate Button */}
      <div className="flex items-center gap-3">
        <Button
          data-testid={selectors.manifest.validateButton}
          onClick={validate}
          disabled={isValidating || !parsedManifest.ok}
        >
          <Play className="h-4 w-4 mr-1.5" />
          {isValidating ? "Validating..." : "Run Validation"}
        </Button>
        {!parsedManifest.ok && (
          <span className="text-sm text-red-400">Fix JSON errors first</span>
        )}
      </div>

      {/* Loading State */}
      {isValidating && (
        <LoadingState message="Validating manifest..." />
      )}

      {/* Validation Error */}
      {validationError && (
        <Alert variant="error" title="Validation Failed">
          {validationError}
        </Alert>
      )}

      {/* Results */}
      {validationIssues !== null && !isValidating && (
        <div data-testid={selectors.manifest.validateResult} className="space-y-4">
          {/* Summary */}
          {hasNoIssues && (
            <Alert variant="success" title="Manifest is Valid">
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4" />
                No issues found. Your manifest is ready for planning.
              </div>
            </Alert>
          )}

          {hasOnlyWarnings && (
            <Alert variant="warning" title="Manifest Has Warnings">
              <div className="flex items-center gap-2">
                <AlertTriangle className="h-4 w-4" />
                Found {warnCount} warning{warnCount !== 1 ? "s" : ""}. You can proceed, but review the issues below.
              </div>
            </Alert>
          )}

          {hasErrors && (
            <Alert variant="error" title="Manifest Has Errors">
              <div className="flex items-center gap-2">
                <AlertCircle className="h-4 w-4" />
                Found {errorCount} error{errorCount !== 1 ? "s" : ""} and {warnCount} warning{warnCount !== 1 ? "s" : ""}. Fix errors before proceeding.
              </div>
            </Alert>
          )}

          {/* Issue List */}
          {validationIssues.length > 0 && (
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-slate-300">Issues Found</h4>
              <ul className="space-y-2">
                {validationIssues.map((issue, index) => (
                  <li
                    key={`${issue.severity}-${issue.path}-${index}`}
                    className="p-3 rounded-lg bg-slate-800/50 border border-slate-700"
                  >
                    <div className="flex items-start gap-2">
                      {issue.severity === "error" ? (
                        <AlertCircle className="h-4 w-4 text-red-400 flex-shrink-0 mt-0.5" />
                      ) : (
                        <AlertTriangle className="h-4 w-4 text-amber-400 flex-shrink-0 mt-0.5" />
                      )}
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span
                            className={`text-xs font-medium px-1.5 py-0.5 rounded ${
                              issue.severity === "error"
                                ? "bg-red-500/20 text-red-300"
                                : "bg-amber-500/20 text-amber-300"
                            }`}
                          >
                            {issue.severity.toUpperCase()}
                          </span>
                          <code className="text-xs text-slate-400 font-mono">
                            {issue.path}
                          </code>
                        </div>
                        <p className="mt-1 text-sm text-slate-200">
                          {issue.message}
                        </p>
                        {issue.hint && (
                          <p className="mt-1 text-xs text-slate-400">
                            Hint: {issue.hint}
                          </p>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
