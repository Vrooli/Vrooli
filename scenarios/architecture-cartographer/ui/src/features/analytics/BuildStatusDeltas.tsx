import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../components/EmptyState";
import { useEvents } from "./controllers/useAnalyticsController";
import { EventKind } from "@vrooli/proto-types/architecture-cartographer/v1/analytics/analytics_pb";

/**
 * Derives a per-apply build status delta sequence from the events log.
 * Pairs of (APPLY_RAN, APPLY_BUILD_GREEN | APPLY_BUILD_RED | APPLY_REVERTED)
 * are projected onto a green/red/reverted timeline.
 */
export interface BuildStatusDeltasProps {
  scenario: string;
}

export function BuildStatusDeltas({ scenario }: BuildStatusDeltasProps) {
  const { t } = useTranslation();
  const events = useEvents(scenario);

  if (events.isPending || events.isError) return null;

  const buildEvents = events.data.events.filter(
    (e) =>
      e.kind === EventKind.APPLY_BUILD_GREEN ||
      e.kind === EventKind.APPLY_BUILD_RED ||
      e.kind === EventKind.APPLY_REVERTED,
  );

  if (buildEvents.length === 0) {
    return (
      <div data-testid={selectors.features.analytics.buildDeltas.empty}>
        <EmptyState title={t(strings.pages.targetAnalytics.buildDeltasEmpty)} />
      </div>
    );
  }

  return (
    <ul
      data-testid={selectors.features.analytics.buildDeltas.root}
      className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-3 text-sm"
    >
      {buildEvents.map((e) => {
        const isGreen = e.kind === EventKind.APPLY_BUILD_GREEN;
        const isRed = e.kind === EventKind.APPLY_BUILD_RED;
        return (
          <li key={e.id} className="flex items-center gap-2">
            <span
              aria-hidden="true"
              className={
                isGreen
                  ? "inline-block size-2 rounded-full bg-app-success"
                  : isRed
                    ? "inline-block size-2 rounded-full bg-app-danger"
                    : "inline-block size-2 rounded-full bg-app-muted-foreground"
              }
            />
            <span className="font-mono text-xs text-app-muted-foreground">{e.runId || e.id}</span>
            <span className="text-sm">
              {isGreen ? "green" : isRed ? "red" : "reverted"}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
