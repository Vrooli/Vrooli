import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { Badge } from "../../components/ui/badge";
import { useApplyHistory } from "./controllers/useApplyController";
import { statusToState, type ApplyState } from "./flow/transition";
import type { ApplyRun } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

const STATE_LABEL_KEY = {
  baseline_captured: strings.apply.status.baseline_captured,
  plan_generated: strings.apply.status.plan_generated,
  dry_run_ok: strings.apply.status.dry_run_ok,
  applied: strings.apply.status.applied,
  committed: strings.apply.status.committed,
  refused_build_break: strings.apply.status.refused_build_break,
  force_committed: strings.apply.status.force_committed,
} as const satisfies Record<ApplyState, string>;

export interface ApplyHistoryPanelProps {
  scenario: string;
  domain: string;
}

export function ApplyHistoryPanel({ scenario, domain }: ApplyHistoryPanelProps) {
  const { t } = useTranslation();
  const history = useApplyHistory({ scenario, domain });

  if (history.isPending) {
    return (
      <div data-testid={selectors.features.apply.history.loading}>
        <LoadingState label={t(strings.pages.targetApply.loading)} />
      </div>
    );
  }
  if (history.isError) {
    return (
      <div data-testid={selectors.features.apply.history.error}>
        <ErrorState
          title={t(strings.pages.targetApply.errorTitle)}
          message={history.error instanceof Error ? history.error.message : String(history.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void history.refetch();
          }}
        />
      </div>
    );
  }

  const runs = history.data.runs;
  if (runs.length === 0) {
    return (
      <div data-testid={selectors.features.apply.history.empty}>
        <EmptyState title={t(strings.pages.targetApply.historyEmpty)} />
      </div>
    );
  }

  const columns: ReadonlyArray<DataTableColumn<ApplyRun>> = [
    {
      key: "id",
      header: "ID",
      cell: (row) => <span className="font-mono text-xs">{row.id}</span>,
    },
    {
      key: "status",
      header: "Status",
      cell: (row) => {
        const state = statusToState(row.status);
        return <Badge variant="default">{t(STATE_LABEL_KEY[state])}</Badge>;
      },
    },
    {
      key: "domain",
      header: t(strings.pages.targetApply.domainHeading),
      cell: (row) => <span className="text-sm">{row.domain}</span>,
    },
  ];

  return (
    <div data-testid={selectors.features.apply.history.root}>
      <DataTable
        rows={runs}
        getRowId={(r) => r.id || `${r.scenario}-${r.domain}`}
        columns={columns}
        emptyMessage={t(strings.pages.targetApply.historyEmpty)}
      />
    </div>
  );
}
