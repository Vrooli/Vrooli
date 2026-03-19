import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, AlertTriangle, XCircle, CheckCircle2, HelpCircle, X, ChevronLeft } from "lucide-react";
import { validateConfig } from "../../lib/api";
import { formatQueryError } from "../../lib/formatQueryError";
import { Button } from "../ui/button";

interface StepReviewProps {
  selected: Set<string>;
  onRemove?: (name: string) => void;
  onGoBack?: () => void;
}

export function StepReview({ selected, onRemove, onGoBack }: StepReviewProps) {
  const selectedArray = useMemo(() => Array.from(selected), [selected]);

  const resourcesMap = useMemo(() => {
    const map: Record<string, { enabled: boolean; name: string }> = {};
    for (const name of selectedArray) {
      map[name] = { enabled: true, name };
    }
    return map;
  }, [selectedArray]);

  const { data: result, isLoading: loading, error: queryError } = useQuery({
    queryKey: ["validation", selected.size],
    queryFn: () => validateConfig(resourcesMap),
    enabled: selectedArray.length > 0,
  });

  const error = formatQueryError(queryError, "Validation failed");

  return (
    <div data-testid="step-review">
      <h1 className="text-2xl font-semibold">Review Configuration</h1>
      <p className="mt-2 text-slate-300">
        Review your selected resources before generating the configuration.
      </p>

      {/* Selected resources list */}
      <div className="mt-6">
        <h2 className="mb-3 text-sm font-medium uppercase tracking-wider text-slate-300">
          Selected Resources ({selectedArray.length})
        </h2>
        {selectedArray.length === 0 ? (
          <div className="flex flex-col items-center py-6 text-center" data-testid="review-empty-state">
            <p className="text-slate-300">No resources selected.</p>
            {onGoBack && (
              <Button
                variant="outline"
                size="sm"
                onClick={onGoBack}
                className="mt-3"
                data-testid="review-go-back"
                aria-label="Go back to select resources"
              >
                <ChevronLeft className="mr-1 h-3 w-3" aria-hidden="true" />
                Select Resources
              </Button>
            )}
          </div>
        ) : (
          <div className="flex flex-wrap gap-2" data-testid="review-resource-chips">
            {selectedArray.map((name) => (
              <span
                key={name}
                className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-sm text-emerald-400"
              >
                {name}
                {onRemove && (
                  <button
                    type="button"
                    onClick={() => onRemove(name)}
                    className="ml-0.5 rounded-full p-0.5 hover:bg-emerald-500/20 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500 transition-colors"
                    aria-label={`Remove ${name}`}
                    data-testid={`remove-resource-${name}`}
                  >
                    <X className="h-3 w-3" aria-hidden="true" />
                  </button>
                )}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* Validation results */}
      <div className="mt-6 rounded-xl border border-white/10 bg-white/5 p-4">
        <div className="flex items-center gap-2 mb-3">
          <h2 className="text-sm font-medium uppercase tracking-wider text-slate-300">
            Validation
          </h2>
          <span
            className="group relative"
            data-testid="validation-help"
          >
            <HelpCircle className="h-3.5 w-3.5 text-slate-300 cursor-help" aria-hidden="true" />
            <span
              role="tooltip"
              className="pointer-events-none absolute bottom-full left-1/2 z-10 mb-2 -translate-x-1/2 whitespace-nowrap rounded-lg bg-slate-800 px-3 py-1.5 text-xs text-slate-200 opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
            >
              Checks for dependency conflicts, port collisions, and missing prerequisites
            </span>
            <span className="sr-only">Checks for dependency conflicts, port collisions, and missing prerequisites</span>
          </span>
        </div>

        {loading && (
          <div className="flex items-center gap-2 text-slate-300" data-testid="validation-loading" role="status">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            Validating configuration...
          </div>
        )}

        {error && (
          <div className="flex items-center gap-2 text-red-400" data-testid="validation-error" role="alert">
            <XCircle className="h-4 w-4" aria-hidden="true" />
            {error}
          </div>
        )}

        {!loading && !error && result && (
          <div className="space-y-3">
            {result.valid ? (
              <div className="flex items-center gap-2 text-emerald-400" data-testid="validation-success">
                <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
                Configuration is valid
              </div>
            ) : (
              <div className="flex items-center gap-2 text-red-400" data-testid="validation-invalid">
                <XCircle className="h-4 w-4" aria-hidden="true" />
                Configuration has issues
              </div>
            )}

            {result.errors && result.errors.length > 0 && (
              <div className="space-y-1">
                {result.errors.map((msg, i) => (
                  <div key={i} className="flex items-start gap-2 text-sm text-red-400">
                    <XCircle className="mt-0.5 h-3 w-3 shrink-0" aria-hidden="true" />
                    {msg}
                  </div>
                ))}
              </div>
            )}

            {result.warnings && result.warnings.length > 0 && (
              <div className="space-y-1">
                {result.warnings.map((msg, i) => (
                  <div key={i} className="flex items-start gap-2 text-sm text-yellow-400">
                    <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0" aria-hidden="true" />
                    {msg}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {!loading && !error && !result && selectedArray.length === 0 && (
          <p className="text-sm text-slate-300">Select resources to validate.</p>
        )}
      </div>
    </div>
  );
}
