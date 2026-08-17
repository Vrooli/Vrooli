import { useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { FormSection } from "../components/FormSection";
import { DirtyStateGuard } from "../components/DirtyStateGuard";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Select } from "../components/ui/select";
import { HealthCard } from "../features/health/HealthCard";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { configuredBookId, declareGoal, fetchBooks, fetchGoals, fetchPosition, fetchPostings } from "../api/ledger";
import { formatCurrency, formatDate } from "../i18n/format";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { SustainPeriodUnit } from "@vrooli/proto-types/money-ledger/v1/ledger/ledger_pb";
import { Basis } from "@vrooli/proto-types/money-ledger/v1/shared/ledger_types_pb";
import { CartesianCharts } from "../components/CartesianCharts";
import type { ChartDatum } from "../components/Chart";

interface PositionView {
  cashMinor: bigint;
  runwayMonths: number;
  runwayAvailable?: boolean;
  partial: boolean;
  availability: Array<{ adapterId: string; reason: string; lastSuccessAt?: { seconds: bigint | number; nanos?: number } }>;
}

const unitName = (unit: SustainPeriodUnit) => SustainPeriodUnit[unit];
const age = (timestamp?: { seconds: bigint | number; nanos?: number }) => {
  if (!timestamp) return "unknown";
  return formatDate(new Date(Number(timestamp.seconds) * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000)));
};

type PostingLike = {
  event?: {
    amountMinor: bigint;
    occurredAt?: { seconds: bigint | number; nanos?: number };
    basis: Basis;
  };
};

type TrendRow = { id: string; period: string; inflow: bigint; outflow: bigint; net: bigint };

const dayKey = (timestamp?: { seconds: bigint | number; nanos?: number }) => {
  if (!timestamp) return "";
  return new Date(Number(timestamp.seconds) * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000)).toISOString().slice(0, 10);
};

const basisLabel = (basis: Basis, t: ReturnType<typeof useTranslation>["t"]) => {
  if (basis === Basis.AUTHORITATIVE) return t(strings.pages.dashboard.trendBasisAuthoritative);
  if (basis === Basis.DERIVED) return t(strings.pages.dashboard.trendBasisDerived);
  if (basis === Basis.OPERATOR_ASSERTED) return t(strings.pages.dashboard.trendBasisOperator);
  return t(strings.pages.dashboard.trendBasisMixed);
};

export function DashboardPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [trendWindowDays, setTrendWindowDays] = useState(30);
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const selectedBook = books.data?.books.find((book) => book.id === bookId);
  const query = useQuery({ queryKey: ["position", bookId], queryFn: () => fetchPosition(bookId), retry: false, enabled: Boolean(bookId) });
  const postings = useQuery({ queryKey: ["postings", bookId], queryFn: () => fetchPostings(bookId), retry: false, enabled: Boolean(bookId) });
  const goals = useQuery({ queryKey: ["goals", bookId], queryFn: () => fetchGoals(bookId), retry: false, enabled: Boolean(bookId) });
  const previous = useQuery({ queryKey: ["position-previous", bookId], queryFn: () => fetchPosition(bookId, "previous", "current"), retry: false, enabled: Boolean(bookId) });
  const data = query.data as PositionView | null | undefined;
  const previousData = previous.data as PositionView | null | undefined;
  const trendRows = useMemo<TrendRow[]>(() => {
    const cutoff = Date.now() - trendWindowDays * 24 * 60 * 60 * 1000;
    const byDay = new Map<string, TrendRow>();
    for (const posting of (postings.data?.postings ?? []) as PostingLike[]) {
      const event = posting.event;
      const date = dayKey(event?.occurredAt);
      if (!event || !date || Date.parse(`${date}T00:00:00.000Z`) < cutoff) continue;
      const row = byDay.get(date) ?? { id: date, period: date, inflow: 0n, outflow: 0n, net: 0n };
      const amount = event.amountMinor;
      if (amount >= 0n) row.inflow += amount;
      else row.outflow += -amount;
      row.net += amount;
      byDay.set(date, row);
    }
    return [...byDay.values()].sort((left, right) => left.period.localeCompare(right.period));
  }, [postings.data?.postings, trendWindowDays]);
  const trendStatus = postings.isError ? "partial-error" : postings.isFetching ? "refreshing" : trendRows.length ? (data?.partial ? "stale" : "success") : "empty";
  const trendBasis = useMemo(() => {
    const bases = new Set(((postings.data?.postings ?? []) as PostingLike[]).map((posting) => posting.event?.basis).filter((basis): basis is Basis => basis !== undefined));
    const firstBasis = [...bases][0];
    return firstBasis !== undefined ? basisLabel(firstBasis, t) : t(strings.pages.dashboard.trendBasisMixed);
  }, [postings.data?.postings, t]);
  const currency = selectedBook?.currency || "USD";
  const chartData = (value: (row: TrendRow) => bigint): ChartDatum[] => trendRows.map((row) => ({ id: row.id, label: row.period.slice(5), value: Number(value(row)) / 100, detail: currency }));
  const showError = query.isError;
  const showEmpty = !query.isLoading && !data && !query.isError;
  const surface = useSurfaceState({
    query: { isLoading: query.isLoading || books.isLoading || goals.isLoading, isFetching: query.isFetching || books.isFetching || goals.isFetching, isError: query.isError || books.isError, error: query.error || books.error },
    availability: { partial: data?.partial },
    empty: showEmpty,
  });
  const positionStatus = showError
    ? t(strings.pages.dashboard.positionUnavailable, { message: query.error instanceof Error ? query.error.message : t(strings.pages.dashboard.sourceUnavailable) })
    : data?.partial
      ? t(strings.pages.dashboard.partial)
      : data
        ? t(strings.pages.dashboard.complete)
        : t(strings.pages.dashboard.notConfigured);
  const runwayText = data?.runwayAvailable === false || (data && !data.runwayAvailable)
    ? t(strings.pages.dashboard.runwayUndefined)
    : data
      ? t(strings.pages.dashboard.runwayMonths, { months: data.runwayMonths.toFixed(2) })
      : t(strings.pages.dashboard.runwayUndefined);
  const positionDelta = data && previousData
    ? formatCurrency(Number(data.cashMinor - previousData.cashMinor) / 100, selectedBook?.currency || "USD")
    : t(strings.pages.dashboard.changeUnavailable);
  const [goalForm, setGoalForm] = useState({ name: "", metric: "revenue", comparator: ">=", thresholdMinor: "", sustainPeriods: "1", periodUnit: String(SustainPeriodUnit.MONTH) });
  const [goalMessage, setGoalMessage] = useState("");
  const [goalError, setGoalError] = useState(false);
  const goalMutation = useMutation({
    mutationFn: (input: { bookId: string; name: string; metric: string; comparator: string; thresholdMinor: bigint; sustainPeriods: number; periodUnit: SustainPeriodUnit }) => declareGoal(input),
    onSuccess: async () => {
      await Promise.all([queryClient.invalidateQueries({ queryKey: ["goals"] }), queryClient.invalidateQueries({ queryKey: ["position"] })]);
      setGoalForm({ name: "", metric: "revenue", comparator: ">=", thresholdMinor: "", sustainPeriods: "1", periodUnit: String(SustainPeriodUnit.MONTH) });
      setGoalError(false);
      setGoalMessage(t(strings.pages.dashboard.savedNotice));
    },
    onError: () => setGoalError(true),
  });
  const submitGoal = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setGoalMessage("");
    let threshold: bigint;
    try { threshold = BigInt(goalForm.thresholdMinor); } catch { threshold = 0n; }
    const periods = Number(goalForm.sustainPeriods);
    if (!bookId || !goalForm.name.trim() || !goalForm.metric.trim() || !goalForm.comparator || threshold < 0n || !Number.isInteger(periods) || periods < 1) {
      setGoalError(true);
      return;
    }
    setGoalError(false);
    goalMutation.mutate({ bookId, name: goalForm.name.trim(), metric: goalForm.metric.trim(), comparator: goalForm.comparator, thresholdMinor: threshold, sustainPeriods: periods, periodUnit: Number(goalForm.periodUnit) });
  };
  const goalRows = useMemo(() => goals.data?.goals ?? [], [goals.data?.goals]);

  return (
    <ExperienceSurface surfaceId="dashboard" state={surface.state} data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-4" statusMessage={showError ? t(strings.pages.dashboard.positionUnavailable, { message: t(strings.pages.dashboard.sourceUnavailable) }) : surface.reason}>
      <h2 id="dashboard-heading" className="text-2xl font-semibold">{t(strings.pages.dashboard.title)}</h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.runwayLabel)}</CardTitle></CardHeader>
          <CardContent>
            <p data-testid={selectors.pages.runwayFigure} role="status" className="text-2xl font-semibold tabular-nums">{runwayText}</p>
            <p data-testid={selectors.pages.runwayBasis} role="note" aria-label={t(strings.pages.dashboard.runwayBasis)} className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.runwayBasis)}</p>
            {data?.availability.map((item) => <p data-testid={selectors.pages.missingAdapter} role="note" aria-label={item.adapterId} key={item.adapterId} className="text-sm text-amber-700">{t(strings.pages.dashboard.missingAdapter, { adapterId: item.adapterId, reason: `${item.reason} · ${t(strings.pages.dashboard.availabilityAge, { age: age(item.lastSuccessAt) })}` })}</p>)}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.completenessLabel)}</CardTitle></CardHeader>
          <CardContent>
            <p data-testid={selectors.pages.completeness} role="status" aria-label={t(strings.pages.dashboard.completenessLabel)} className="font-medium">{positionStatus}</p>
            {data?.availability.map((item) => <p data-testid={selectors.pages.missingAdapter} role="note" aria-label={item.adapterId} key={`complete-${item.adapterId}`} className="text-sm text-amber-700">{t(strings.pages.dashboard.missingAdapter, item)}</p>)}
          </CardContent>
        </Card>
        <HealthCard />
      </div>
      <p data-testid={selectors.pages.positionError} role={showError ? "alert" : "status"} aria-live={showError ? "assertive" : "off"} className={showError ? undefined : "sr-only"}>{positionStatus}</p>
      <p data-testid={selectors.pages.emptyGuidance} role="note" className="rounded-md border border-dashed p-4 text-app-muted-foreground">{t(strings.pages.dashboard.emptyGuidance)}</p>
      <Card>
        <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div><CardTitle>{t(strings.pages.dashboard.trendTitle)}</CardTitle><p className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.trendDescription)}</p></div>
          <label className="grid min-w-36 gap-1" htmlFor="trend-window"><span>{t(strings.pages.dashboard.trendWindowLabel)}</span><Select id="trend-window" data-testid={selectors.pages.trendWindow} value={String(trendWindowDays)} onChange={(event) => setTrendWindowDays(Number(event.target.value))} options={[7, 30, 90].map((days) => ({ value: String(days), label: t(strings.pages.dashboard.trendWindowDays, { days }) }))} /></label>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div data-testid={selectors.pages.runwayBurnTrend} aria-label={t(strings.pages.dashboard.trendTitle)} className="grid gap-4 xl:grid-cols-3">
            <CartesianCharts data={chartData((row) => row.inflow)} title={t(strings.pages.dashboard.trendInflow)} kind="area" status={trendStatus} emptyMessage={t(strings.pages.dashboard.trendEmpty)} valueFormatter={(value) => formatCurrency(value, currency)} />
            <CartesianCharts data={chartData((row) => row.outflow)} title={t(strings.pages.dashboard.trendOutflow)} kind="area" status={trendStatus} emptyMessage={t(strings.pages.dashboard.trendEmpty)} valueFormatter={(value) => formatCurrency(value, currency)} />
            <CartesianCharts data={chartData((row) => row.net)} title={t(strings.pages.dashboard.trendNet)} kind="line" status={trendStatus} emptyMessage={t(strings.pages.dashboard.trendEmpty)} valueFormatter={(value) => formatCurrency(value, currency)} />
          </div>
          <p data-testid={selectors.pages.trendSourceBasis} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.trendSourceBasis, { basis: trendBasis })}</p>
          {postings.isError && <p data-testid={selectors.pages.trendGap} role="alert" className="rounded-md border border-dashed p-3 text-sm text-amber-800">{t(strings.pages.dashboard.trendGap, { reason: postings.error instanceof Error ? postings.error.message : t(strings.pages.dashboard.sourceUnavailable) })}</p>}
          {data?.partial && !postings.isError && <p data-testid={selectors.pages.trendGap} role="note" aria-label={t(strings.pages.dashboard.trendGapLabel)} className="rounded-md border border-dashed p-3 text-sm text-amber-800">{t(strings.pages.dashboard.trendGap, { reason: t(strings.pages.dashboard.trendStale, { age: age(data.availability[0]?.lastSuccessAt) }) })}</p>}
          {!postings.isError && trendRows.length === 0 && !postings.isLoading && <p data-testid={selectors.pages.trendGap} role="note" aria-label={t(strings.pages.dashboard.trendGapLabel)} className="rounded-md border border-dashed p-3 text-sm text-app-muted-foreground">{t(strings.pages.dashboard.trendEmpty)} {t(strings.pages.dashboard.trendNoPostings)}</p>}
          <div className="overflow-x-auto">
            <table data-testid={selectors.pages.runwayBurnTable} className="w-full min-w-[32rem] border-collapse text-sm" aria-label={t(strings.pages.dashboard.trendTableCaption)}>
              <caption className="sr-only">{t(strings.pages.dashboard.trendTableCaption)}</caption>
              <thead><tr className="border-b text-left"><th className="p-2">{t(strings.pages.dashboard.trendPeriod)}</th><th className="p-2">{t(strings.pages.dashboard.trendInflow)}</th><th className="p-2">{t(strings.pages.dashboard.trendOutflow)}</th><th className="p-2">{t(strings.pages.dashboard.trendNet)}</th></tr></thead>
              <tbody>{postings.isError ? <tr><td className="p-2" colSpan={4}>{t(strings.pages.dashboard.trendUnavailable)}</td></tr> : trendRows.length ? trendRows.map((row) => <tr key={row.id} className="border-b"><th scope="row" className="p-2 text-left font-normal">{row.period}</th><td className="p-2 tabular-nums">{formatCurrency(Number(row.inflow) / 100, currency)}</td><td className="p-2 tabular-nums">{formatCurrency(Number(row.outflow) / 100, currency)}</td><td className="p-2 tabular-nums">{formatCurrency(Number(row.net) / 100, currency)}</td></tr>) : !postings.isLoading ? <tr><td className="p-2" colSpan={4}>{t(strings.pages.dashboard.trendNoPostings)}</td></tr> : null}</tbody>
            </table>
          </div>
        </CardContent>
      </Card>
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>{t(strings.pages.dashboard.goalsTitle)}</CardTitle></CardHeader>
          <CardContent>
            <ul data-testid={selectors.pages.goalList} aria-label={t(strings.pages.dashboard.goalsTitle)} className="grid gap-3">
              {goalRows.length === 0 && <li>{t(strings.pages.dashboard.goalsDescription)}</li>}
              {goalRows.map((verdict) => verdict.goal && <li key={verdict.goal.id} className="rounded-md border p-3">
                <div className="flex items-start justify-between gap-3"><span className="font-medium" aria-label={t(strings.pages.dashboard.goalName, { name: verdict.goal.name })}>{verdict.goal.name}</span><span className="font-medium">{verdict.met ? t(strings.pages.dashboard.goalVerdictMet) : t(strings.pages.dashboard.goalVerdictUnmet)}</span></div>
                <p className="text-sm tabular-nums">{t(strings.pages.dashboard.goalThreshold, { value: `${formatCurrency(Number(verdict.goal.thresholdMinor) / 100, selectedBook?.currency || "USD")} · ${verdict.goal.metric} ${verdict.goal.comparator}` })}</p>
                <p className="text-sm">{t(strings.pages.dashboard.goalProgress, { current: verdict.sustainedPeriods, required: verdict.requiredPeriods || verdict.goal.sustainPeriods, unit: unitName(verdict.periodUnit || verdict.goal.sustainPeriodUnit) })}</p>
                <p className="text-sm">{t(strings.pages.dashboard.goalPeriod, { unit: unitName(verdict.periodUnit || verdict.goal.sustainPeriodUnit) })}</p>
                <meter data-testid={selectors.pages.sustainProgress} min={0} max={verdict.requiredPeriods || verdict.goal.sustainPeriods} value={verdict.sustainedPeriods} className="mt-2 block w-full" aria-label={verdict.goal.name} />
                <p className="text-sm text-app-muted-foreground">{verdict.explanation}</p>
              </li>)}
            </ul>
            <DirtyStateGuard isDirty={Boolean(goalForm.name || goalForm.thresholdMinor)} protectUnload title={t(strings.pages.dashboard.declareGoalTitle)} description={t(strings.pages.dashboard.goalsDescription)}>
              <FormSection title={t(strings.pages.dashboard.declareGoalTitle)}>
                <form className="mt-3 grid gap-3" onSubmit={submitGoal}>
                  <label className="grid gap-1" htmlFor="goal-name"><span>{t(strings.pages.dashboard.goalNameLabel)}</span><Input id="goal-name" value={goalForm.name} onChange={(event) => setGoalForm({ ...goalForm, name: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="goal-metric"><span>{t(strings.pages.dashboard.goalMetricLabel)}</span><Input id="goal-metric" value={goalForm.metric} onChange={(event) => setGoalForm({ ...goalForm, metric: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="goal-comparator"><span>{t(strings.pages.dashboard.goalComparatorLabel)}</span><Select id="goal-comparator" value={goalForm.comparator} onChange={(event) => setGoalForm({ ...goalForm, comparator: event.target.value })} options={[{ value: ">=", label: ">=" }, { value: "<=", label: "<=" }, { value: ">", label: ">" }]} /></label>
                  <label className="grid gap-1" htmlFor="goal-threshold"><span>{t(strings.pages.dashboard.goalThresholdLabel)}</span><Input id="goal-threshold" type="number" min="0" step="1" value={goalForm.thresholdMinor} onChange={(event) => setGoalForm({ ...goalForm, thresholdMinor: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="goal-sustain-periods"><span>{t(strings.pages.dashboard.goalSustainPeriodsLabel)}</span><Input id="goal-sustain-periods" type="number" min="1" step="1" value={goalForm.sustainPeriods} onChange={(event) => setGoalForm({ ...goalForm, sustainPeriods: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="goal-period-unit"><span>{t(strings.pages.dashboard.goalPeriodUnitLabel)}</span><Select id="goal-period-unit" value={goalForm.periodUnit} onChange={(event) => setGoalForm({ ...goalForm, periodUnit: event.target.value })} options={[{ value: String(SustainPeriodUnit.DAY), label: "DAY" }, { value: String(SustainPeriodUnit.WEEK), label: "WEEK" }, { value: String(SustainPeriodUnit.MONTH), label: "MONTH" }]} /></label>
                  {goalError && <p role="alert" className="text-sm text-app-danger">{goalMutation.isError ? t(strings.pages.dashboard.requestError) : t(strings.pages.dashboard.validationError)}</p>}
                  {goalMessage && <p role="status" className="text-sm text-app-success">{goalMessage}</p>}
                  <Button type="submit" disabled={goalMutation.isPending || !bookId}>{t(strings.pages.dashboard.goalCreateAction)}</Button>
                </form>
              </FormSection>
            </DirtyStateGuard>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>{t(strings.pages.dashboard.changeTitle)}</CardTitle></CardHeader>
          <CardContent><p data-testid={selectors.pages.positionDelta} className="tabular-nums">{positionDelta}</p><p className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.changeDescription)}</p></CardContent>
        </Card>
      </div>
    </ExperienceSurface>
  );
}
