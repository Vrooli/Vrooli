import { cn } from "../../lib/utils";
import { TONE_BG_CLASS, usageMeta } from "../../lib/status";
import { UsageState } from "@vrooli/proto-types/data-backup-manager/v1/destinations/destinations_pb";
import { formatBytes } from "../../lib/format";
import { USAGE_STRINGS } from "../../consts/statusStrings";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Destination storage usage as a labeled bar, colored by `usageState` (the
 * server-computed within/near/over classification, not a client threshold).
 * An uncapped destination (capBytes === 0) shows just the used figure with no
 * fill, since there is no cap to approach.
 */
export function UsageBar({
  usageBytes,
  capBytes,
  usageState,
  className,
}: {
  usageBytes: bigint;
  capBytes: bigint;
  usageState: UsageState;
  className?: string;
}) {
  const { t } = useTranslation();
  const meta = usageMeta(usageState);
  const cap = Number(capBytes);
  const used = Number(usageBytes);
  const capped = cap > 0;
  const pct = capped ? Math.min(100, Math.max(0, (used / cap) * 100)) : 0;

  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <div
        className="h-1.5 w-full overflow-hidden rounded-pill bg-app-surface-muted"
        role="progressbar"
        aria-valuenow={Math.round(pct)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={t(USAGE_STRINGS[meta.slug])}
      >
        {capped && <div className={cn("h-full rounded-pill", TONE_BG_CLASS[meta.tone])} style={{ width: `${pct}%` }} />}
      </div>
      <p className="text-xs text-app-muted-foreground">
        {capped
          ? t(strings.overview.usageOfCap, {
              used: formatBytes(usageBytes),
              cap: formatBytes(capBytes),
            })
          : t(strings.overview.usageUncapped, { used: formatBytes(usageBytes) })}
      </p>
    </div>
  );
}
