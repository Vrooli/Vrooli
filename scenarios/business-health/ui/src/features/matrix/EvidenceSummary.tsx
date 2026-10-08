import type { EvidenceCell } from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

import { StatusChip, type ChipTone } from "../../components/StatusChip";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { timestampToDate } from "../../lib/protoTime";
import { useTranslation } from "../../i18n";

export interface EvidenceSummaryProps {
  readonly evidence?: EvidenceCell;
  /** Verbose renders sync/attestation timestamps; compact shows chips only. */
  readonly verbose?: boolean;
}

const liveTone = (status: string): ChipTone => {
  const normalized = status.toLowerCase();
  if (normalized === "passed" || normalized === "pass") return "success";
  if (normalized === "failed" || normalized === "fail") return "danger";
  if (normalized === "missing" || normalized === "") return "neutral";
  return "warning";
};

const fmt = (date: Date | undefined): string =>
  date ? formatDate(date, { dateStyle: "medium", timeStyle: "short" }) : "";

/**
 * Renders the evidence rollup for a requirement: live suite status, staleness,
 * and manual attestation state. Compact by default (chips only) for dense grid
 * cells; `verbose` adds sync/expiry timestamps for the drawer.
 */
export function EvidenceSummary({ evidence, verbose = false }: EvidenceSummaryProps) {
  const { t } = useTranslation();

  const liveStatus = evidence?.liveStatus ?? "";
  const hasLive = liveStatus !== "";
  const manual = evidence?.manual;
  const hasAny = hasLive || Boolean(manual);

  if (!hasAny) {
    return <StatusChip tone="neutral">{t(strings.matrix.evidence.none)}</StatusChip>;
  }

  const syncedAt = timestampToDate(evidence?.lastSyncedAt);
  const expiresAt = timestampToDate(manual?.expiresAt);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-1.5">
        {hasLive && (
          <StatusChip tone={liveTone(liveStatus)}>
            {t(strings.matrix.evidence.live, { status: liveStatus })}
          </StatusChip>
        )}
        {evidence?.stale && (
          <StatusChip tone="warning">{t(strings.matrix.evidence.stale)}</StatusChip>
        )}
        {manual && (
          <StatusChip tone={manual.expired ? "danger" : "info"}>
            {manual.expired ? t(strings.matrix.evidence.expired) : t(strings.matrix.evidence.manual)}
          </StatusChip>
        )}
      </div>
      {verbose && (
        <div className="flex flex-col gap-0.5 text-xs text-app-muted-foreground">
          {syncedAt && <span>{t(strings.matrix.evidence.syncedAt, { when: fmt(syncedAt) })}</span>}
          {manual?.attestedBy && (
            <span>{t(strings.matrix.evidence.attestedBy, { who: manual.attestedBy })}</span>
          )}
          {manual && expiresAt && (
            <span>{t(strings.matrix.evidence.expires, { when: fmt(expiresAt) })}</span>
          )}
        </div>
      )}
    </div>
  );
}
