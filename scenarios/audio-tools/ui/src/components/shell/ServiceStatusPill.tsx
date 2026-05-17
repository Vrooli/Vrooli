import { useQuery } from "@tanstack/react-query";
import { State } from "@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb";
import { fetchHealth } from "../../api/health";
import { getProviderHealth } from "../../api/healthStatus";
import { StatusDot } from "../ui/status-dot";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

const MIN_REFETCH_MS = 5_000;

/**
 * Persistent header pill. Aggregates two signals:
 *   - REST `/health` (existing) — detects "API offline" because Connect
 *     fails opaquely when the whole server is down.
 *   - Connect `HealthStatusService.GetProviderHealth` — reports
 *     per-provider degradation.
 *
 * Label resolution (highest priority first):
 *   "API offline"   — REST /health fails.
 *   "Degraded — N down" — REST /health ok, but ≥1 provider UNAVAILABLE.
 *   "Healthy"        — REST /health ok and every provider AVAILABLE/UNKNOWN.
 */
export function ServiceStatusPill() {
  const { t } = useTranslation();

  const restHealth = useQuery({
    queryKey: ["health", "pill"],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
    retry: false,
  });

  const providerHealth = useQuery({
    queryKey: ["healthStatus", "pill"],
    queryFn: getProviderHealth,
    refetchInterval: (q) => {
      const ttl = q.state.data?.cacheTtlSeconds ?? 0;
      return Math.max(MIN_REFETCH_MS, Math.floor((ttl / 2) * 1000));
    },
    retry: false,
    // REST `/health` going red already tells us the API is down — we
    // don't need provider polling to keep retrying against a corpse.
    enabled: restHealth.isSuccess,
  });

  if (restHealth.isLoading) {
    return <StatusDot tone="neutral" pulse label={t(strings.status.checking)} />;
  }
  if (restHealth.isError || !restHealth.data) {
    return <StatusDot tone="danger" label={t(strings.status.offline)} />;
  }

  // REST is fine; consult provider rollup.
  const caps = providerHealth.data?.capabilities ?? [];
  let downCount = 0;
  for (const c of caps) {
    for (const p of c.providers) {
      if (p.state === State.UNAVAILABLE) downCount += 1;
    }
  }

  if (downCount > 0) {
    return (
      <StatusDot tone="warning" label={`Degraded — ${downCount} down`} />
    );
  }
  return <StatusDot tone="success" label={restHealth.data.service || t(strings.overview.summaryApi)} />;
}
