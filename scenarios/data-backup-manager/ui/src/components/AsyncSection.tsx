import type { ReactNode } from "react";

import { Button } from "./ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Wraps an async surface so every list/panel honors the full request lifecycle
 * (DESIGN.md): a skeleton while loading, a readable error with a retry action,
 * an empty state when there's no data, and the content otherwise. Centralizing
 * this keeps loading/error/empty consistent across all six surfaces.
 */
export function AsyncSection({
  isLoading,
  isError,
  isEmpty,
  onRetry,
  emptyState,
  skeletonRows = 3,
  children,
}: {
  isLoading: boolean;
  isError: boolean;
  isEmpty?: boolean;
  onRetry?: () => void;
  emptyState?: ReactNode;
  skeletonRows?: number;
  children: ReactNode;
}) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div data-testid={selectors.async.loading} className="flex flex-col gap-2" aria-busy="true">
        {Array.from({ length: skeletonRows }).map((_, i) => (
          <div key={i} className="h-12 animate-pulse rounded-panel bg-app-surface-muted" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div
        data-testid={selectors.async.error}
        role="alert"
        className="flex flex-col items-start gap-3 rounded-panel border border-app-border bg-app-surface p-4"
      >
        <p className="text-sm text-app-danger">{t(strings.common.loadError)}</p>
        {onRetry && (
          <Button variant="outline" size="sm" data-testid={selectors.async.retry} onClick={onRetry}>
            {t(strings.common.retry)}
          </Button>
        )}
      </div>
    );
  }

  if (isEmpty) {
    return <div data-testid={selectors.async.empty}>{emptyState}</div>;
  }

  return <>{children}</>;
}
