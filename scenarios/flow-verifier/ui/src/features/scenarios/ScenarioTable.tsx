// ScenarioTable is the searchable list body used by /scenarios. The
// grid card layout still lives on the page as a fallback for empty
// states; this component owns the row-per-scenario table with
// checkboxes for the bulk-action surface.
import { Link } from "react-router-dom";
import { AlertTriangle } from "lucide-react";

import type { ScenarioSummary } from "../../api/scenarios";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";

interface Props {
  scenarios: ScenarioSummary[];
  selectedIds: Set<string>;
  onToggleOne: (id: string) => void;
  onToggleAll: (selectAll: boolean) => void;
}

export function ScenarioTable({ scenarios, selectedIds, onToggleOne, onToggleAll }: Props) {
  const { t } = useTranslation();
  const allSelected = scenarios.length > 0 && scenarios.every((s) => selectedIds.has(s.id));
  const someSelected = selectedIds.size > 0 && !allSelected;

  return (
    <table data-testid="scenario-table" className="w-full border-collapse text-left text-sm">
      <thead>
        <tr className="border-b border-app-border text-xs uppercase tracking-wide text-app-muted-foreground">
          <th className="w-8 py-2 pr-2">
            <input
              type="checkbox"
              data-testid="scenario-toggle-all"
              aria-label={t("scenarios.toggleAll", { defaultValue: "Select all scenarios" })}
              checked={allSelected}
              ref={(el) => {
                if (el) el.indeterminate = someSelected;
              }}
              onChange={(e) => onToggleAll(e.target.checked)}
            />
          </th>
          <th className="py-2 pr-3 font-medium">
            {t("scenarios.col.name", { defaultValue: "Scenario" })}
          </th>
          <th className="py-2 pr-3 font-medium">
            {t("scenarios.col.flows", { defaultValue: "Flows" })}
          </th>
          <th className="py-2 pr-3 font-medium">
            {t("scenarios.col.description", { defaultValue: "Description" })}
          </th>
          <th className="py-2 font-medium">
            {t("scenarios.col.path", { defaultValue: "Path" })}
          </th>
        </tr>
      </thead>
      <tbody>
        {scenarios.map((s) => {
          const checked = selectedIds.has(s.id);
          return (
            <tr
              key={s.id}
              data-testid={`scenario-row-${s.id}`}
              className="border-b border-app-border last:border-b-0"
            >
              <td className="py-2 pr-2 align-top">
                <input
                  type="checkbox"
                  data-testid={`scenario-select-${s.id}`}
                  aria-label={t("scenarios.selectRow", {
                    defaultValue: "Select {{name}}",
                    name: s.displayName,
                  })}
                  checked={checked}
                  onChange={() => onToggleOne(s.id)}
                />
              </td>
              <td className="py-2 pr-3 align-top">
                <Link
                  to={ROUTES.scenarioDetail(encodeURIComponent(s.id))}
                  data-testid={`scenario-link-${s.id}`}
                  className="font-medium text-app-primary hover:underline"
                >
                  {s.displayName}
                </Link>
                <div className="font-mono text-xs text-app-muted-foreground">{s.id}</div>
              </td>
              <td className="py-2 pr-3 align-top text-xs">
                {s.discoveryError ? (
                  <span
                    data-testid={`scenario-row-error-${s.id}`}
                    className="inline-flex items-center gap-1 text-app-danger"
                  >
                    <AlertTriangle className="h-3 w-3" />
                    {s.discoveryError}
                  </span>
                ) : (
                  <span data-testid={`scenario-row-flowcount-${s.id}`} className="text-app-muted-foreground">
                    {s.flowCount}
                  </span>
                )}
              </td>
              <td className="py-2 pr-3 align-top text-xs text-app-muted-foreground">
                {s.description ?? "—"}
              </td>
              <td className="py-2 text-right text-xs">
                <span className="font-mono text-app-muted-foreground">{s.path}</span>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
