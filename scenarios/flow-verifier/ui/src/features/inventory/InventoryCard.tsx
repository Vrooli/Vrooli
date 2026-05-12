import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import {
  fetchFlows,
  fetchRuns,
  verifyFlow,
  type FlowSummary,
  type RunRow,
} from "../../api/inventory";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";

const DEFAULT_ROOT = ".";

const FLOWS_KEY = (root: string) => ["flows", root] as const;
const RUNS_KEY = ["runs", "all"] as const;

/**
 * InventoryCard is the Phase F MVP for flow-verifier: it lists every
 * flow discovered under a configurable root and joins each row with the
 * most recent verification run from /api/v1/runs. "Verify all" posts
 * /api/v1/verifications for each flow sequentially and refreshes both
 * queries when each finishes.
 */
export function InventoryCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [root, setRoot] = useState(DEFAULT_ROOT);

  const flowsQuery = useQuery({
    queryKey: FLOWS_KEY(root),
    queryFn: () => fetchFlows(root),
  });

  const runsQuery = useQuery({
    queryKey: RUNS_KEY,
    queryFn: () => fetchRuns({ limit: 200 }),
  });

  const latestByFlow = useMemo(() => {
    const m = new Map<string, RunRow>();
    for (const run of runsQuery.data ?? []) {
      const existing = m.get(run.flowId);
      if (!existing || run.finishedAt > existing.finishedAt) m.set(run.flowId, run);
    }
    return m;
  }, [runsQuery.data]);

  const verifyAll = useMutation({
    mutationFn: async (flows: FlowSummary[]) => {
      for (const flow of flows) {
        await verifyFlow(root, flow.flowId);
        await queryClient.invalidateQueries({ queryKey: RUNS_KEY });
      }
    },
  });

  const flows = flowsQuery.data ?? [];

  return (
    <section
      data-testid="inventory-card"
      aria-label={t("inventory.title", { defaultValue: "Flow Inventory" })}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">
        {t("inventory.title", { defaultValue: "Flow Inventory" })}
      </h2>

      <div className="mt-3 flex items-end gap-3">
        <label className="flex flex-1 flex-col gap-1 text-xs text-slate-400">
          <span>{t("inventory.rootLabel", { defaultValue: "Root path" })}</span>
          <input
            data-testid="inventory-root"
            value={root}
            onChange={(e) => setRoot(e.target.value)}
            className="rounded-control border border-white/10 bg-app-surface-muted px-2 py-1 text-sm text-app-foreground"
          />
        </label>
        <Button
          data-testid="inventory-reload"
          onClick={() => {
            void flowsQuery.refetch();
            void runsQuery.refetch();
          }}
        >
          {t("inventory.reload", { defaultValue: "Reload" })}
        </Button>
        <Button
          data-testid="inventory-verify-all"
          onClick={() => verifyAll.mutate(flows)}
          disabled={verifyAll.isPending || flows.length === 0}
        >
          {verifyAll.isPending
            ? t("inventory.verifyingAll", { defaultValue: "Verifying…" })
            : t("inventory.verifyAll", { defaultValue: "Verify all" })}
        </Button>
      </div>

      {flowsQuery.isLoading && (
        <p data-testid="inventory-loading" className="mt-3 text-slate-200">
          {t("inventory.loading", { defaultValue: "Discovering flows…" })}
        </p>
      )}
      {flowsQuery.error && (
        <p data-testid="inventory-error" className="mt-3 text-red-400">
          {errorMessage(flowsQuery.error, t)}
        </p>
      )}
      {!flowsQuery.isLoading && flows.length === 0 && !flowsQuery.error && (
        <p data-testid="inventory-empty" className="mt-3 text-slate-200">
          {t("inventory.empty", { defaultValue: "No flows discovered under this root." })}
        </p>
      )}

      {flows.length > 0 && (
        <table
          data-testid="inventory-table"
          className="mt-4 w-full text-left text-sm text-slate-200"
        >
          <thead className="text-xs uppercase text-slate-400">
            <tr>
              <th className="py-1 pr-3">{t("inventory.colFlow", { defaultValue: "Flow" })}</th>
              <th className="py-1 pr-3">{t("inventory.colLang", { defaultValue: "Lang" })}</th>
              <th className="py-1 pr-3">{t("inventory.colStatus", { defaultValue: "Last status" })}</th>
              <th className="py-1 pr-3">{t("inventory.colWhen", { defaultValue: "When" })}</th>
            </tr>
          </thead>
          <tbody>
            {flows.map((flow) => {
              const last = latestByFlow.get(flow.flowId);
              return (
                <tr
                  key={flow.flowId}
                  data-testid={`inventory-row-${flow.flowId}`}
                  className="border-t border-white/5"
                >
                  <td className="py-1 pr-3 font-medium">{flow.flowId}</td>
                  <td className="py-1 pr-3">{flow.language}</td>
                  <td className="py-1 pr-3" data-testid={`inventory-status-${flow.flowId}`}>
                    {last ? last.status : "—"}
                  </td>
                  <td className="py-1 pr-3 text-xs text-slate-400">
                    {last ? new Date(last.finishedAt).toLocaleString() : ""}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}

      {verifyAll.error && (
        <p data-testid="inventory-verify-error" className="mt-3 text-red-400">
          {errorMessage(verifyAll.error, t)}
        </p>
      )}
    </section>
  );
}
