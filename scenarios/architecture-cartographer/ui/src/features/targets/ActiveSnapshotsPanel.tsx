import { Link } from "react-router-dom";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { GraphSnapshot } from "@vrooli/proto-types/architecture-cartographer/v1/graph/graph_pb";

import { ApiError } from "../../api/client";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { useSharedLabels } from "../../hooks/useSharedLabels";
import { encodeScenarioPath } from "../../hooks/useScenarioPath";
import { useListSnapshots } from "./controllers/useTargetsController";

function describeExtractedAt(snapshot: GraphSnapshot): string {
  if (!snapshot.extractedAt) return "—";
  const date = timestampDate(snapshot.extractedAt);
  return formatDate(date, { dateStyle: "medium", timeStyle: "short" });
}

/**
 * Reads `ListGraphSnapshots` (across all scenarios) and renders the most
 * recent few as a deep-link list. Loading/empty/error states are all
 * explicit; no silent placeholders.
 */
export function ActiveSnapshotsPanel() {
  const { t } = useTranslation();
  const labels = useSharedLabels();
  const query = useListSnapshots({ pageSize: 8 });

  if (query.isLoading) {
    return (
      <div data-testid={selectors.features.targets.activeSnapshots.loading}>
        <LoadingState label={t(strings.pages.overview.loadingSnapshots)} />
      </div>
    );
  }

  if (query.isError) {
    const message =
      query.error instanceof ApiError
        ? query.error.message
        : t(strings.pages.overview.snapshotsError);
    return (
      <div data-testid={selectors.features.targets.activeSnapshots.error}>
        <ErrorState
          title={t(strings.pages.overview.snapshotsError)}
          message={message}
          retryLabel={labels.error.retry}
          onRetry={() => void query.refetch()}
        />
      </div>
    );
  }

  const snapshots = query.data?.snapshots ?? [];

  if (snapshots.length === 0) {
    return (
      <div data-testid={selectors.features.targets.activeSnapshots.empty}>
        <EmptyState title={t(strings.pages.overview.noActiveSnapshots)} />
      </div>
    );
  }

  return (
    <ul
      data-testid={selectors.features.targets.activeSnapshots.root}
      className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border bg-app-surface"
    >
      {snapshots.map((snapshot) => (
        <li
          key={snapshot.id}
          data-testid={selectors.features.targets.activeSnapshots.item({ id: snapshot.id })}
          className="flex items-center justify-between gap-3 p-4"
        >
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-app-foreground">
              {t(strings.targets.snapshots.scenarioLabel)} {snapshot.scenario}
            </p>
            <p className="mt-1 truncate text-xs text-app-muted-foreground">
              {t(strings.targets.snapshots.snapshotIdLabel)} {snapshot.id}
            </p>
            <p className="mt-1 truncate text-xs text-app-muted-foreground">
              {t(strings.targets.snapshots.extractedAtLabel)} {describeExtractedAt(snapshot)}
            </p>
          </div>
          <Button asChild size="sm" variant="outline">
            <Link to={`/targets/${encodeScenarioPath(snapshot.scenario)}`}>
              {t(strings.targets.snapshots.openButton)}
            </Link>
          </Button>
        </li>
      ))}
    </ul>
  );
}
