import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { DataTable } from "@vrooli/react-component-library/DataTable/1";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { HealthCard } from "../features/health/HealthCard";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fetchBoard } from "../api/offers";

type TimestampLike = { seconds: bigint | number; nanos?: number };
type RuntimeAvailability = { reason?: string };
type RuntimeGoalVerdict = { goal?: { id?: string; name?: string }; met?: boolean; explanation?: string };
type RuntimeEvaluation = { lastRunAt?: TimestampLike; nodesScored?: number; ageSeconds?: bigint | number; degraded?: boolean; reason?: string };
type RuntimeBoardData = { postureAgeSeconds?: bigint | number; goals?: RuntimeGoalVerdict[]; evaluation?: RuntimeEvaluation; postureSource?: string; defaultAliveGap?: string };

const enumLabel = (value: unknown) => {
  if (typeof value === "number") {
    return ["UNSPECIFIED", "IDEA", "CANDIDATE", "TRIGGER_MET", "ACTIVE", "SHIPPED", "RETIRED", "PROPOSED"][value] ?? "UNKNOWN";
  }
  return typeof value === "string" ? value.replace(/^STATUS_/, "") : "UNKNOWN";
};

const localizedRankReason = (reason: string | undefined, t: ReturnType<typeof useTranslation>["t"]) => {
  if (!reason) return t(strings.pages.dashboard.rankReasonMissing);
  if (reason === "status not set") return t(strings.pages.dashboard.rankReasonStatusNotSet);
  if (reason === "captured, not planned against") return t(strings.pages.dashboard.rankReasonIdea);
  if (reason === "blocked: trigger not met") return t(strings.pages.dashboard.rankReasonCandidate);
  if (reason === "trigger fired") return t(strings.pages.dashboard.rankReasonTriggerMet);
  if (reason === "awaiting operator decision") return t(strings.pages.dashboard.rankReasonProposed);
  if (reason === "active and earning nothing") return t(strings.pages.dashboard.rankReasonActiveEarningNothing);
  if (reason === "active and earning") return t(strings.pages.dashboard.rankReasonActiveEarning);
  if (reason === "shipped and earning nothing") return t(strings.pages.dashboard.rankReasonShippedEarningNothing);
  if (reason === "shipped and earning") return t(strings.pages.dashboard.rankReasonShippedEarning);
  if (reason === "retired") return t(strings.pages.dashboard.rankReasonRetired);
  const unknown = /^(active|shipped); earnings unknown — (.+) unavailable$/.exec(reason);
  if (unknown) return t(unknown[1] === "active" ? strings.pages.dashboard.rankReasonActiveUnknown : strings.pages.dashboard.rankReasonShippedUnknown, { source: unknown[2] });
  const status = /^unknown status: (.+)$/.exec(reason);
  return status ? t(strings.pages.dashboard.rankReasonUnknown, { status: status[1] }) : reason;
};

const timestampLabel = (timestamp?: TimestampLike) => timestamp ? new Date(Number(timestamp.seconds) * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000)).toISOString() : "";

const hasStatus = (value: unknown, expected: string) => enumLabel(value) === expected;
const entryAvailability = (entry: unknown): RuntimeAvailability[] => {
  const value = (entry as { availability?: unknown }).availability;
  return Array.isArray(value) ? value as RuntimeAvailability[] : [];
};

export function DashboardPage() {
  const { t } = useTranslation();
  const query = useQuery({ queryKey: ["offer-board"], queryFn: fetchBoard, retry: false });
  const boardData = query.data as unknown as RuntimeBoardData | undefined;
  const entries = useMemo(() => query.data?.entries ?? [], [query.data?.entries]);
  const availability = query.data?.availability ?? [];
  const position = query.data?.position;
  const showError = query.isError;
  const showEmpty = !query.isLoading && Boolean(query.data) && entries.length === 0;
  const state = useSurfaceState({
    query: { isLoading: query.isLoading, isFetching: query.isFetching, isError: query.isError, error: query.error },
    availability: { partial: availability.length > 0 },
    empty: showEmpty,
  });
  const firedTriggers = useMemo(() => entries.filter((entry) => hasStatus(entry.status, "TRIGGER_MET")), [entries]);
  const blockedOffers = useMemo(() => entries.filter((entry) => hasStatus(entry.status, "CANDIDATE")), [entries]);
  const earningNothing = useMemo(() => entries.filter((entry) => hasStatus(entry.status, "ACTIVE") && entry.actualsAvailable && entry.actualMinor === 0n), [entries]);
  const postureUnavailable = !position || availability.length > 0;
  const ledgerUnavailable = query.isError || availability.some((item) => item.source.includes("money-ledger"));
  const evaluation = boardData?.evaluation;
  const actualReason = (entry: typeof entries[number]) => {
    const first = entryAvailability(entry)[0];
    return first?.reason ?? t(strings.pages.dashboard.rankReasonMissing);
  };
  const rankingColumns = [
    { id: "offer", header: t(strings.pages.dashboard.boardOffer), accessor: (entry: typeof entries[number]) => entry.title, searchValue: (entry: typeof entries[number]) => entry.title },
    { id: "status", header: t(strings.pages.dashboard.boardStatus), accessor: (entry: typeof entries[number]) => enumLabel(entry.status), searchValue: (entry: typeof entries[number]) => enumLabel(entry.status) },
    { id: "reason", header: t(strings.pages.dashboard.rankReason), accessor: (entry: typeof entries[number]) => localizedRankReason(entry.rankReason, t), searchValue: (entry: typeof entries[number]) => entry.rankReason || "", className: "break-words" },
    { id: "actual", header: t(strings.pages.dashboard.boardActual), accessor: (entry: typeof entries[number]) => entry.actualsAvailable ? entry.actualMinor.toString() : t(strings.pages.dashboard.actualUnavailable, { reason: actualReason(entry) }), searchValue: (entry: typeof entries[number]) => entry.actualsAvailable ? entry.actualMinor.toString() : actualReason(entry), className: "break-words" },
  ];

  const renderGroup = (testId: string, title: string, group: typeof entries) => (
    <section data-testid={testId} aria-label={title} className="rounded-md border p-4 min-w-0">
      <h3 className="font-semibold" aria-label={`${title}: ${t(strings.pages.dashboard.noCurrentRecords)}`}>{title}</h3>
      {/* Empty groups remain visible so their contract bindings are present in the accessibility tree. */}
      {group.length ? <ul>{group.map((entry) => <li key={entry.nodeId} className="text-sm">{entry.title}: {localizedRankReason(entry.rankReason, t)}</li>)}</ul> : <p role="status">{t(strings.pages.dashboard.noCurrentRecords)}</p>}
    </section>
  );

  return (
    <ExperienceSurface surfaceId="board" state={state.state} statusMessage={state.reason} data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-4">
      <h2 id="dashboard-heading" className="text-2xl font-semibold">{t(strings.pages.dashboard.title)}</h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <section data-testid={selectors.pages.sourceAvailability} aria-label={t(strings.pages.dashboard.sourceUnavailableReason)} className="rounded-md border p-3">
        <h3 className="font-semibold" aria-label={`${t(strings.pages.dashboard.sourceAvailability)}: ${t(strings.pages.dashboard.unavailableSourcesCount, { count: availability.length })}`}>{t(strings.pages.dashboard.sourceAvailability)}</h3>
        {availability.length ? <ul>{availability.map((item) => <li data-testid={`${selectors.pages.sourceAvailability}-${item.source}`} key={item.source}>{t(strings.pages.dashboard.sourceUnavailable, item)}{item.lastSuccessAt ? ` (${timestampLabel(item.lastSuccessAt)})` : ""}</li>)}</ul> : <p role="status">{t(strings.pages.dashboard.unavailableSourcesCount, { count: 0 })}</p>}
      </section>
      {renderGroup(selectors.pages.firedTriggers, t(strings.pages.dashboard.firedTriggers), firedTriggers)}
      <div className="grid gap-4 lg:grid-cols-3">
        <HealthCard />
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.priorityBoard)}</CardTitle></CardHeader>
          <CardContent>
            <div data-testid={selectors.pages.boardRanking}>
            <DataTable
              rows={entries}
              columns={rankingColumns}
              getRowKey={(entry, index) => entry.nodeId || `${entry.title}-${index}`}
              caption={t(strings.pages.dashboard.offerRecords, { count: entries.length })}
              searchLabel={t(strings.pages.dashboard.priorityBoard)}
              searchPlaceholder={t(strings.pages.dashboard.boardOffer)}
              emptyMessage={t(strings.pages.dashboard.noCurrentRecords)}
              tableTestId="board-ranking-table"
            />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.ledgerPosture)}</CardTitle></CardHeader>
          <CardContent>
            <p data-testid={selectors.pages.postureSummary} role="status" aria-label={t(strings.pages.dashboard.ledgerPosture)} className="font-medium">{position ? t(strings.pages.dashboard.runwayMonths, { months: position.runwayMonths.toFixed(2) }) : t(strings.pages.dashboard.postureUnavailable)}</p>
            <p data-testid={selectors.pages.postureBasis} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.postureSource)}: {boardData?.postureSource || t(strings.pages.dashboard.postureUnavailable)}</p>
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.postureAge)}: {boardData?.postureAgeSeconds?.toString() ?? t(strings.pages.dashboard.postureUnavailable)}</p>
            <ul data-testid={selectors.pages.postureGoalVerdicts} aria-label={t(strings.pages.dashboard.goalVerdict)} className="text-sm">{boardData?.goals?.length ? boardData.goals.map((goalVerdict) => <li key={goalVerdict.goal?.id}>{goalVerdict.goal?.name || goalVerdict.goal?.id || t(strings.pages.dashboard.postureUnavailable)}: {goalVerdict.met ? t(strings.pages.dashboard.goalMet) : t(strings.pages.dashboard.goalNotMet)}{goalVerdict.explanation ? ` — ${goalVerdict.explanation}` : ""}</li>) : <li>{t(strings.pages.dashboard.postureUnavailable)}</li>}</ul>
          </CardContent>
        </Card>
      </div>
      {renderGroup(selectors.pages.blockedOffers, t(strings.pages.dashboard.blockedOffers), blockedOffers)}
      {renderGroup(selectors.pages.earningNothing, t(strings.pages.dashboard.earningNothing), earningNothing)}
      <section data-testid={selectors.pages.evaluationCondition} aria-label={t(strings.pages.dashboard.evaluationCondition)} className="rounded-md border p-4">
        <h3 className="font-semibold">{t(strings.pages.dashboard.evaluationCondition)}</h3>
        {evaluation ? <dl className="grid gap-1 text-sm"><div><dt className="inline font-medium">{t(strings.pages.dashboard.evaluationLastRun)}: </dt><dd className="inline">{timestampLabel(evaluation.lastRunAt) || t(strings.pages.dashboard.evaluationNotRun)}</dd></div><div><dt className="inline font-medium">{t(strings.pages.dashboard.evaluationNodesScored)}: </dt><dd className="inline">{evaluation.nodesScored ?? 0}</dd></div><div><dt className="inline font-medium">{t(strings.pages.dashboard.evaluationAge)}: </dt><dd className="inline">{evaluation.ageSeconds?.toString() ?? t(strings.pages.dashboard.evaluationNotRun)}</dd></div><div role={evaluation.degraded ? "alert" : undefined}>{evaluation.degraded ? t(strings.pages.dashboard.evaluationDegraded, { reason: evaluation.reason || t(strings.pages.dashboard.evaluationNotRun) }) : evaluation.reason || t(strings.pages.dashboard.evaluationNotRun)}</div></dl> : <p>{t(strings.pages.dashboard.evaluationNotRun)}</p>}
      </section>
      <p data-testid={selectors.pages.defaultAliveGap} role="status" className={boardData?.defaultAliveGap ? "rounded-md border p-3" : "sr-only"}>{t(strings.pages.dashboard.defaultAliveGap)}: {boardData?.defaultAliveGap}</p>
      <p data-testid={selectors.pages.emptyGuidance} role="note" className={showEmpty ? "rounded-md border border-dashed p-4 text-app-muted-foreground" : "sr-only"}>{t(strings.pages.dashboard.emptyGuidance)}</p>
      <p data-testid="board-ledger-gap" role="status" className={ledgerUnavailable ? undefined : "sr-only"}>{t(strings.pages.dashboard.boardUnavailable, { message: t(strings.pages.dashboard.postureUnavailable) })}</p>
      <p data-testid={selectors.pages.postureGap} role="status" className={showError || postureUnavailable ? undefined : "sr-only"}>{t(strings.pages.dashboard.boardUnavailable, { message: query.error instanceof Error ? query.error.message : t(strings.pages.dashboard.postureUnavailable) })}</p>
    </ExperienceSurface>
  );
}
