import { BellRing } from "lucide-react";
import { Link } from "react-router-dom";
import { strings } from "../../consts/strings";
import { useAttention } from "../../features/health/useAttention";
import { useTranslation } from "../../i18n";

export function AttentionPill() {
  const { t } = useTranslation();
  const { pending } = useAttention();
  if (!pending) return null;
  return (
    <Link to="/" data-testid="conversations-attention" className="inline-flex min-h-11 items-center gap-1.5 rounded-pill border border-app-warning/50 bg-app-warning/10 px-3 text-xs font-semibold text-app-warning">
      <BellRing aria-hidden="true" className="h-3.5 w-3.5" />
      {t(strings.console.attention.pendingCount, { count: pending })}
    </Link>
  );
}
