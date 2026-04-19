import { type DataSourceStatus } from "../lib/api";

interface GapBadgeProps {
  status: DataSourceStatus;
  whatIsNeeded?: string | null;
}

/**
 * Small badge indicating that a metric is partially implemented or entirely
 * missing upstream. Renders nothing for `status === "live"`.
 */
export function GapBadge({ status, whatIsNeeded }: GapBadgeProps) {
  if (status === "live") {
    return null;
  }

  const label = status === "gap" ? "GAP" : "PARTIAL";
  const className = status === "gap" ? "cc-badge cc-badge-gap" : "cc-badge cc-badge-partial";
  const title = whatIsNeeded ?? undefined;

  return (
    <span
      className={className}
      data-testid="gap-badge"
      data-status={status}
      title={title}
    >
      {label}
    </span>
  );
}
