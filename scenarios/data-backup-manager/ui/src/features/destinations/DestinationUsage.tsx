/**
 * Live destination usage.
 *
 * The catalog list endpoint is intentionally cheap and may carry a stale
 * `usage_bytes`/`usage_state`; repository usage changes with every write. Each
 * surface that shows usage therefore reads the read-only per-destination usage
 * endpoint, falling back to the catalog value only while that read is in
 * flight. Keeping the chip and meter here means every page reports the same
 * number instead of each re-deriving it.
 */
import { UsageBar } from "../../components/ui/usage-bar";
import { StatusChip } from "../../components/ui/status-chip";
import { useDestinationUsage } from "../../hooks/useDestinations";
import type { Destination } from "../../api/destinations";
import { usageMeta } from "../../lib/status";
import { USAGE_STRINGS } from "../../consts/statusStrings";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function DestinationUsageChip({ destination }: { destination: Destination }) {
  const { data } = useDestinationUsage(destination.id);
  const usage = usageMeta(data?.usageState ?? destination.usageState);
  return <StatusChip tone={usage.tone} labelKey={USAGE_STRINGS[usage.slug]} />;
}

export function DestinationUsageMeter({ destination }: { destination: Destination }) {
  const { t } = useTranslation();
  const { data, isPending, isError } = useDestinationUsage(destination.id);
  // `isPending` means there is no data yet at all; a background refetch keeps
  // showing the last measured value instead of flashing a spinner.
  if (isPending) {
    return <p className="text-xs text-app-muted-foreground">{t(strings.common.loading)}</p>;
  }
  // A failed usage read is real evidence that the repository could not be
  // measured. Showing the stale catalog bytes (often 0) would falsely suggest
  // the drive is empty or healthy.
  if (isError) {
    return (
      <p role="alert" className="text-xs text-app-muted-foreground">
        {t(strings.overview.usageUnavailable)}
      </p>
    );
  }
  return (
    <UsageBar
      usageBytes={data.usageBytes}
      capBytes={data.capBytes}
      usageState={data.usageState}
    />
  );
}
