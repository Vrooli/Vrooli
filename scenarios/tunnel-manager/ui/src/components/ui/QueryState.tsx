import type { ReactNode } from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { Button } from "./button";

interface QueryStateProps {
  isLoading: boolean;
  error: unknown;
  /** True when the query resolved but produced no rows. */
  isEmpty?: boolean;
  /** Override copy for each state; defaults to the shared `common.*` strings. */
  loadingLabel?: string;
  errorLabel?: string;
  emptyLabel?: string;
  /** Refetch the failed query when the operator wants to try again. */
  onRetry?: () => void;
  retryLabel?: string;
  /** Optional capability-specific remediation action shown beside retry. */
  errorAction?: ReactNode;
  children: ReactNode;
}

/**
 * QueryState renders the canonical loading / error / empty fallbacks for a
 * react-query result, or its children when data is present. Every surface
 * routes its query results through this so the three states look and read the
 * same across the app and so tests have stable selectors to assert on.
 *
 * Pass `error` straight from `useQuery`; it is normalised through
 * `errorMessage` (Connect codes → translated copy) when no `errorLabel`
 * override is given.
 */
export function QueryState({
  isLoading,
  error,
  isEmpty = false,
  loadingLabel,
  errorLabel,
  emptyLabel,
  onRetry,
  retryLabel,
  errorAction,
  children,
}: QueryStateProps) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div
        data-testid={selectors.queryState.loading}
        role="status"
        aria-busy="true"
        aria-label={loadingLabel ?? t(strings.common.loading)}
        className="flex flex-col gap-3 rounded-control border border-app-border bg-app-surface-muted/60 p-4"
      >
        <span className="h-3 w-32 animate-pulse rounded-full bg-app-border" />
        <span className="h-3 w-full animate-pulse rounded-full bg-app-border/80" />
        <span className="h-3 w-2/3 animate-pulse rounded-full bg-app-border/60" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        data-testid={selectors.queryState.error}
        role="alert"
        className="flex flex-col items-stretch gap-3 rounded-control border border-app-danger/30 bg-app-danger/5 p-4 text-sm sm:flex-row sm:items-center sm:justify-between"
      >
        <p className="w-full min-w-0 break-words text-app-danger sm:flex-1">{errorLabel ?? errorMessage(error, t)}</p>
        <div className="flex shrink-0 flex-wrap gap-2">
          {errorAction}
          {onRetry ? (
            <Button type="button" variant="outline" onClick={onRetry}>
              {retryLabel ?? t(strings.common.refresh)}
            </Button>
          ) : null}
        </div>
      </div>
    );
  }

  if (isEmpty) {
    return (
      <p data-testid={selectors.queryState.empty} className="text-sm text-app-muted-foreground">
        {emptyLabel ?? t(strings.common.empty)}
      </p>
    );
  }

  return <>{children}</>;
}
