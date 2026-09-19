// DOC: docs/internal/ERROR-SEMANTICS.md
// DOC: docs/internal/COHERENCE-NOTES.md
import { AlertCircle, RefreshCw, WifiOff } from "lucide-react";
import { Button } from "./ui/button";
import { categorizeError } from "../lib/errors";

interface ErrorAlertProps {
  error: Error;
  onRetry?: () => void;
  compact?: boolean;
}

export function ErrorAlert({ error, onRetry, compact }: ErrorAlertProps) {
  const { category, userMessage, guidance } = categorizeError(error);
  const Icon = category === "connection" ? WifiOff : AlertCircle;

  if (compact) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-[var(--error-border)] bg-[var(--error-bg)] px-3 py-2 text-xs text-[var(--error-text)]">
        <Icon className="h-3.5 w-3.5 shrink-0" />
        <span>{userMessage}</span>
        {onRetry && (
          <button onClick={onRetry} className="ml-auto text-[var(--error-link)] underline hover:text-[var(--error-link-hover)]">
            Retry
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-[var(--error-border)] bg-[var(--error-bg)] p-4">
      <div className="flex items-start gap-3">
        <Icon className="mt-0.5 h-5 w-5 shrink-0 text-[var(--error-icon)]" />
        <div className="flex-1">
          <p className="text-sm font-medium text-[var(--error-text)]">{userMessage}</p>
          <p className="mt-1 text-xs text-[var(--text-muted)]">{guidance}</p>
        </div>
        {onRetry && (
          <Button size="sm" variant="outline" onClick={onRetry}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Retry
          </Button>
        )}
      </div>
    </div>
  );
}
