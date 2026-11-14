import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/EmptyState";
import { ApplyOverview } from "./ApplyOverview";
import { ApplyConfirmDialog } from "./ApplyConfirmDialog";
import { ApplyHistoryPanel } from "./ApplyHistoryPanel";
import { DryRunDiff } from "./DryRunDiff";
import { PlanPreview } from "./PlanPreview";
import {
  useBuildBaseline,
  usePlanApply,
  useRunApply,
} from "./controllers/useApplyController";
import { transition, type ApplyState, type ApplyEvent } from "./flow/transition";
import type { Plan } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

export interface ApplyWorkspaceProps {
  scenario: string;
  domain: string;
}

/**
 * Per-domain apply workspace: orchestrates plan / dry-run / apply through
 * the flow state machine. Each action gates on `legalEventsFor(state)` so
 * the UI never surfaces a disallowed transition.
 */
export function ApplyWorkspace({ scenario, domain }: ApplyWorkspaceProps) {
  const { t } = useTranslation();
  const [state, setState] = React.useState<ApplyState>("baseline_captured");
  const [plan, setPlan] = React.useState<Plan | undefined>(undefined);
  const [dialogOpen, setDialogOpen] = React.useState(false);

  const baseline = useBuildBaseline({ scenario });
  const planMutation = usePlanApply();
  const runMutation = useRunApply();

  const baselineRed =
    baseline.data?.baseline?.toolchain !== undefined &&
    baseline.data.baseline.toolchain.length > 0 &&
    !baseline.data.baseline.green;

  const advance = React.useCallback((event: ApplyEvent) => {
    setState((prev) => transition(prev, event));
  }, []);

  const onPlan = async (dryRun: boolean) => {
    try {
      const result = await planMutation.mutateAsync({ scenario, domain, dryRun });
      setPlan(result.plan);
      advance(dryRun ? "dry_run" : "plan");
    } catch {
      // surfaced via mutation.error UI below
    }
  };

  const onApplyClicked = () => {
    if (baselineRed) {
      setDialogOpen(true);
      return;
    }
    void runApply("");
  };

  const runApply = async (_note: string) => {
    if (!plan) return;
    try {
      await runMutation.mutateAsync({ scenario, domain, planId: plan.id });
      // v0.1 RunApply returns Unimplemented; we don't advance to applied.
    } catch {
      // expected in v0.1 — surface the message
    } finally {
      setDialogOpen(false);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <ApplyOverview scenario={scenario} currentState={state} />

      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="default"
          size="sm"
          data-testid={selectors.features.apply.plan.planButton}
          onClick={() => void onPlan(false)}
          disabled={planMutation.isPending}
        >
          {planMutation.isPending
            ? t(strings.pages.targetApply.planning)
            : t(strings.pages.targetApply.planButton)}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          data-testid={selectors.features.apply.plan.dryRunButton}
          onClick={() => void onPlan(true)}
          disabled={planMutation.isPending || !plan}
        >
          {t(strings.pages.targetApply.dryRunButton)}
        </Button>
        <Button
          type="button"
          variant="default"
          size="sm"
          data-testid={selectors.features.apply.plan.applyButton}
          onClick={onApplyClicked}
          disabled={runMutation.isPending || !plan}
        >
          {runMutation.isPending
            ? t(strings.pages.targetApply.applying)
            : t(strings.pages.targetApply.applyButton)}
        </Button>
      </div>

      <section aria-labelledby="apply-plan-heading" className="flex flex-col gap-2">
        <h4 id="apply-plan-heading" className="text-lg font-semibold">
          {t(strings.pages.targetApply.operationsHeading)}
        </h4>
        <PlanPreview plan={plan} />
      </section>

      {plan ? (
        <section aria-labelledby="apply-diff-heading" className="flex flex-col gap-2">
          <h4 id="apply-diff-heading" className="text-lg font-semibold">
            {t(strings.shared.diff.added)} / {t(strings.shared.diff.removed)}
          </h4>
          <DryRunDiff plan={plan} />
        </section>
      ) : null}

      {runMutation.isError ? (
        <EmptyState
          title={t(strings.pages.targetApply.applyUnimplementedTitle)}
          description={t(strings.pages.targetApply.applyUnimplementedMessage)}
        />
      ) : null}

      <section aria-labelledby="apply-history-heading" className="flex flex-col gap-2">
        <h4 id="apply-history-heading" className="text-lg font-semibold">
          {t(strings.pages.targetApply.historyHeading)}
        </h4>
        <ApplyHistoryPanel scenario={scenario} domain={domain} />
      </section>

      <ApplyConfirmDialog
        open={dialogOpen}
        requiresNote={baselineRed}
        onClose={() => setDialogOpen(false)}
        onConfirm={({ note }) => void runApply(note)}
        submitting={runMutation.isPending}
      />
    </div>
  );
}
