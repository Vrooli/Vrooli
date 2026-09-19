import type { IngressEntry } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { Button } from "../../components/ui/button";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ownershipStateLabel, ownershipStateTone, ingressSourceLabel } from "./labels";

interface IngressDriftTableProps {
  entries: readonly IngressEntry[];
  actionPending: boolean;
  onAdopt: (hostname: string) => void;
  onIgnore: (hostname: string) => void;
  onPrune: (hostname: string) => void;
}

/** Accessible, action-oriented drift inventory for the governance surface. */
export function IngressDriftTable({ entries, actionPending, onAdopt, onIgnore, onPrune }: IngressDriftTableProps) {
  const { t } = useTranslation();

  return (
    <div className="overflow-x-auto rounded-panel border border-app-border">
      <table data-testid={selectors.drift.table} className="w-full text-left text-sm">
        <caption className="sr-only">{t(strings.drift.heading)}</caption>
        <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
          <tr>
            <th scope="col" className="px-3 py-2">{t(strings.drift.colHostname)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.drift.colTarget)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.drift.colState)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.drift.colSource)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.drift.colActions)}</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr key={entry.hostname} data-testid={selectors.drift.row} className="border-b border-app-border last:border-0">
              <th scope="row" data-testid={selectors.drift.hostname} className="px-3 py-2 text-left font-medium">{entry.hostname}</th>
              <td data-testid={selectors.drift.target} className="px-3 py-2 text-app-muted-foreground">{entry.serviceTarget || "—"}</td>
              <td className="px-3 py-2">
                <StatusBadge tone={ownershipStateTone(entry.state)} data-testid={selectors.drift.stateBadge}>{t(ownershipStateLabel(entry.state))}</StatusBadge>
              </td>
              <td className="px-3 py-2">
                <StatusBadge tone="neutral" data-testid={selectors.drift.sourceBadge}>{t(ingressSourceLabel(entry.source))}</StatusBadge>
              </td>
              <td className="px-3 py-2">
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" data-testid={selectors.drift.adoptButton({ hostname: entry.hostname })} disabled={actionPending} onClick={() => onAdopt(entry.hostname)}>{t(strings.drift.adoptButton)}</Button>
                  <Button variant="outline" data-testid={selectors.drift.ignoreButton({ hostname: entry.hostname })} disabled={actionPending} onClick={() => onIgnore(entry.hostname)}>{t(strings.drift.ignoreButton)}</Button>
                  <Button variant="outline" data-testid={selectors.drift.pruneButton({ hostname: entry.hostname })} disabled={actionPending} onClick={() => onPrune(entry.hostname)}>{t(strings.drift.pruneButton)}</Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
