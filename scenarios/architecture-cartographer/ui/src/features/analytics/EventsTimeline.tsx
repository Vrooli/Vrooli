import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { useEvents } from "./controllers/useAnalyticsController";
import {
  EventKind,
  type Event,
} from "@vrooli/proto-types/architecture-cartographer/v1/analytics/analytics_pb";

function kindLabelKey(kind: EventKind) {
  switch (kind) {
    case EventKind.CONFLICT_DETECTED:
      return strings.analytics.eventKind.conflict_detected;
    case EventKind.CONFLICT_ASSIGNED:
      return strings.analytics.eventKind.conflict_assigned;
    case EventKind.CONFLICT_RESOLVED:
      return strings.analytics.eventKind.conflict_resolved;
    case EventKind.CONFLICT_REOPENED:
      return strings.analytics.eventKind.conflict_reopened;
    case EventKind.CONFLICT_FORCE_RESOLVED:
      return strings.analytics.eventKind.conflict_force_resolved;
    case EventKind.VERDICT_PRODUCED:
      return strings.analytics.eventKind.verdict_produced;
    case EventKind.PLACEMENT_AUTO:
      return strings.analytics.eventKind.placement_auto;
    case EventKind.PLACEMENT_SUGGEST:
      return strings.analytics.eventKind.placement_suggest;
    case EventKind.OVERRIDE_RECORDED:
      return strings.analytics.eventKind.override_recorded;
    case EventKind.APPLY_PLANNED:
      return strings.analytics.eventKind.apply_planned;
    case EventKind.APPLY_RAN:
      return strings.analytics.eventKind.apply_ran;
    case EventKind.APPLY_BUILD_GREEN:
      return strings.analytics.eventKind.apply_build_green;
    case EventKind.APPLY_BUILD_RED:
      return strings.analytics.eventKind.apply_build_red;
    case EventKind.APPLY_REVERTED:
      return strings.analytics.eventKind.apply_reverted;
    default:
      return strings.analytics.eventKind.unspecified;
  }
}

export interface EventsTimelineProps {
  scenario: string;
}

export function EventsTimeline({ scenario }: EventsTimelineProps) {
  const { t } = useTranslation();
  const events = useEvents(scenario);

  if (events.isPending) {
    return (
      <div data-testid={selectors.features.analytics.events.loading}>
        <LoadingState label={t(strings.pages.targetAnalytics.eventsLoading)} />
      </div>
    );
  }
  if (events.isError) {
    return (
      <div data-testid={selectors.features.analytics.events.error}>
        <ErrorState
          title={t(strings.pages.targetAnalytics.eventsError)}
          message={events.error instanceof Error ? events.error.message : String(events.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void events.refetch();
          }}
        />
      </div>
    );
  }

  const rows = events.data.events;
  if (rows.length === 0) {
    return (
      <div data-testid={selectors.features.analytics.events.empty}>
        <EmptyState title={t(strings.pages.targetAnalytics.eventsEmpty)} />
      </div>
    );
  }

  const columns: ReadonlyArray<DataTableColumn<Event>> = [
    {
      key: "kind",
      header: t(strings.pages.targetAnalytics.columns.kind),
      cell: (row) => <span className="text-sm">{t(kindLabelKey(row.kind))}</span>,
    },
    {
      key: "domain",
      header: t(strings.pages.targetAnalytics.columns.domain),
      cell: (row) => <span className="text-sm">{row.domain || "—"}</span>,
    },
    {
      key: "conflict",
      header: t(strings.pages.targetAnalytics.columns.conflictId),
      cell: (row) => <span className="font-mono text-xs">{row.conflictId || "—"}</span>,
    },
    {
      key: "actor",
      header: t(strings.pages.targetAnalytics.columns.actor),
      cell: (row) => <span className="text-sm">{row.actor || "—"}</span>,
    },
  ];

  return (
    <div data-testid={selectors.features.analytics.events.root}>
      <DataTable
        rows={rows}
        getRowId={(e) => e.id}
        columns={columns}
        emptyMessage={t(strings.pages.targetAnalytics.eventsEmpty)}
      />
    </div>
  );
}
