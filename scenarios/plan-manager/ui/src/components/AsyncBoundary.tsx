import { type ReactNode } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

/**
 * AsyncBoundary is the canonical loading / error / empty wrapper for every
 * query-backed surface in the console. Pass the TanStack Query flags plus the
 * data, and it renders the right state — children only render once data is
 * present and non-empty. Centralizing this keeps every board's three non-happy
 * states visually and accessibly consistent (status role for loading,
 * alert role for errors).
 */
export interface AsyncBoundaryProps {
  isLoading: boolean;
  error: unknown;
  /** When true, render the empty state instead of children. */
  isEmpty?: boolean;
  /** Test id applied to whichever state element renders. */
  testIdPrefix: string;
  /** Optional override copy for the empty state. */
  emptyLabel?: string;
  children: ReactNode;
}

export function AsyncBoundary({
  isLoading,
  error,
  isEmpty,
  testIdPrefix,
  emptyLabel,
  children,
}: AsyncBoundaryProps) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div
        role="status"
        aria-live="polite"
        data-testid={`${testIdPrefix}-${selectors.asyncSuffix.loading}`}
        className="flex items-center gap-2 rounded-panel border border-app-border bg-app-surface px-4 py-6 text-sm text-app-muted-foreground"
      >
        <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
        {t(strings.common.loading)}
      </div>
    );
  }

  if (error) {
    return (
      <div
        role="alert"
        data-testid={`${testIdPrefix}-${selectors.asyncSuffix.error}`}
        className="flex items-start gap-2 rounded-panel border border-app-danger/40 bg-app-danger/10 px-4 py-4 text-sm text-app-danger"
      >
        <AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
        <span>{errorMessage(error, t)}</span>
      </div>
    );
  }

  if (isEmpty) {
    return (
      <div
        data-testid={`${testIdPrefix}-${selectors.asyncSuffix.empty}`}
        className="rounded-panel border border-dashed border-app-border bg-app-surface px-4 py-10 text-center text-sm text-app-muted-foreground"
      >
        {emptyLabel ?? t(strings.common.empty)}
      </div>
    );
  }

  return <>{children}</>;
}
