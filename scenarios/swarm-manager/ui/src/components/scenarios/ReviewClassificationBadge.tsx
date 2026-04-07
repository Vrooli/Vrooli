import {
  REVIEW_CLASSIFICATION_LABELS,
  REVIEW_CLASSIFICATION_COLORS,
} from "../../types";
import { formatRelativeTime } from "../../lib";

interface ReviewClassificationBadgeProps {
  classification: string;
  reviewedAt?: string;
  showTimestamp?: boolean;
}

const validClassifications = new Set<string>(Object.keys(REVIEW_CLASSIFICATION_LABELS));

export function ReviewClassificationBadge({
  classification,
  reviewedAt,
  showTimestamp = false,
}: ReviewClassificationBadgeProps) {
  const isKnown = validClassifications.has(classification);
  const colors = isKnown ? REVIEW_CLASSIFICATION_COLORS[classification] : "bg-slate-500/20 text-slate-400";
  const label = isKnown ? REVIEW_CLASSIFICATION_LABELS[classification] : "Unknown";

  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${colors}`}>
      {label}
      {showTimestamp && reviewedAt && (
        <span className="text-slate-500">
          &middot; {formatRelativeTime(reviewedAt)}
        </span>
      )}
    </span>
  );
}
