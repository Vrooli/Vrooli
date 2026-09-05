import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Badge } from "../../components/ui/badge";
import { useBuildBaseline } from "./controllers/useApplyController";
import { type ApplyState } from "./flow/transition";

const STATE_LABEL_KEY = {
  baseline_captured: strings.apply.status.baseline_captured,
  plan_generated: strings.apply.status.plan_generated,
  dry_run_ok: strings.apply.status.dry_run_ok,
  applied: strings.apply.status.applied,
  committed: strings.apply.status.committed,
  refused_build_break: strings.apply.status.refused_build_break,
  force_committed: strings.apply.status.force_committed,
} as const satisfies Record<ApplyState, string>;

export interface ApplyOverviewProps {
  scenario: string;
  /** Current UI flow state (derived from the latest run, if any). */
  currentState: ApplyState;
}

export function ApplyOverview({ scenario, currentState }: ApplyOverviewProps) {
  const { t } = useTranslation();
  const baseline = useBuildBaseline({ scenario });
  const data = baseline.data?.baseline;

  let baselineText: string;
  let baselineVariant: "success" | "danger" | "muted" = "muted";
  if (!data || !data.toolchain) {
    baselineText = t(strings.pages.targetApply.baselineUnknown);
  } else if (data.green) {
    baselineText = t(strings.pages.targetApply.baselineGreen, { toolchain: data.toolchain });
    baselineVariant = "success";
  } else {
    baselineText = t(strings.pages.targetApply.baselineRed, { toolchain: data.toolchain });
    baselineVariant = "danger";
  }

  return (
    <section
      data-testid={selectors.features.apply.overview.root}
      aria-label={t(strings.pages.targetApply.baselineHeading)}
      className="flex flex-wrap items-center gap-2 rounded-panel border border-app-border bg-app-surface p-3 text-sm"
    >
      <span className="text-app-muted-foreground">
        {t(strings.pages.targetApply.baselineHeading)}:
      </span>
      <span
        data-testid={selectors.features.apply.overview.baseline}
        className={
          baselineVariant === "success"
            ? "text-app-success"
            : baselineVariant === "danger"
              ? "text-app-danger"
              : "text-app-muted-foreground"
        }
      >
        {baselineText}
      </span>
      <span data-testid={selectors.features.apply.overview.state}>
        <Badge variant="default">{t(STATE_LABEL_KEY[currentState])}</Badge>
      </span>
    </section>
  );
}
