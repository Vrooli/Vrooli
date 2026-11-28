import { useNow } from "../hooks/useNow";
import { formatAgo, formatExact } from "../lib/ago";
import { useTranslation } from "../i18n";
import { strings } from "../consts/strings";

/**
 * A timestamp as "12 minutes ago" with the exact local time in the tooltip and
 * in `dateTime`, so it stays glanceable and checkable.
 */
export function Time({ value, className }: { value: string | undefined; className?: string }) {
  const { t } = useTranslation();
  const now = useNow();
  const ago = formatAgo(value, now);
  if (!ago) return <span className={className}>{t(strings.common.timeUnavailable)}</span>;
  return <time dateTime={value} title={formatExact(value) ?? undefined} className={className}>{ago}</time>;
}
