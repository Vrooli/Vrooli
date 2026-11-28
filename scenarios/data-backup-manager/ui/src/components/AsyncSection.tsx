import type { ReactNode } from "react";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1";

import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { cn } from "../lib/utils";

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
      <ExperienceSurface
        as="div"
        surfaceId="data-backup-manager.async-section"
        state="loading"
        statusMessage={t(strings.common.loading)}
        data-testid={selectors.async.loading}
        className="flex flex-col gap-2"
      >
        {Array.from({ length: skeletonRows }).map((_, i) => (
          <div key={i} className="h-12 animate-pulse rounded-panel bg-app-surface-muted" />
        ))}
      </ExperienceSurface>
    );
  }

  if (isError) {
    return (
      <ExperienceSurface
        as="div"
        surfaceId="data-backup-manager.async-section"
        state="error"
        statusMessage={t(strings.common.loadError)}
        data-testid={selectors.async.error}
      >
        <div
          role="alert"
          className="flex flex-col items-start gap-3 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <p className="text-sm text-app-danger">{t(strings.common.loadError)}</p>
          {onRetry && (
            <Button
              variant="ghost"
              size="sm"
              data-testid={selectors.async.retry}
              onClick={onRetry}
              shape="square"
              className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
            >
              {t(strings.common.retry)}
            </Button>
          )}
        </div>
      </ExperienceSurface>
    );
  }

  if (isEmpty) {
    return (
      <ExperienceSurface
        as="div"
        surfaceId="data-backup-manager.async-section"
        state="empty"
        data-testid={selectors.async.empty}
      >
        {emptyState}
      </ExperienceSurface>
    );
  }

  return <>{children}</>;
}
