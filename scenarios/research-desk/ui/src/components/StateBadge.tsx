import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import type { CommissionState } from "../api/activity";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { STATE_META } from "../lib/states";

export function StateBadge({ state, stalled = false, pastAllowance = false }: { state: CommissionState; stalled?: boolean; pastAllowance?: boolean }) {
  const { t } = useTranslation();
  const meta = STATE_META[state];
  return (
    <span className="inline-flex flex-wrap items-center gap-space-3xs">
      <StatusBadge tone={meta.tone} data-state={meta.label}>{t(meta.label)}</StatusBadge>
      {stalled ? <StatusBadge tone="warning">{t(strings.state.stalled)}</StatusBadge> : null}
      {pastAllowance ? <StatusBadge tone="warning">{t(strings.state.pastAllowance)}</StatusBadge> : null}
    </span>
  );
}
