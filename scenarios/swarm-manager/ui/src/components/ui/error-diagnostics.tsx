/**
 * Error Diagnostics Panel
 *
 * A shared component that renders collapsible error diagnostic information
 * directly on screen. Designed for environments where browser dev tools
 * are unavailable (e.g. TV browsers, kiosk devices).
 *
 * Used by ErrorBoundary, PageErrorBoundary, and CanvasErrorBoundary to
 * provide consistent diagnostic output across all error boundary levels.
 *
 * Collapsed by default for clean UX; expandable for debugging.
 */

import { useCallback, useRef, useState } from "react";
import { ChevronDown, ChevronRight, Check, Copy } from "lucide-react";
import { Button } from "./button";
import { selectors } from "../../consts/selectors";
import { sanitizeErrorMessage, categorizeError, type ErrorCategory } from "../../lib/error-utils";
import { cn } from "../../lib/utils";

interface ErrorDiagnosticsProps {
  /** The caught error object */
  error: Error;
  /** React component stack from errorInfo.componentStack */
  componentStack?: string | null;
  /** Error ID for correlation */
  errorId: string | null;
  /** Error category — if omitted, derived from the error via categorizeError() */
  category?: ErrorCategory;
  /** ISO timestamp of when the error occurred */
  timestamp: string;
  /** Additional CSS class for the outer container */
  className?: string;
  /** Compact mode for inline boundaries (e.g. CanvasErrorBoundary) */
  compact?: boolean;
}

const CATEGORY_COLORS: Record<ErrorCategory, string> = {
  NETWORK: "bg-blue-500/20 text-blue-300 border-blue-500/30",
  TIMEOUT: "bg-yellow-500/20 text-yellow-300 border-yellow-500/30",
  AUTH: "bg-orange-500/20 text-orange-300 border-orange-500/30",
  NOT_FOUND: "bg-slate-500/20 text-slate-300 border-slate-500/30",
  SERVER: "bg-red-500/20 text-red-300 border-red-500/30",
  VALIDATION: "bg-amber-500/20 text-amber-300 border-amber-500/30",
  PARSE: "bg-purple-500/20 text-purple-300 border-purple-500/30",
  STALE_CHUNK: "bg-cyan-500/20 text-cyan-300 border-cyan-500/30",
  RUNTIME: "bg-red-500/20 text-red-300 border-red-500/30",
};

/** Duration to show "Copied!" confirmation (ms) */
const COPY_CONFIRMATION_MS = 2000;

/**
 * Copies text to clipboard with fallback for older browsers (Chrome 67+).
 * Returns true if copy succeeded.
 */
async function copyToClipboard(text: string): Promise<boolean> {
  // Modern Clipboard API (Chrome 66+, requires secure context)
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Falls through to legacy fallback
    }
  }

  // Legacy fallback for non-secure contexts or older browsers
  try {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    textarea.style.top = "-9999px";
    document.body.appendChild(textarea);
    textarea.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(textarea);
    return ok;
  } catch {
    return false;
  }
}

function formatDiagnosticsText(
  error: Error,
  category: ErrorCategory,
  errorId: string | null,
  timestamp: string,
  componentStack: string | null | undefined,
): string {
  const lines = [
    `=== Error Diagnostics ===`,
    `Category: ${category}`,
    `Error: ${error.name}`,
    `Message: ${sanitizeErrorMessage(error.message)}`,
    `Error ID: ${errorId ?? "N/A"}`,
    `Timestamp: ${timestamp}`,
    `User Agent: ${navigator.userAgent}`,
  ];

  if (componentStack) {
    lines.push("", "Component Stack:", componentStack.trim());
  }

  return lines.join("\n");
}

export function ErrorDiagnostics({
  error,
  componentStack,
  errorId,
  category,
  timestamp,
  className,
  compact = false,
}: ErrorDiagnosticsProps) {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);
  const copyTimerRef = useRef<ReturnType<typeof setTimeout>>();

  const resolvedCategory = category ?? categorizeError(error);

  const handleCopy = useCallback(async () => {
    const text = formatDiagnosticsText(error, resolvedCategory, errorId, timestamp, componentStack);
    const ok = await copyToClipboard(text);
    if (ok) {
      setCopied(true);
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
      copyTimerRef.current = setTimeout(() => setCopied(false), COPY_CONFIRMATION_MS);
    }
  }, [error, resolvedCategory, errorId, timestamp, componentStack]);

  const textSize = compact ? "text-xs" : "text-sm";
  const labelSize = compact ? "text-[10px]" : "text-xs";
  const padding = compact ? "p-3" : "p-4";
  const stackMaxHeight = compact ? "max-h-32" : "max-h-64";

  return (
    <div className={cn("mt-4 w-full min-w-0 overflow-hidden", className)}>
      {/* Toggle button */}
      <button
        type="button"
        className={cn(
          "mx-auto flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-slate-400 transition-colors hover:bg-slate-800/60 hover:text-slate-200",
          labelSize,
        )}
        onClick={() => setExpanded((v) => !v)}
        data-testid={selectors.errorBoundary.showDetailsButton}
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5" />
        )}
        {expanded ? "Hide Details" : "Show Details"}
      </button>

      {/* Diagnostics panel */}
      {expanded && (
        <div
          className={cn(
            "mt-2 overflow-hidden rounded-lg border border-slate-700/50 bg-slate-950/80 text-left",
            padding,
          )}
          data-testid={selectors.errorBoundary.diagnosticsPanel}
        >
          {/* Category badge + Copy button row */}
          <div className="flex items-center justify-between gap-2">
            <span
              className={cn(
                "inline-block rounded-md border px-2 py-0.5 font-mono",
                labelSize,
                CATEGORY_COLORS[resolvedCategory],
              )}
              data-testid={selectors.errorBoundary.errorCategory}
            >
              {resolvedCategory}
            </span>
            <Button
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 px-2.5 text-xs"
              onClick={handleCopy}
              data-testid={selectors.errorBoundary.copyButton}
            >
              {copied ? (
                <>
                  <Check className="h-3 w-3" />
                  <span data-testid={selectors.errorBoundary.copyConfirmation}>Copied!</span>
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3" />
                  Copy Details
                </>
              )}
            </Button>
          </div>

          {/* Error name */}
          <div className="mt-3">
            <span className={cn("font-medium uppercase tracking-wider text-slate-500", labelSize)}>
              Error
            </span>
            <p
              className={cn("mt-0.5 break-all font-mono text-slate-300", textSize)}
              data-testid={selectors.errorBoundary.errorName}
            >
              {error.name}
            </p>
          </div>

          {/* Sanitized message */}
          <div className="mt-3">
            <span className={cn("font-medium uppercase tracking-wider text-slate-500", labelSize)}>
              Message
            </span>
            <p
              className={cn("mt-0.5 break-words text-slate-300", textSize)}
              data-testid={selectors.errorBoundary.errorMessage}
            >
              {sanitizeErrorMessage(error.message)}
            </p>
          </div>

          {/* Component stack */}
          <div className="mt-3">
            <span className={cn("font-medium uppercase tracking-wider text-slate-500", labelSize)}>
              Component Stack
            </span>
            {componentStack ? (
              <pre
                className={cn(
                  "mt-0.5 overflow-auto whitespace-pre-wrap rounded-md bg-slate-900/80 p-2 font-mono text-slate-400",
                  textSize,
                  stackMaxHeight,
                )}
                data-testid={selectors.errorBoundary.componentStack}
              >
                {componentStack.trim()}
              </pre>
            ) : (
              <p
                className={cn("mt-0.5 italic text-slate-500", textSize)}
                data-testid={selectors.errorBoundary.componentStack}
              >
                Component stack not available
              </p>
            )}
          </div>

          {/* Timestamp */}
          <div className="mt-3">
            <span className={cn("font-medium uppercase tracking-wider text-slate-500", labelSize)}>
              Timestamp
            </span>
            <p
              className={cn("mt-0.5 font-mono text-slate-400", textSize)}
              data-testid={selectors.errorBoundary.timestamp}
            >
              {timestamp}
            </p>
          </div>

          {/* User agent */}
          <div className="mt-3">
            <span className={cn("font-medium uppercase tracking-wider text-slate-500", labelSize)}>
              User Agent
            </span>
            <p
              className={cn("mt-0.5 break-all text-slate-400", textSize)}
              data-testid={selectors.errorBoundary.userAgent}
            >
              {navigator.userAgent}
            </p>
          </div>

          {/* Error ID */}
          {errorId && (
            <div className="mt-3">
              <span className={cn("font-medium uppercase tracking-wider text-slate-500", labelSize)}>
                Error ID
              </span>
              <p className={cn("mt-0.5 font-mono text-slate-500", textSize)}>
                {errorId}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
