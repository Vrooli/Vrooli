import type { TFunction } from "i18next";
import type { Exposure } from "@vrooli/proto-types/tunnel-manager/v1/exposure/exposure_pb";
import type { RouteClassification } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { Button } from "./button";
import { StatusBadge, type BadgeTone } from "./StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { failureClassLabel, failureClassTone } from "../../features/metrics/labels";

/**
 * Custom route inventory primitive.
 *
 * DataTable@1.3.0 was reviewed for this surface, but its preflight contract
 * requires twenty design tokens that Tunnel Manager does not define. This
 * route-specific table keeps the richer action cells, status semantics, and
 * responsive companion cards without introducing fallback styling or a second
 * token vocabulary.
 */

export interface RouteInventoryRow {
  exposure: Exposure;
  classification?: RouteClassification;
}

interface RouteInventoryTableProps {
  rows: readonly RouteInventoryRow[];
  t: TFunction;
  onSelect: (exposure: Exposure) => void;
  onExtend: (leaseId: string) => void;
  onRevoke: (leaseId: string) => void;
  extendPending: boolean;
  revokePending: boolean;
}

function tierTone(tier: string): BadgeTone {
  if (tier === "core") return "info";
  if (tier === "leased") return "success";
  return "neutral";
}

function tierLabel(tier: string) {
  if (tier === "core") return strings.exposure.tier.core;
  if (tier === "leased") return strings.exposure.tier.leased;
  return strings.exposure.tier.unknown;
}

export function RouteInventoryTable({
  rows,
  t,
  onSelect,
  onExtend,
  onRevoke,
  extendPending,
  revokePending,
}: RouteInventoryTableProps) {
  return (
    <div className="hidden overflow-x-auto rounded-panel border border-app-border md:block">
      <table data-testid={selectors.exposure.table} className="w-full text-left text-sm">
        <caption className="sr-only">{t(strings.exposure.heading)}</caption>
        <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
          <tr>
            <th scope="col" className="px-3 py-2">{t(strings.exposure.colScenario)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.exposure.colTier)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.exposure.colHealth)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.exposure.colUrl)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.exposure.colPort)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.exposure.colLease)}</th>
            <th scope="col" className="px-3 py-2">{t(strings.exposure.colActions)}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(({ exposure, classification }) => {
            const lease = exposure.lease;
            return (
              <tr key={exposure.scenario} data-testid={selectors.exposure.row} className="border-b border-app-border last:border-0">
                <th scope="row" className="px-3 py-3 font-medium">
                  <button type="button" className="break-words text-left text-app-primary underline-offset-2 hover:underline" onClick={() => onSelect(exposure)}>
                    {exposure.scenario}
                  </button>
                </th>
                <td className="px-3 py-3">
                  <StatusBadge tone={tierTone(exposure.tier)} data-testid={selectors.exposure.tierBadge}>{t(tierLabel(exposure.tier))}</StatusBadge>
                </td>
                <td className="px-3 py-3">
                  {classification ? <StatusBadge tone={failureClassTone(classification.classification)} data-testid={selectors.exposure.healthBadge}>{t(failureClassLabel(classification.classification))}</StatusBadge> : <StatusBadge tone="neutral" data-testid={selectors.exposure.healthBadge}>{t(strings.exposure.healthUnknown)}</StatusBadge>}
                </td>
                <td className="max-w-[18rem] break-words px-3 py-3">
                  <a data-testid={selectors.exposure.url} href={exposure.publicUrl} target="_blank" rel="noreferrer" className="break-words text-app-primary underline-offset-2 hover:underline">{exposure.publicUrl}</a>
                </td>
                <td className="px-3 py-3 tabular-nums">{exposure.localPort}</td>
                <td data-testid={selectors.exposure.leaseExpiry} className="px-3 py-3">
                  {lease?.expiresAt ? t(strings.exposure.leaseActive, { when: formatDate(timestampDate(lease.expiresAt), { dateStyle: "medium", timeStyle: "short" }) }) : t(strings.exposure.leaseNone)}
                </td>
                <td className="px-3 py-3">
                  {lease ? <div className="flex gap-2"><Button variant="outline" data-testid={selectors.exposure.extendButton} disabled={extendPending} onClick={() => onExtend(lease.id)}>{t(strings.exposure.extendButton)}</Button><Button variant="outline" data-testid={selectors.exposure.revokeButton} disabled={revokePending} onClick={() => onRevoke(lease.id)}>{t(strings.exposure.revokeButton)}</Button></div> : null}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
