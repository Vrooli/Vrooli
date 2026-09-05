/** @vrooliComponentSource react-component-library:StatusBadge */
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { fetchHealth } from "../../api/health";

/**
 * HealthCard is the canonical *system-status* feature: it polls
 * /health, renders status + service + timestamp, and exposes a
 * refresh button + counter to demonstrate the i18n plural pipeline.
 *
 * New scenarios keep this card unchanged (every API ships a /health
 * surface) and add their own domain features alongside it in
 * `features/<name>/`.
 */
export function HealthCard() {
  const { t } = useTranslation();
  const [refreshCount, setRefreshCount] = useState(0);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
  });

  const handleRefresh = () => {
    setRefreshCount((count) => count + 1);
    void refetch();
  };

  return (
    <div
      data-testid={selectors.health.card}
      className="mt-space-md rounded-xl border border-app-border bg-app-surface p-space-sm"
    >
      <p className="text-sm font-medium text-app-muted-foreground">{t(strings.health.title)}</p>
      {isLoading && (
        <p data-testid={selectors.health.loading} className="mt-space-2xs text-app-foreground">
          {t(strings.health.loading)}
        </p>
      )}
      {error && (
        <p data-testid={selectors.health.error} className="mt-space-2xs text-app-danger">
          {t(strings.health.error)}
        </p>
      )}
      {data && (
        <div className="mt-space-2xs text-sm text-app-foreground">
          <p>
            {t(strings.health.statusLabel)}{" "}
            <span data-testid={selectors.health.statusValue}>{data.status}</span>
          </p>
          <p>
            {t(strings.health.serviceLabel)}{" "}
            <span data-testid={selectors.health.serviceValue}>{data.service}</span>
          </p>
          <p>
            {t(strings.health.timestampLabel)}{" "}
            <span data-testid={selectors.health.timestampValue}>
              {formatDate(new Date(data.timestamp), { dateStyle: "medium", timeStyle: "short" })}
            </span>
          </p>
        </div>
      )}
      <Button
        data-testid={selectors.health.refreshButton}
        className="mt-space-sm"
        onClick={handleRefresh}
      >
        {t(strings.health.refresh)}
        <ArrowRight aria-hidden="true" className="ms-space-2xs h-icon-sm w-icon-sm" />
      </Button>
      {refreshCount > 0 && (
        <p
          data-testid={selectors.health.refreshCount}
          className="mt-space-2xs text-xs text-app-muted-foreground"
        >
          {t(strings.health.refreshCount, { count: refreshCount })}
        </p>
      )}
      <p
        data-testid={selectors.notifications.summary}
        className="mt-space-2xs text-xs text-app-muted-foreground"
      >
        {t(strings.notifications.summary, { count: refreshCount })}
      </p>
    </div>
  );
}
