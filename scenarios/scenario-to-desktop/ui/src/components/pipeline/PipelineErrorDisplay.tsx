/**
 * Shared error display component for pipeline operations.
 * Provides consistent error UI with copy and retry functionality.
 */

import { useCallback } from "react";
import { XCircle, Copy } from "lucide-react";
import { Button } from "../ui/button";
import { writeToClipboard } from "../../lib/browser";

interface PipelineErrorDisplayProps {
  title?: string;
  errorMessage: string;
  suggestion?: string | null;
  onRetry?: () => void;
  onCopy?: () => void;
  className?: string;
}

/**
 * Suggests recovery steps based on common error patterns.
 */
export function suggestRecovery(errorMessage: string, scenarioName?: string): string | null {
  if (errorMessage.includes("not found") || errorMessage.includes("404")) {
    return scenarioName
      ? `Ensure the scenario '${scenarioName}' exists in /scenarios/ first.`
      : "Ensure the scenario exists in /scenarios/ first.";
  }
  if (errorMessage.includes("ui/dist") || errorMessage.includes("UI not built")) {
    return scenarioName
      ? `Build the scenario UI first: cd scenarios/${scenarioName}/ui && npm run build.`
      : "Build the scenario UI first.";
  }
  if (errorMessage.includes("permission") || errorMessage.includes("EACCES")) {
    return "Check file permissions in the scenarios directory.";
  }
  if (errorMessage.includes("ENOSPC") || errorMessage.includes("no space")) {
    return "Free up disk space and try again.";
  }
  if (errorMessage.includes("port") || errorMessage.includes("EADDRINUSE")) {
    return "Another process is using the required port. Stop it or change ports.";
  }
  return null;
}

/**
 * Displays pipeline errors with copy and retry functionality.
 */
export function PipelineErrorDisplay({
  title = "Operation failed",
  errorMessage,
  suggestion,
  onRetry,
  onCopy,
  className = "",
}: PipelineErrorDisplayProps) {
  const handleCopy = useCallback(async () => {
    await writeToClipboard(errorMessage);
    onCopy?.();
  }, [errorMessage, onCopy]);

  return (
    <div className={`space-y-2 rounded-lg border border-red-900 bg-red-950/20 p-3 text-xs text-red-200 ${className}`}>
      <div className="flex items-center gap-2 text-red-300">
        <XCircle className="h-3 w-3" />
        <span>{title}</span>
      </div>
      <pre className="max-h-32 overflow-y-auto whitespace-pre-wrap font-mono text-[11px] text-red-200/80">
        {errorMessage}
      </pre>
      {suggestion && <p className="text-yellow-200">{suggestion}</p>}
      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={handleCopy} className="gap-1">
          <Copy className="h-3 w-3" />
          Copy error
        </Button>
        {onRetry && (
          <Button variant="outline" size="sm" onClick={onRetry}>
            Retry
          </Button>
        )}
      </div>
    </div>
  );
}

/**
 * Compact inline error for smaller spaces.
 */
export function InlineError({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-1 text-red-400">
        <XCircle className="h-4 w-4" />
        <span className="text-sm">{message}</span>
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  );
}
