import { useQuery } from "@tanstack/react-query";
import { getProviderHealth } from "../../api/healthStatus";
import { CapabilityRow } from "./CapabilityRow";

const MIN_REFETCH_MS = 5_000;

export function StatusPage() {
  const { data, error, isLoading } = useQuery({
    queryKey: ["healthStatus", "providers"],
    queryFn: getProviderHealth,
    refetchInterval: (q) => {
      const ttl = q.state.data?.cacheTtlSeconds ?? 0;
      const ms = Math.max(MIN_REFETCH_MS, Math.floor((ttl / 2) * 1000));
      return ms;
    },
    retry: false,
  });

  if (isLoading) {
    return <p className="text-sm text-app-muted-foreground">Loading provider health…</p>;
  }
  if (error || !data) {
    return (
      <p className="text-sm text-app-danger">
        Could not load provider health. The API may be offline.
      </p>
    );
  }

  return (
    <section className="flex flex-col gap-4" aria-label="Provider health">
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold text-app-foreground">Status</h1>
        <p className="text-sm text-app-muted-foreground">
          Per-capability provider health. Generated at {data.generatedAt}; refresh
          cadence {data.cacheTtlSeconds}s.
        </p>
      </header>
      {data.capabilities.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">
          No capabilities reported. The capability registry may be empty in this
          deployment.
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {data.capabilities.map((c) => (
            <CapabilityRow key={c.capability} capability={c} />
          ))}
        </div>
      )}
    </section>
  );
}

export default StatusPage;
