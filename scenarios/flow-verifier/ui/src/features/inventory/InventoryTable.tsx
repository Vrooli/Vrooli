import { Link } from "react-router-dom";

import type { FlowSummary, RunRow } from "../../api/inventory";
import { useTranslation } from "../../i18n";

interface Props {
  flows: FlowSummary[];
  latestByFlow: Map<string, RunRow>;
  onVerifyOne: (flowId: string) => void;
  verifyingFlowId?: string;
  anyPending: boolean;
}

function statusTone(status?: string): string {
  if (status === "passed") return "bg-app-success/15 text-app-success";
  if (status === "failed") return "bg-app-danger/15 text-app-danger";
  if (status === "error") return "bg-app-warning/15 text-app-warning";
  return "bg-app-surface-muted text-app-muted-foreground";
}

export function InventoryTable({
  flows,
  latestByFlow,
  onVerifyOne,
  verifyingFlowId,
  anyPending,
}: Props) {
  const { t } = useTranslation();

  return (
    <div className="overflow-x-auto rounded-panel border border-app-border bg-app-surface">
      <table data-testid="inventory-table" className="w-full text-left text-sm text-app-foreground">
        <thead className="bg-app-surface-muted text-xs uppercase tracking-wide text-app-muted-foreground">
          <tr>
            <th className="px-3 py-2 font-medium">{t("inventory.colFlow", { defaultValue: "Flow" })}</th>
            <th className="px-3 py-2 font-medium">{t("inventory.colLang", { defaultValue: "Lang" })}</th>
            <th className="px-3 py-2 font-medium">{t("inventory.colStatus", { defaultValue: "Last status" })}</th>
            <th className="px-3 py-2 font-medium">{t("inventory.colWhen", { defaultValue: "When" })}</th>
            <th className="px-3 py-2 text-right font-medium">
              {t("inventory.colActions", { defaultValue: "Actions" })}
            </th>
          </tr>
        </thead>
        <tbody>
          {flows.map((flow) => {
            const last = latestByFlow.get(flow.flowId);
            const status = last?.status ?? "none";
            return (
              <tr
                key={flow.flowId}
                data-testid={`inventory-row-${flow.flowId}`}
                className="border-t border-app-border hover:bg-app-surface-muted"
              >
                <td className="px-3 py-2 font-medium">
                  <Link
                    data-testid={`inventory-link-${flow.flowId}`}
                    to={`/flows/${flow.flowId}`}
                    className="text-app-primary hover:underline"
                  >
                    {flow.flowId}
                  </Link>
                </td>
                <td className="px-3 py-2 text-app-muted-foreground">{flow.language}</td>
                <td className="px-3 py-2">
                  <span
                    data-testid={`inventory-status-${flow.flowId}`}
                    className={`inline-flex items-center rounded-pill px-2.5 py-0.5 text-xs font-medium ${statusTone(last?.status)}`}
                  >
                    {status === "none" ? "—" : status}
                  </span>
                </td>
                <td className="px-3 py-2 text-xs text-app-muted-foreground">
                  {last ? new Date(last.finishedAt).toLocaleString() : ""}
                </td>
                <td className="px-3 py-2 text-right">
                  <button
                    type="button"
                    data-testid={`inventory-verify-${flow.flowId}`}
                    onClick={() => onVerifyOne(flow.flowId)}
                    disabled={anyPending}
                    className="inline-flex h-8 items-center rounded-control border border-app-border bg-app-surface px-3 text-xs text-app-foreground hover:bg-app-surface-muted disabled:opacity-60"
                  >
                    {verifyingFlowId === flow.flowId
                      ? t("inventory.verifyingOne", { defaultValue: "Verifying…" })
                      : t("inventory.verifyOne", { defaultValue: "Verify" })}
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
