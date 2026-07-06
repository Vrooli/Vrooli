/**
 * GoalDrainToggle — the continuous goal-directed auto-enqueue switch (plan D4,
 * default OFF). Self-contained: it reads/writes the swarm-manager-local
 * auto-drain endpoint via react-query rather than the proto settings form, so
 * goal-directed execution stays inside the scenario change boundary. When on,
 * the poller continuously enqueues ready goal items through the same governed
 * QueueBacklog path (lane caps, preflight, circuit breaker, cost caps).
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card } from "../ui/card";
import { defaultQueryOptions } from "../../lib";
import { autoDrainService } from "../../services/auto-drain-service";
import { ToggleButtons } from "./ToggleButtons";

const AUTO_DRAIN_QUERY_KEY = ["auto-drain"] as const;

export function GoalDrainToggle() {
  const queryClient = useQueryClient();
  const { data } = useQuery({
    queryKey: AUTO_DRAIN_QUERY_KEY,
    queryFn: () => autoDrainService.get(),
    ...defaultQueryOptions,
  });
  const mutation = useMutation({
    mutationFn: (enabled: boolean) => autoDrainService.set(enabled),
    onSuccess: (next) => queryClient.setQueryData(AUTO_DRAIN_QUERY_KEY, next),
  });

  const enabled = data?.enabled ?? false;

  return (
    <Card data-testid="settings-goal-drain">
      <div>
        <h3 className="text-lg font-medium text-slate-200">Goal-Directed Execution</h3>
        <p className="mt-1 text-sm text-slate-400">
          When enabled, swarm-manager continuously enqueues ready items from your goals (highest
          priority first) toward completion. Every enqueue still passes lane caps, preflight, the
          circuit breaker, and cost caps — this only automates the queueing, not the governance.
        </p>
      </div>
      <div className="mt-4 border-t border-white/5 pt-4">
        <label className="block text-sm font-medium text-slate-300">Continuous auto-enqueue</label>
        <p className="mt-1 text-xs text-slate-400">Default off. Turn on to let goals drain autonomously.</p>
        <ToggleButtons
          value={enabled}
          options={[
            { value: false as const, label: "Off" },
            { value: true as const, label: "On" },
          ]}
          onChange={(v) => mutation.mutate(v)}
        />
      </div>
    </Card>
  );
}
