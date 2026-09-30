/**
 * @libraryId react-component-library:HealthCard
 * @displayName Health Card
 * @description A presentational system-health card with accessible loading, error, status, metadata, and refresh states.
 * @version 0.1.6
 * @tags ["data-display","feedback","async"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@vrooli/react-component-library/Card/1";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";

import type { HTMLAttributes, ReactNode } from "react";

const healthCardStyles = `
[data-rcl-health-card] [data-rcl-health-card-header] { display: flex; min-inline-size: 0; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); }
[data-rcl-health-card] [data-rcl-health-card-heading] { min-inline-size: 0; }
[data-rcl-health-card] [data-rcl-health-card-content] { display: grid; gap: var(--space-sm); }
[data-rcl-health-card] [data-rcl-health-card-state] { margin: 0; color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-health-card] [data-rcl-health-card-state][data-state="error"] { color: var(--color-danger); }
[data-rcl-health-card] [data-rcl-health-card-details] { display: grid; gap: var(--space-sm); margin: 0; }
[data-rcl-health-card] [data-rcl-health-card-details] > div { min-inline-size: 0; }
[data-rcl-health-card] [data-rcl-health-card-details] dt { color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-health-card] [data-rcl-health-card-details] dd { margin: 0; color: var(--color-foreground); font-weight: 650; }
@media (min-width: 40rem) { [data-rcl-health-card] [data-rcl-health-card-details] { grid-template-columns: repeat(2, minmax(0, 1fr)); } [data-rcl-health-card] [data-rcl-health-card-details] > div:last-child { grid-column: 1 / -1; } }
`;

export interface HealthCardData {
  status: string;
  service: string;
  timestamp: string | Date;
}

export interface HealthCardTestIds {
  card?: string;
  loading?: string;
  error?: string;
  statusValue?: string;
  serviceValue?: string;
  timestampValue?: string;
  refreshButton?: string;
  refreshCount?: string;
}

export interface HealthCardProps extends Omit<HTMLAttributes<HTMLDivElement>, "title"> {
  data?: HealthCardData;
  loading?: boolean;
  error?: ReactNode;
  onRefresh?: () => void | Promise<void>;
  refreshing?: boolean;
  refreshCount?: number;
  refreshCountLabel?: ReactNode;
  title?: ReactNode;
  description?: ReactNode;
  statusLabel?: ReactNode;
  serviceLabel?: ReactNode;
  timestampLabel?: ReactNode;
  loadingLabel?: ReactNode;
  errorLabel?: ReactNode;
  refreshLabel?: ReactNode;
  emptyLabel?: ReactNode;
  formatTimestamp?: (timestamp: string | Date) => ReactNode;
  statusTone?: StatusTone;
  testIds?: HealthCardTestIds;
  footer?: ReactNode;
}

const defaultTestIds: Required<HealthCardTestIds> = {
  card: "data-display.health-card",
  loading: "data-display.health-card.loading",
  error: "data-display.health-card.error",
  statusValue: "data-display.health-card.status",
  serviceValue: "data-display.health-card.service",
  timestampValue: "data-display.health-card.timestamp",
  refreshButton: "data-display.health-card.refresh",
  refreshCount: "data-display.health-card.refresh-count",
};

function toneForStatus(status: string): StatusTone {
  switch (status.trim().toLowerCase()) {
    case "ok":
    case "healthy":
    case "ready":
      return "success";
    case "degraded":
    case "warning":
      return "warning";
    case "error":
    case "failed":
    case "unavailable":
    case "unhealthy":
      return "danger";
    default:
      return "neutral";
  }
}

function defaultTimestamp(timestamp: string | Date): ReactNode {
  const date = timestamp instanceof Date ? timestamp : new Date(timestamp);
  if (Number.isNaN(date.getTime())) return String(timestamp);
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function dateTimeValue(timestamp: string | Date): string | undefined {
  const date = timestamp instanceof Date ? timestamp : new Date(timestamp);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

export const HealthCard = withClassName(function HealthCard({
  data,
  loading = false,
  error,
  onRefresh,
  refreshing = false,
  refreshCount,
  refreshCountLabel,
  title,
  description,
  statusLabel,
  serviceLabel,
  timestampLabel,
  loadingLabel,
  errorLabel,
  refreshLabel,
  emptyLabel,
  formatTimestamp = defaultTimestamp,
  statusTone,
  testIds,
  footer,
  className,
  ...props
}: HealthCardProps) {
  useLibraryStyleSheet("react-component-library:HealthCard", "0.1.6", healthCardStyles);
  const strings = useStrings();
  const ids = { ...defaultTestIds, ...testIds };
  const resolvedTitle = title ?? strings("data-display.health-card.title", "System health");
  const resolvedDescription =
    description ?? strings("data-display.health-card.description", "Runtime status and reachability");
  const resolvedStatusLabel = statusLabel ?? strings("data-display.health-card.status", "Status");
  const resolvedServiceLabel = serviceLabel ?? strings("data-display.health-card.service", "Service");
  const resolvedTimestampLabel =
    timestampLabel ?? strings("data-display.health-card.last-checked", "Last checked");
  const resolvedLoadingLabel =
    loadingLabel ?? strings("data-display.health-card.loading", "Checking health…");
  const resolvedErrorLabel =
    errorLabel ?? strings("data-display.health-card.error", "Health information is unavailable.");
  const resolvedEmptyLabel =
    emptyLabel ?? strings("data-display.health-card.empty", "No health information yet.");
  const resolvedRefreshLabel = refreshLabel ?? strings("data-display.health-card.refresh", "Refresh");
  const hasError = error !== undefined && error !== null;
  const stateMessage = loading
    ? resolvedLoadingLabel
    : hasError
      ? error || resolvedErrorLabel
      : !data
        ? resolvedEmptyLabel
        : null;

  return (
    <Card
      {...props}
      className={className}
      data-testid={ids.card}
      data-rcl-health-card
      aria-busy={loading || refreshing || undefined}
    >
      <CardHeader>
        <div data-rcl-health-card-header>
          <div data-rcl-health-card-heading>
            <CardTitle as="h2">{resolvedTitle}</CardTitle>
            <CardDescription>{resolvedDescription}</CardDescription>
          </div>
          {data ? (
            <StatusBadge
              tone={statusTone ?? toneForStatus(data.status)}
            >
              {data.status}
            </StatusBadge>
          ) : null}
        </div>
      </CardHeader>
      <CardContent data-rcl-health-card-content>
        {stateMessage ? (
          <p
            data-testid={loading ? ids.loading : ids.error}
            data-rcl-health-card-state
            data-state={hasError ? "error" : "loading"}
            role={hasError ? "alert" : "status"}
            aria-live="polite"
          >
            {stateMessage}
          </p>
        ) : null}
        {data ? (
          <dl data-rcl-health-card-details>
            <div>
              <dt>{resolvedStatusLabel}</dt>
              <dd data-testid={ids.statusValue}>
                {data.status}
              </dd>
            </div>
            <div>
              <dt>{resolvedServiceLabel}</dt>
              <dd data-testid={ids.serviceValue}>
                {data.service}
              </dd>
            </div>
            <div>
              <dt>{resolvedTimestampLabel}</dt>
              <dd>
                <time dateTime={dateTimeValue(data.timestamp)} data-testid={ids.timestampValue}>
                  {formatTimestamp(data.timestamp)}
                </time>
              </dd>
            </div>
          </dl>
        ) : null}
        {onRefresh ? (
          <Button
            type="button"
            variant="secondary"
            onClick={() => void onRefresh()}
            pending={refreshing}
            disabled={refreshing}
            data-testid={ids.refreshButton}
          >
            {resolvedRefreshLabel}
          </Button>
        ) : null}
        {refreshCount !== undefined && refreshCount > 0 && refreshCountLabel ? (
          <p data-testid={ids.refreshCount} data-rcl-health-card-state>
            {refreshCountLabel}
          </p>
        ) : null}
        {footer}
      </CardContent>
    </Card>
  );
});

export default HealthCard;
