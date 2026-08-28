import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";

import { fetchHealth } from "../api/health";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { formatDate } from "../i18n/format";
import { useTranslation } from "../i18n";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

/**
 * HealthCard is the canonical system-status component: it polls /health,
 * renders status + service + timestamp, and exposes a refresh control.
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
    <Card data-testid={selectors.health.card} className="h-full">
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle>{t(strings.health.title)}</CardTitle>
            <CardDescription>{t(strings.pages.dashboard.description)}</CardDescription>
          </div>
          {data && (
            <StatusBadge tone={data.status === "ok" ? "success" : "warning"}>
              <span data-testid={selectors.health.statusValue}>{data.status}</span>
            </StatusBadge>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading && (
          <p data-testid={selectors.health.loading} className="text-sm text-app-muted-foreground">
            {t(strings.health.loading)}
          </p>
        )}
        {error && (
          <p data-testid={selectors.health.error} className="text-sm text-app-danger">
            {t(strings.health.error)}
          </p>
        )}
        {data && (
          <dl className="grid gap-2 text-sm md:grid-cols-2">
            <div>
              <dt className="text-app-muted-foreground">{t(strings.health.serviceLabel)}</dt>
              <dd data-testid={selectors.health.serviceValue} className="font-medium">
                {data.service}
              </dd>
            </div>
            <div>
              <dt className="text-app-muted-foreground">{t(strings.health.timestampLabel)}</dt>
              <dd data-testid={selectors.health.timestampValue} className="font-medium">
                {formatDate(new Date(data.timestamp), { dateStyle: "medium", timeStyle: "short" })}
              </dd>
            </div>
          </dl>
        )}
        <div className="flex flex-wrap items-center gap-3">
          <Button
            data-testid={selectors.health.refreshButton}
            variant="secondary"
            onClick={handleRefresh}
          >
            <RefreshCw aria-hidden="true" className="h-4 w-4" />
            {t(strings.health.refresh)}
          </Button>
          {refreshCount > 0 && (
            <p
              data-testid={selectors.health.refreshCount}
              className="text-xs text-app-muted-foreground"
            >
              {t(strings.health.refreshCount, { count: refreshCount })}
            </p>
          )}
        </div>
        <p
          data-testid={selectors.notifications.summary}
          className="text-xs text-app-muted-foreground"
        >
          {t(strings.notifications.summary, { count: refreshCount })}
        </p>
      </CardContent>
    </Card>
  );
}
