import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const STATUS_LABEL = {
  [FindingStatus.ACTIVE]: strings.findings.statusActive,
  [FindingStatus.DISPUTED]: strings.findings.statusDisputed,
  [FindingStatus.SUPERSEDED]: strings.findings.statusSuperseded,
  [FindingStatus.UNSPECIFIED]: strings.findings.statusUnspecified,
} as const;

// active = neutral, disputed = warning, superseded = muted (per the plan).
const STATUS_CLASS = {
  [FindingStatus.ACTIVE]: "border-app-border bg-app-surface-muted text-app-foreground",
  [FindingStatus.DISPUTED]: "border-app-warning bg-app-warning/15 text-app-warning",
  [FindingStatus.SUPERSEDED]: "border-app-border bg-transparent text-app-muted-foreground",
  [FindingStatus.UNSPECIFIED]: "border-app-border bg-transparent text-app-muted-foreground",
} as const;

/**
 * StatusBadge renders a finding's lifecycle state as a small pill. Used both in
 * the findings management list and in the learnings search-hit list. Disputed
 * findings render in the warning tone; superseded render muted.
 */
export function StatusBadge({ status }: { status: FindingStatus }) {
  const { t } = useTranslation();

  return (
    <span
      data-testid={selectors.findings.statusBadge}
      data-status={FindingStatus[status]}
      className={`inline-block rounded-pill border px-2 py-0.5 text-xs font-medium ${STATUS_CLASS[status]}`}
    >
      {t(STATUS_LABEL[status])}
    </span>
  );
}
