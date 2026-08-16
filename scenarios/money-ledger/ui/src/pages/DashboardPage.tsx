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
import { configuredBookId, declareGoal, fetchBooks, fetchGoals, fetchPosition } from "../api/ledger";
import { formatCurrency, formatDate } from "../i18n/format";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { SustainPeriodUnit } from "@vrooli/proto-types/money-ledger/v1/ledger/ledger_pb";

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

export function DashboardPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const selectedBook = books.data?.books.find((book) => book.id === bookId);
  const query = useQuery({ queryKey: ["position", bookId], queryFn: () => fetchPosition(bookId), retry: false, enabled: Boolean(bookId) });
  const goals = useQuery({ queryKey: ["goals", bookId], queryFn: () => fetchGoals(bookId), retry: false, enabled: Boolean(bookId) });
  const previous = useQuery({ queryKey: ["position-previous", bookId], queryFn: () => fetchPosition(bookId, "previous", "current"), retry: false, enabled: Boolean(bookId) });
  const data = query.data as PositionView | null | undefined;
  const previousData = previous.data as PositionView | null | undefined;
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
