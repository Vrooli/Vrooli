import { AlertTriangle, RefreshCw } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ExperienceSurface, type ExperienceSurfaceState } from "../experience/ExperienceSurface";

interface RegionProps {
  /** Matches the region id declared in `experience/pages/*.json`. */
  surfaceId: string;
  state: ExperienceSurfaceState;
  children?: ReactNode;
  /** Rendered while `state === "loading"`; defaults to a skeleton block. */
  loading?: ReactNode;
  empty?: ReactNode;
  error?: ReactNode;
  errorDetail?: string;
  onRetry?: () => void;
  /** Adds a header row above the surface. */
  title?: ReactNode;
  actions?: ReactNode;
  testId?: string;
  className?: string;
  /** Number of skeleton rows for the default loading treatment. */
  skeletonRows?: number;
}

/**
 * One declared async region. Every surface the experience contract names
 * renders through this so the loading, empty, and error states are consistent
 * and machine-checkable via `data-experience-surface` / `data-experience-state`.
 */
export function Region({
  surfaceId,
  state,
  children,
  loading,
  empty,
  error,
  errorDetail,
  onRetry,
  title,
  actions,
  testId,
  className,
  skeletonRows = 3,
}: RegionProps) {
  const { t } = useTranslation();
  let body: ReactNode = children;
  if (state === "loading") {
    body = loading ?? <Skeleton rows={skeletonRows} />;
  } else if (state === "empty") {
    body = empty ?? children;
  } else if (state === "error") {
    body = error ?? (
      <div role="alert" className="flex flex-col items-start gap-3 rounded-panel border border-app-danger/40 bg-app-danger/5 p-4 text-sm">
        <div className="flex items-start gap-2">
          <AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-app-danger" />
          <div>
            <p className="font-medium text-app-foreground">{t(strings.console.region.errorTitle)}</p>
            {errorDetail ? <p className="mt-1 break-words font-mono text-xs text-app-muted-foreground">{errorDetail}</p> : null}
          </div>
        </div>
        {onRetry ? (
          <Button type="button" variant="secondary" size="sm" onClick={onRetry} data-testid={testId ? `${testId}-retry` : undefined}>
            <RefreshCw aria-hidden="true" className="h-3.5 w-3.5" />
            {t(strings.console.region.retry)}
          </Button>
        ) : null}
      </div>
    );
  }
  return (
    <ExperienceSurface
      surfaceId={surfaceId}
      state={state}
      data-testid={testId}
      className={["flex min-w-0 flex-col gap-3", className ?? ""].join(" ")}
      statusMessage={state === "loading" ? t(strings.console.region.loading) : state === "error" ? t(strings.console.region.errorTitle) : undefined}
    >
      {title || actions ? (
        <header className="flex flex-wrap items-center justify-between gap-2">
          {title ? <h3 className="text-sm font-semibold text-app-foreground">{title}</h3> : <span />}
          {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
        </header>
      ) : null}
      {body}
    </ExperienceSurface>
  );
}

export function Skeleton({ rows = 3, className }: { rows?: number; className?: string }) {
  return (
    <div aria-hidden="true" className={["flex flex-col gap-2", className ?? ""].join(" ")}>
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          className="h-10 animate-pulse rounded-control bg-app-surface-muted"
          style={{ width: `${100 - (index % 3) * 12}%` }}
        />
      ))}
    </div>
  );
}

/** Quiet empty treatment for a region: an icon, a line, an optional action. */
export function Quiet({ icon, title, description, action, testId }: { icon?: ReactNode; title: string; description?: string; action?: ReactNode; testId?: string }) {
  return (
    <div
      data-testid={testId}
      role="status"
      className="flex flex-col items-center justify-center gap-2 rounded-panel border border-dashed border-app-border px-4 py-8 text-center"
    >
      {icon ? <span className="text-app-muted-foreground">{icon}</span> : null}
      <p className="text-sm font-medium text-app-foreground">{title}</p>
      {description ? <p className="max-w-prose text-sm text-app-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}
