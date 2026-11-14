/**
 * HealthPill — compact API status indicator anchored in the top bar.
 *
 * Polls `/health` on a 30s interval. Replaces the bulky template
 * `HealthCard` for normal operation; the dashboard's "API status" card
 * can expand into a full detail view later.
 */
import { useQuery } from "@tanstack/react-query";
import { Activity } from "lucide-react";

import { fetchHealth } from "../api/health";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export function HealthPill() {
  const { t } = useTranslation();
  const { data, error, isLoading } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
  });

  const tone = error
    ? "bg-app-danger/15 text-app-danger"
    : isLoading
      ? "bg-app-surface-muted text-app-muted-foreground"
      : "bg-app-success/15 text-app-success";

  const label = error
    ? t(strings.health.pill.error)
    : isLoading
      ? t(strings.health.pill.loading)
      : data?.status
        ? data.status
        : t(strings.health.pill.ok);

  return (
    <span
      data-testid={selectors.health.pill}
      role="status"
      aria-live="polite"
      className={`inline-flex items-center gap-1.5 rounded-pill px-2.5 py-1 text-xs font-medium ${tone}`}
    >
      <Activity aria-hidden className="h-3.5 w-3.5" />
      {label}
    </span>
  );
}
