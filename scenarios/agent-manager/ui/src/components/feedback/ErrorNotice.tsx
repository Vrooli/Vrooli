import { AlertCircle, RefreshCw } from "lucide-react";
import { ErrorState } from "@vrooli/react-component-library/ErrorState/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { presentError } from "../../lib/errorPresentation";

interface ErrorNoticeProps {
  error: unknown;
  /** What failed to load, e.g. "runs"; prefixes the compact line. */
  subject?: string;
  onRetry?: () => void | Promise<void>;
  /** One line for dense contexts (cards, rows, a list that still shows data). */
  compact?: boolean;
  testId?: string;
}

/**
 * The single error presentation: cause, what to do, retry, and the verbatim
 * transport detail collapsed for engineers. Full mode is the RCL ErrorState.
 */
export function ErrorNotice({ error, subject, onRetry, compact = false, testId = "error-notice" }: ErrorNoticeProps) {
  const view = presentError(error);
  const retry = view.retryable ? onRetry : undefined;
  if (!compact) {
    return (
      <div data-testid={testId} data-error-kind={view.kind}>
        <ErrorState
          title={subject ? `${view.title} — ${subject} unavailable` : view.title}
          message={view.message}
          detail={view.detail || undefined}
          onRetry={retry}
        />
      </div>
    );
  }
  return (
    <div
      role="alert"
      data-testid={testId}
      data-error-kind={view.kind}
      className="flex min-w-0 items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm"
    >
      <AlertCircle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
      <p className="min-w-0 flex-1" title={view.detail || undefined}>
        <span className="font-medium text-foreground">{subject ? `${subject}: ${view.title}` : view.title}.</span>{" "}
        <span className="text-muted-foreground">{view.message}</span>
      </p>
      {retry ? (
        <Button type="button" variant="ghost" size="sm" density="compact" onClick={() => void retry()} icon={<RefreshCw aria-hidden="true" className="h-3.5 w-3.5" />}>
          Retry
        </Button>
      ) : null}
    </div>
  );
}
