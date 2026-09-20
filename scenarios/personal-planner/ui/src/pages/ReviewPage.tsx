import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { BookOpenCheck } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { DataTable } from "@vrooli/react-component-library/DataTable/1.4.3";
import { FormField } from "@vrooli/react-component-library/FormField/1";

import { fetchDailyReview, fetchReflection, fetchWeeklyReview, saveReflection } from "../api/review";
import { carryForwardAllocation, fetchAllocations, fetchTodayAllocations, type Allocation } from "../api/calendar";
import { selectors } from "../consts/selectors";
import { EvidenceInsight } from "../components/EvidenceInsight";

export function ReviewPage() {
  const queryClient = useQueryClient();
  const [localDate, setLocalDate] = useState(() => formatLocalDate(new Date()));
  const [view, setView] = useState<"day" | "week">("day");
  const [carryTargetDate, setCarryTargetDate] = useState(() => addDays(localDate, 1));
  const [reflectionText, setReflectionText] = useState("");
  const weekStart = mondayLocalDate(localDate);
  const daily = useQuery({ enabled: view === "day", queryKey: ["review", "daily", localDate], queryFn: () => fetchDailyReview(localDate) });
  const weekly = useQuery({ enabled: view === "week", queryKey: ["review", "weekly", weekStart], queryFn: () => fetchWeeklyReview(weekStart) });
  const reflection = useQuery({ enabled: view === "day", queryKey: ["review", "reflection", localDate], queryFn: () => fetchReflection(localDate) });
  const carryCandidates = useQuery({ enabled: view === "day", queryKey: ["review", "carry-candidates", localDate], queryFn: () => fetchAllocations(localDate, localDate) });
  const carryTarget = useQuery({ enabled: view === "day" && Boolean(carryTargetDate), queryKey: ["review", "carry-target", carryTargetDate], queryFn: () => fetchTodayAllocations(carryTargetDate) });
  const [carryMessage, setCarryMessage] = useState("");
  const carryMutation = useMutation({
    mutationFn: (allocation: Allocation) => carryForwardAllocation({ allocationId: allocation.id, targetLocalDate: carryTargetDate, startMinutes: Number(allocation.startMinutes) }),
    onSuccess: async () => {
      setCarryMessage("Carried forward once; the original placement remains visible in history.");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["review", "daily", localDate] }),
        queryClient.invalidateQueries({ queryKey: ["review", "carry-candidates", localDate] }),
        queryClient.invalidateQueries({ queryKey: ["review", "carry-target", carryTargetDate] }),
        queryClient.invalidateQueries({ queryKey: ["today-allocations"] }),
        queryClient.invalidateQueries({ queryKey: ["calendar-range"] }),
      ]);
    },
    onError: () => setCarryMessage("That placement could not move. The original schedule is unchanged."),
  });
  const reflectionMutation = useMutation({
    mutationFn: () => saveReflection({ localDate, text: reflectionText }),
    onSuccess: async (saved) => { setReflectionText(saved.text); await queryClient.invalidateQueries({ queryKey: ["review", "reflection", localDate] }); },
  });
  useEffect(() => { if (reflection.data) setReflectionText(reflection.data.text); }, [reflection.data]);
  const review = view === "day" ? daily : weekly;
  const shiftDate = (days: number) => {
    const next = new Date(`${localDate}T12:00:00`);
    next.setDate(next.getDate() + days);
    const nextLocalDate = formatLocalDate(next);
    setLocalDate(nextLocalDate);
    setCarryTargetDate(addDays(nextLocalDate, 1));
  };
  return <section className="planner-surface review-surface" data-testid={selectors.pages.review} aria-labelledby="review-heading">
    <PageHeader className="planner-page-header" headingId="review-heading" eyebrow="A truthful look back" title="Review" description="Notice what happened without turning missing data into a story." leading={<span className="planner-page-mark" aria-hidden="true"><BookOpenCheck size={21} /></span>} />
    <div className="review-view-switcher" role="tablist" aria-label="Review period">
      <Button type="button" variant={view === "day" ? "primary" : "secondary"} role="tab" aria-selected={view === "day"} onClick={() => setView("day")}>Day</Button>
      <Button type="button" variant={view === "week" ? "primary" : "secondary"} role="tab" aria-selected={view === "week"} onClick={() => setView("week")}>Week</Button>
    </div>
    <div className="review-date-controls" aria-label={view === "day" ? "Review date" : "Review week"}>
      <Button type="button" className="quiet-action" variant="secondary" onClick={() => shiftDate(view === "day" ? -1 : -7)} aria-label={view === "day" ? "Previous day" : "Previous week"}>←</Button>
      <FormField required label={view === "day" ? "Day" : "Week starting"} control={<Input required aria-label={view === "day" ? "Day" : "Week starting"} type="date" value={view === "day" ? localDate : weekStart} onChange={(event) => { setLocalDate(event.target.value); setCarryTargetDate(addDays(event.target.value, 1)); }} />} />
      <Button type="button" className="quiet-action" variant="secondary" onClick={() => shiftDate(view === "day" ? 1 : 7)} aria-label={view === "day" ? "Next day" : "Next week"}>→</Button>
    </div>
    <div className="review-compass"><div><span className="card-kicker">{view === "day" ? "ONE DAY, CLEARLY" : "A WEEK IN VIEW"}</span><h2>{view === "day" ? "Notice what the day is telling you." : "Look for a pattern, not a verdict."}</h2><p>{view === "day" ? "Measured activity, accepted plans, and one small observation are enough. Missing data stays visibly missing." : "Compare planned and recorded time across the week without turning the difference into a diagnosis."}</p></div><span className="review-compass-mark" aria-hidden="true">{view === "day" ? "01" : "07"}</span></div>
    {review.isLoading && <p role="status">Preparing this review…</p>}
    {review.isError && <p role="alert">Review data is unavailable right now.</p>}
    {view === "day" && daily.data && <>
      <div className="review-summary-grid"><article className="review-summary-recorded"><span className="card-kicker">RECORDED ACTIVE TIME</span><strong>{Number(daily.data.recordedActiveMinutes)} min</strong><p>Focus sessions and manual actuals.</p></article><article className="review-summary-sessions"><span className="card-kicker">FOCUS SESSIONS</span><strong>{Number(daily.data.focusSessionCount)}</strong><p>Sessions with a recorded start.</p></article><article className="review-summary-goals"><span className="card-kicker">ACTIVE GOALS</span><strong>{Number(daily.data.activeGoalCount)}</strong><p>Outcome direction, not achievement.</p></article></div>
      <div className="review-honesty-card"><div className="review-honesty-heading"><span className="card-kicker">COVERAGE</span><span className="review-evidence-dot" aria-hidden="true" /></div><p>{daily.data.coverageNote}</p><p className="review-unknown">Planned: {Number(daily.data.plannedMinutes)} min · Unrecorded: unknown</p></div>
      <EvidenceInsight planned={Number(daily.data.plannedMinutes)} recorded={Number(daily.data.recordedActiveMinutes)} />
      <section className="review-carry-card" aria-labelledby="review-carry-heading">
        <div className="review-carry-heading"><div><span className="card-kicker">SELECTIVE CARRY-FORWARD</span><h2 id="review-carry-heading">Choose what still deserves tomorrow</h2><p>Nothing is assumed unfinished. Select an accepted placement only when it remains a promise you want to keep.</p></div><FormField required label="Carry to" control={<Input required aria-label="Carry to" type="date" value={carryTargetDate} onChange={(event) => setCarryTargetDate(event.target.value)} />} /></div>
        {carryTarget.data && <p className="review-carry-preview">Target capacity preview: {Number(carryTarget.data.availableMinutes)} min available · {Number(carryTarget.data.breathingRoomMinutes)} min breathing room before this move.</p>}
        {carryCandidates.isLoading && <p role="status">Loading accepted placements…</p>}
        {carryCandidates.isError && <p role="alert">Accepted placements are unavailable for carry-forward.</p>}
        {carryCandidates.data?.allocations.length === 0 && <EmptyState className="planner-empty-state" title="No accepted placements need a decision for this day." />}
        {carryCandidates.data && carryCandidates.data.allocations.length > 0 && <div className="review-carry-list">{carryCandidates.data.allocations.map((allocation) => <article key={allocation.id}><div><strong>{allocation.title}</strong><span>{clock(Number(allocation.startMinutes))} · {Number(allocation.durationMinutes)} min</span></div><Button type="button" className="quiet-action" variant="secondary" onClick={() => carryMutation.mutate(allocation)} disabled={carryMutation.isPending} pending={carryMutation.isPending} pendingLabel="Moving…">Carry this</Button></article>)}</div>}
        {carryMessage && <p role="status" className="review-carry-message">{carryMessage}</p>}
      </section>
      <section className="review-reflection-card" aria-labelledby="review-reflection-heading">
        <div><span className="card-kicker">OPTIONAL REFLECTION</span><h2 id="review-reflection-heading">What did you learn?</h2><p>Keep the observation, not a forced conclusion. This note is saved for {localDate}.</p></div>
        <Textarea aria-label="Daily reflection" value={reflectionText} onChange={(event) => setReflectionText(event.target.value)} maxLength={4000} placeholder="A small observation about energy, focus, or what to try next…" />
        <div className="review-reflection-footer"><span>{reflectionText.length}/4000</span><Button type="button" onClick={() => reflectionMutation.mutate()} disabled={reflectionMutation.isPending || reflectionText.trim() === ""} pending={reflectionMutation.isPending} pendingLabel="Saving…">Save reflection</Button></div>
        {reflectionMutation.isSuccess && <p role="status">Reflection saved.</p>}
        {reflectionMutation.isError && <p role="alert">Reflection could not be saved.</p>}
      </section>
    </>}
    {view === "week" && weekly.data && <>
      <div className="review-summary-grid"><article className="review-summary-planned"><span className="card-kicker">PLANNED TIME</span><strong>{Number(weekly.data.plannedMinutes)} min</strong><p>Accepted calendar allocations.</p></article><article className="review-summary-recorded"><span className="card-kicker">RECORDED ACTIVE TIME</span><strong>{Number(weekly.data.recordedActiveMinutes)} min</strong><p>Focus sessions and manual actuals.</p></article><article className="review-summary-sessions"><span className="card-kicker">FOCUS SESSIONS</span><strong>{Number(weekly.data.focusSessionCount)}</strong><p>Sessions with a recorded start.</p></article></div>
      <DataTable
        className="review-week-table"
        rows={weekly.data.days}
        columns={[
          { id: "day", header: "Day", accessor: (day) => day.localDate, sortValue: (day) => day.localDate },
          { id: "planned", header: "Planned", accessor: (day) => `${Number(day.plannedMinutes)} min`, sortValue: (day) => Number(day.plannedMinutes) },
          { id: "recorded", header: "Recorded", accessor: (day) => `${Number(day.recordedActiveMinutes)} min`, sortValue: (day) => Number(day.recordedActiveMinutes) },
          { id: "sessions", header: "Sessions", accessor: (day) => Number(day.focusSessionCount), sortValue: (day) => Number(day.focusSessionCount) },
        ]}
        getRowKey={(day) => day.localDate}
        caption="Planned and recorded activity by day"
        hideQueryControls
        hideDensityControl
        density="compact"
        tableTestId="review-week-table"
      />
      <div className="review-honesty-card"><span className="card-kicker">COVERAGE</span><p>{weekly.data.coverageNote}</p><p className="review-unknown">Active goals: {Number(weekly.data.activeGoalCount)} · Unrecorded time: unknown</p></div>
      <EvidenceInsight planned={Number(weekly.data.plannedMinutes)} recorded={Number(weekly.data.recordedActiveMinutes)} />
    </>}
  </section>;
}

function clock(minutes: number): string { return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`; }

function addDays(localDate: string, days: number): string {
  const date = new Date(`${localDate}T12:00:00`);
  date.setDate(date.getDate() + days);
  return formatLocalDate(date);
}

function mondayLocalDate(localDate: string): string {
  const date = new Date(`${localDate}T12:00:00`);
  const offset = (date.getDay() + 6) % 7;
  date.setDate(date.getDate() - offset);
  return formatLocalDate(date);
}

function formatLocalDate(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
