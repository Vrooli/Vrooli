import { useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { fetchHealth } from "../../api/health";

/**
 * HealthCard reports the API's /health surface (status + service + timestamp)
 * with a manual refresh. Folded into Settings since every API ships /health.
 */
export function HealthCard() {
  const { t } = useTranslation();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
  });

  return (
    <div
      data-testid={selectors.health.card}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <p className="text-sm font-medium text-app-muted-foreground">{t(strings.health.title)}</p>
      {isLoading && (
        <p data-testid={selectors.health.loading} className="mt-2 text-sm text-app-foreground">
          {t(strings.health.loading)}
        </p>
      )}
      {error && (
        <p data-testid={selectors.health.error} className="mt-2 text-sm text-app-danger">
          {t(strings.health.error)}
        </p>
      )}
      {data && (
        <div className="mt-2 text-sm text-app-foreground">
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
        variant="outline"
        size="sm"
        className="mt-4"
        onClick={() => void refetch()}
      >
        <RefreshCw aria-hidden="true" className="me-2 h-4 w-4" />
        {t(strings.health.refresh)}
      </Button>
    </div>
  );
}
