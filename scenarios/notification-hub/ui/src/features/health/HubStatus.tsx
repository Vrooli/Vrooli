import { useQuery } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { fetchHealth } from "../../api/health";
import { useAgo } from "../../hooks/useAgo";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** Whether the Hub itself is up: one line, one fact, one way to check again. */
export function HubStatus() {
  const { t } = useTranslation();
  const { data, error, isFetching, isLoading, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 60_000,
  });
  const healthy = data?.status === "healthy";
  const checkedAgo = useAgo(data?.timestamp);
  return (
    <section
      aria-label={t(strings.health.title)}
      data-testid={selectors.health.card}
      className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-panel border border-app-border bg-app-surface px-4 py-3"
    >
      <div className="flex min-w-0 flex-col gap-1">
        <h2 className="text-sm font-semibold">{t(strings.health.title)}</h2>
        {isLoading && <p data-testid={selectors.health.loading} role="status" className="text-sm text-app-muted-foreground">{t(strings.health.loading)}</p>}
        {error && <p data-testid={selectors.health.error} role="alert" className="text-sm text-app-danger">{t(strings.health.error)}</p>}
        {data && (
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-app-muted-foreground">
            <StatusBadge tone={healthy ? "success" : "warning"} data-testid={selectors.health.statusValue}>
              {t(healthy ? strings.health.running : strings.health.notRunning)}
            </StatusBadge>
            <span data-testid={selectors.health.serviceValue} className="sr-only">{data.service}</span>
            {checkedAgo && <span data-testid={selectors.health.timestampValue}>{t(strings.health.checked, { time: checkedAgo })}</span>}
          </p>
        )}
      </div>
      <Button type="button" variant="outline" data-testid={selectors.health.refreshButton} disabled={isFetching} onClick={() => void refetch()}>
        {t(strings.health.refresh)}
      </Button>
    </section>
  );
}
