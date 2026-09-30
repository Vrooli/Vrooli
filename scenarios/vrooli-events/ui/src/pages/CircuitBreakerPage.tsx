// REQ-UI-008: Circuit breaker dashboard
// DOC: docs/reference/api-endpoints.md#circuit-breaker-override
// DOC: docs/guides/managing-policies.md
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Zap, RefreshCw } from "lucide-react";
import { Button } from "../components/ui/button";
import { ErrorAlert } from "../components/ErrorAlert";
import { Spinner } from "../components/Spinner";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { StatusBadge } from "../components/StatusBadge";
import { fetchPolicies, overrideCircuitBreaker, type PolicyRule } from "../lib/api";
import { HEALTH_POLL_INTERVAL_MS, INPUT_CLASS } from "../lib/constants";

export function CircuitBreakerPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [overrideTarget, setOverrideTarget] = useState<number | null>(null);
  const [overrideState, setOverrideState] = useState("closed");
  const [overrideTtl, setOverrideTtl] = useState(60);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["policies"],
    queryFn: fetchPolicies,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
    select: (rules: PolicyRule[]) => rules.filter((r) => r.rule_type === "circuit_breaker"),
  });

  const overrideMutation = useMutation({
    mutationFn: ({ id, state, ttl }: { id: number; state: string; ttl: number }) =>
      overrideCircuitBreaker(id, state, ttl),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
      setOverrideTarget(null);
    },
  });

  const handleOverride = () => {
    if (overrideTarget === null) return;
    overrideMutation.mutate({ id: overrideTarget, state: overrideState, ttl: overrideTtl });
  };

  return (
    <div className="space-y-4" data-testid="circuit-breakers-page">
      <PageHeader
        icon={Zap}
        title="Circuit Breakers"
        actions={
          <Button size="sm" variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Refresh
          </Button>
        }
      />

      {isLoading && <Spinner label="Loading circuit breakers..." />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}
      {overrideMutation.error && <ErrorAlert error={overrideMutation.error} compact />}

      {overrideTarget !== null && (
        <Panel data-testid="cb-override-form">
          <h3 className="mb-3 text-sm font-medium text-[var(--text-secondary)]">
            Override Breaker #{overrideTarget}
          </h3>
          <div className="flex flex-wrap items-end gap-3">
            <div>
              <label className="mb-1 block text-xs text-[var(--text-faint)]">State</label>
              <select
                value={overrideState}
                onChange={(e) => setOverrideState(e.target.value)}
                className={INPUT_CLASS}
              >
                <option value="open">Open</option>
                <option value="closed">Closed</option>
                <option value="half_open">Half Open</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs text-[var(--text-faint)]">TTL (seconds)</label>
              <input
                type="number"
                value={overrideTtl}
                onChange={(e) => setOverrideTtl(Number(e.target.value))}
                className={`w-28 ${INPUT_CLASS}`}
              />
            </div>
            <Button size="sm" onClick={handleOverride} disabled={overrideMutation.isPending}>
              {overrideMutation.isPending ? "Applying..." : "Apply Override"}
            </Button>
            <Button size="sm" variant="outline" onClick={() => setOverrideTarget(null)}>
              Cancel
            </Button>
          </div>
        </Panel>
      )}

      {data && data.length > 0 && (
        <Panel>
          <div className="overflow-x-auto">
            <table className="w-full text-sm" data-testid="circuit-breakers-table">
              <thead>
                <tr className="border-b border-[var(--border-subtle)] text-left text-xs text-[var(--text-faint)]">
                  <th className="px-3 py-2">ID</th>
                  <th className="px-3 py-2">Source</th>
                  <th className="px-3 py-2">Target</th>
                  <th className="px-3 py-2">Endpoint</th>
                  <th className="px-3 py-2">Failure Threshold</th>
                  <th className="px-3 py-2">Cooldown (s)</th>
                  <th className="px-3 py-2">Success Threshold</th>
                  <th className="px-3 py-2">Enabled</th>
                  <th className="px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {data.map((rule) => (
                  <tr
                    key={rule.id}
                    className="border-b border-[var(--border-subtle)] hover:bg-white/5"
                  >
                    <td className="px-3 py-2 font-mono text-xs text-[var(--text-accent)]">{rule.id}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.source_scenario}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.target_scenario}</td>
                    <td className="px-3 py-2 font-mono text-xs text-[var(--text-secondary)]">
                      {rule.endpoint_pattern ?? "-"}
                    </td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.failure_threshold ?? 0}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.cooldown_seconds ?? 0}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.success_threshold ?? 0}</td>
                    <td className="px-3 py-2">
                      <StatusBadge active={rule.enabled} />
                    </td>
                    <td className="px-3 py-2">
                      <button
                        onClick={() => {
                          setOverrideTarget(rule.id);
                          setOverrideState("closed");
                          setOverrideTtl(60);
                        }}
                        className="text-xs text-[var(--text-accent)] hover:underline"
                      >
                        Override
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}

      {data && data.length === 0 && (
        <Panel>
          <div className="py-8 text-center text-sm text-[var(--text-muted)]">
            <Zap className="mx-auto mb-3 h-8 w-8 text-[var(--text-faint)]" />
            <p>No circuit breaker policies defined yet.</p>
            <p className="mt-1 text-xs text-[var(--text-faint)]">
              Create a policy rule with type "circuit_breaker" on the{" "}
              <button
                onClick={() => navigate("/policies")}
                className="text-[var(--text-accent)] hover:underline"
              >
                Policies page
              </button>.
            </p>
          </div>
        </Panel>
      )}
    </div>
  );
}
