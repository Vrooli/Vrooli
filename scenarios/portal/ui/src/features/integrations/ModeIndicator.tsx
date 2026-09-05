import { BehaviorMode } from "../../api/integrations";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useIntegrationStatus } from "./useIntegrationStatus";

const MODE_KEY = {
  [BehaviorMode.OFF]: strings.integrations.mode.off,
  [BehaviorMode.PASSIVE]: strings.integrations.mode.passive,
  [BehaviorMode.FULL]: strings.integrations.mode.full,
  [BehaviorMode.UNSPECIFIED]: strings.integrations.mode.unknown,
} as const satisfies Record<BehaviorMode, string>;

export function ModeIndicator() {
  const { t } = useTranslation();
  const { status, loading, error } = useIntegrationStatus();
  const mode = status?.activeMode ?? BehaviorMode.UNSPECIFIED;
  const reason = error || status?.reason || t(strings.integrations.mode.loading);

  return (
    <div
      data-testid={selectors.integrations.modeIndicator}
      title={reason}
      className="flex max-w-72 items-center gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 py-1 text-xs text-app-muted-foreground"
    >
      <span
        aria-hidden="true"
        className={mode === BehaviorMode.PASSIVE ? "size-2 rounded-full bg-app-success" : "size-2 rounded-full bg-app-warning"}
      />
      <span className="font-medium text-app-foreground">{t(strings.integrations.mode.label)}</span>
      <span data-testid={selectors.integrations.modeValue}>
        {loading && !status ? t(strings.integrations.mode.loading) : t(MODE_KEY[mode])}
      </span>
    </div>
  );
}
