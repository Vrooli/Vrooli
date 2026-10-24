import { StatusDot } from "../../../components/ui/status-dot";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

export function TierBadge({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation();
  return enabled ? (
    <StatusDot tone="success" label={t(strings.status.enabled)} />
  ) : (
    <StatusDot tone="neutral" label={t(strings.status.off)} />
  );
}
