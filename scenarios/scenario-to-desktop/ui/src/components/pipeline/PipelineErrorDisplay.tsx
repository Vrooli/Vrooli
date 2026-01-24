/**
 * Shared error display component for pipeline operations.
 * Provides consistent error UI with copy and retry functionality.
 */

import { useCallback, useMemo } from "react";
import { XCircle, Copy, RefreshCw, Lightbulb } from "lucide-react";
import { Button } from "../ui/button";
import { writeToClipboard } from "../../lib/browser";
import { getRecoverySuggestions } from "../../services/pipeline.service";
import type { PipelineErrorInfo } from "../../store/pipelineTypes";

interface PipelineErrorDisplayProps {
  title?: string;
  errorMessage: string;
  suggestion?: string | null;
  onRetry?: () => void;
  onCopy?: () => void;
  className?: string;
}

interface PipelineErrorRecoveryProps {
  /** Structured error info from pipeline store */
  errorInfo: PipelineErrorInfo;
  /** Callback when retry button is clicked */
  onRetry?: () => void;
  /** Callback when dismiss button is clicked */
  onDismiss?: () => void;
  /** Additional CSS class */
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

/**
 * Enhanced error recovery component that uses categorized suggestions.
 * Wires the getRecoverySuggestions service function to the UI.
 */
export function PipelineErrorRecovery({
  errorInfo,
  onRetry,
  onDismiss,
  className = "",
}: PipelineErrorRecoveryProps) {
  // Get recovery suggestions - prefer from errorInfo, fall back to category-based
  const suggestions = useMemo(() => {
    // If errorInfo already has suggestions, use them
    if (errorInfo.suggestions && errorInfo.suggestions.length > 0) {
      return errorInfo.suggestions;
    }
    // Otherwise, get suggestions based on category
    if (errorInfo.category) {
      return getRecoverySuggestions(errorInfo.category);
    }
    return [];
  }, [errorInfo.suggestions, errorInfo.category]);

  const handleCopy = useCallback(async () => {
    const fullMessage = `Error: ${errorInfo.message}\nCategory: ${errorInfo.category ?? "unknown"}\nSuggestions:\n${suggestions.map((s) => `- ${s}`).join("\n")}`;
    await writeToClipboard(fullMessage);
  }, [errorInfo, suggestions]);

  return (
    <div
      className={`space-y-3 rounded-lg border border-red-900 bg-red-950/30 p-4 ${className}`}
    >
      {/* Error header */}
      <div className="flex items-start gap-3">
        <XCircle className="mt-0.5 h-5 w-5 flex-shrink-0 text-red-400" />
        <div className="flex-1 space-y-1">
          <p className="font-medium text-red-200">{errorInfo.message}</p>
          {errorInfo.category && (
            <p className="text-xs text-red-300/60">Error category: {errorInfo.category}</p>
          )}
        </div>
      </div>

      {/* Recovery suggestions */}
      {suggestions.length > 0 && (
        <div className="space-y-2 rounded-md bg-amber-950/30 p-3">
          <div className="flex items-center gap-2 text-amber-200">
            <Lightbulb className="h-4 w-4" />
            <span className="text-sm font-medium">Suggested actions</span>
          </div>
          <ul className="space-y-1 pl-6 text-sm text-amber-100/80">
            {suggestions.map((suggestion, index) => (
              <li key={index} className="list-disc">
                {suggestion}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Action buttons */}
      <div className="flex gap-2 pt-1">
        {onRetry && (
          <Button
            variant="outline"
            size="sm"
            onClick={onRetry}
            className="gap-1.5 border-red-700 text-red-200 hover:bg-red-900/30"
          >
            <RefreshCw className="h-3 w-3" />
            Retry
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          className="gap-1.5 text-red-300/70 hover:text-red-200"
        >
          <Copy className="h-3 w-3" />
          Copy details
        </Button>
        {onDismiss && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onDismiss}
            className="ml-auto text-red-300/70 hover:text-red-200"
          >
            Dismiss
          </Button>
        )}
      </div>
    </div>
  );
}
