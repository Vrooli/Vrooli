import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { fetchHealth } from "../../lib/api";

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
      className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <p className="text-sm font-medium text-slate-400">{t(strings.health.title)}</p>
      {isLoading && (
        <p data-testid={selectors.health.loading} className="mt-2 text-slate-200">
          {t(strings.health.loading)}
        </p>
      )}
      {error && (
        <p data-testid={selectors.health.error} className="mt-2 text-red-400">
          {t(strings.health.error)}
        </p>
      )}
      {data && (
        <div className="mt-2 text-sm text-slate-200">
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
        className="mt-4"
        onClick={handleRefresh}
      >
        {t(strings.health.refresh)}
        <ArrowRight aria-hidden="true" className="ms-2 h-4 w-4" />
      </Button>
      {refreshCount > 0 && (
        <p
          data-testid={selectors.health.refreshCount}
          className="mt-2 text-xs text-slate-500"
        >
          {t(strings.health.refreshCount, { count: refreshCount })}
        </p>
      )}
      <p
        data-testid={selectors.notifications.summary}
        className="mt-2 text-xs text-slate-500"
      >
        {t(strings.notifications.summary, { count: refreshCount })}
      </p>
    </div>
  );
}
