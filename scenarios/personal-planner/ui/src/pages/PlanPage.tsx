import { useQuery } from "@tanstack/react-query";

import { applyAllocationProposal, applyScheduleProposal, createRoutine, fetchAllocations, fetchRoutineOccurrences, fetchRoutines, fetchTodayAllocations, previewAllocation, previewSchedule, rescheduleRoutineOccurrence, skipRoutineOccurrence } from "../api/calendar";
import { createCommitment, fetchCommitments, updateCommitmentState, type Commitment } from "../api/commitments";
import { fetchForecast, fetchForecastHistory } from "../api/forecasts";
import { fetchWorkItems, updateWorkEstimate } from "../api/work";
import { RangeView } from "../components/RangeView";
import { selectors } from "../consts/selectors";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createPortal } from "react-dom";
import { CalendarDays } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { AdaptivePageHeader } from "../components/AdaptivePageHeader";
import { Popover, PopoverParts } from "@vrooli/react-component-library/Popover/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { ObservatoryScene } from "../components/ObservatoryScene";
import { useBreakpoint } from "../hooks/useBreakpoint";
import { PlannerDialog as Dialog } from "../components/PlannerDialog";

function clock(minutes: number) { return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`; }
function localDate() { const now = new Date(); return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`; }
function dateRange() {
  const now = new Date();
  const day = now.getDay();
  const monday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - (day === 0 ? 6 : day - 1));
  const sunday = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + 6);
  const format = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  return { start: format(monday), end: format(sunday) };
}
function monthRange() {
  const now = new Date();
  const format = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  return { start: format(new Date(now.getFullYear(), now.getMonth(), 1)), end: format(new Date(now.getFullYear(), now.getMonth() + 1, 0)) };
}
function isDeepWorkCandidate(item: { title: string; description?: string }) {
  return /write|draft|design|research|code|analy[sz]e|strategy|plan|build|review/i.test(`${item.title} ${item.description ?? ""}`);
}

export function PlanPage() {
  const queryClient = useQueryClient();
  const { isMobile } = useBreakpoint();
  const plan = useQuery({ queryKey: ["today-allocations"], queryFn: () => fetchTodayAllocations() });
  const work = useQuery({ queryKey: ["work-items"], queryFn: fetchWorkItems });
  const range = dateRange();
  const rangeQuery = useQuery({ queryKey: ["calendar-range", range.start, range.end], queryFn: () => fetchAllocations(range.start, range.end) });
  const month = monthRange();
  const monthQuery = useQuery({ queryKey: ["calendar-month", month.start, month.end], queryFn: () => fetchAllocations(month.start, month.end) });
  const commitments = useQuery({ queryKey: ["commitments"], queryFn: fetchCommitments });
  const forecast = useQuery({ queryKey: ["forecast", localDate()], queryFn: () => fetchForecast({ localDate: localDate(), timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC", horizonDays: 28 }) });
  const forecastHistory = useQuery({ queryKey: ["forecast-history"], queryFn: () => fetchForecastHistory(8) });
  const routines = useQuery({ queryKey: ["routines"], queryFn: fetchRoutines });
  const routineOccurrences = useQuery({ queryKey: ["routine-occurrences", range.start, range.end], queryFn: () => fetchRoutineOccurrences(range.start, range.end) });
  const [view, setView] = useState<"day" | "week" | "agenda" | "month" | "timeline" | "capacity" | "commitments" | "forecast">("day");
  const [mobilePlacementOpen, setMobilePlacementOpen] = useState(false);
  const [mobileMoreOpen, setMobileMoreOpen] = useState(false);
  const [mobileEstimateOpen, setMobileEstimateOpen] = useState(false);
  const [mobileRoutinesOpen, setMobileRoutinesOpen] = useState(false);
  const [placing, setPlacing] = useState(false);
  const [message, setMessage] = useState("");
  const [selectedWorkItem, setSelectedWorkItem] = useState("");
  const [startTime, setStartTime] = useState("09:00");
  const [duration, setDuration] = useState("45");
  const [estimateMinutes, setEstimateMinutes] = useState("");
  const [estimateReason, setEstimateReason] = useState("new_information");
  const [proposal, setProposal] = useState<{ id: string; requestedStartMinutes: number; startMinutes: number; durationMinutes: number; reason: string; state: string; baseRevision: bigint } | null>(null);
  const [scheduleProposal, setScheduleProposal] = useState<Awaited<ReturnType<typeof previewSchedule>> | null>(null);
  const [routineTitle, setRoutineTitle] = useState("");
  const [routineKind, setRoutineKind] = useState<"fixed" | "flexible">("fixed");
  const [routineWeekdays, setRoutineWeekdays] = useState<number[]>([1]);
  const [routineStart, setRoutineStart] = useState("09:00");
  const [routineDuration, setRoutineDuration] = useState("30");
  const [routineMessage, setRoutineMessage] = useState("");
  const [commitmentResult, setCommitmentResult] = useState("");
  const [commitmentBoundary, setCommitmentBoundary] = useState("");
  const [commitmentDefinition, setCommitmentDefinition] = useState("");
  const [commitmentBeneficiary, setCommitmentBeneficiary] = useState("");
  const [commitmentAssumptions, setCommitmentAssumptions] = useState("");
  const [commitmentExclusions, setCommitmentExclusions] = useState("");
  const [commitmentState, setCommitmentState] = useState<"proposed" | "active">("proposed");
  const [commitmentMessage, setCommitmentMessage] = useState("");
  const [selectedAllocation, setSelectedAllocation] = useState<{ id: string; title: string; start: string; duration: number; source: string } | null>(null);
  const placementRef = useRef<HTMLDivElement>(null);
  const skipMutation = useMutation({ mutationFn: skipRoutineOccurrence, onSuccess: async () => { setRoutineMessage("Occurrence skipped; the series remains unchanged."); await queryClient.invalidateQueries({ queryKey: ["routine-occurrences"] }); await queryClient.invalidateQueries({ queryKey: ["routines"] }); }, onError: () => setRoutineMessage("That occurrence could not be skipped. Reload the routine and try again.") });
  const rescheduleMutation = useMutation({ mutationFn: rescheduleRoutineOccurrence, onSuccess: async () => { setRoutineMessage("Occurrence moved 30 minutes later; the series remains unchanged."); await queryClient.invalidateQueries({ queryKey: ["routine-occurrences"] }); await queryClient.invalidateQueries({ queryKey: ["routines"] }); }, onError: () => setRoutineMessage("That occurrence could not be moved. Reload the routine and try again.") });
  const routineMutation = useMutation({ mutationFn: () => { const [hours = 9, minutes = 0] = routineStart.split(":").map(Number); return createRoutine({ title: routineTitle, kind: routineKind, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC", startDate: localDate(), endDate: "", weekdays: routineWeekdays, startMinute: hours * 60 + minutes, durationMinutes: Number(routineDuration), frequencyPerWeek: routineKind === "fixed" ? routineWeekdays.length : Math.min(2, routineWeekdays.length) }); }, onSuccess: async () => { setRoutineMessage("Routine saved; future occurrences are now visible in the accepted planning horizon."); setRoutineTitle(""); await queryClient.invalidateQueries({ queryKey: ["routines"] }); await queryClient.invalidateQueries({ queryKey: ["routine-occurrences"] }); }, onError: () => setRoutineMessage("Routine could not be saved. Check its time and eligible weekdays.") });
  const commitmentMutation = useMutation({ mutationFn: () => createCommitment({ result: commitmentResult, definitionOfDone: commitmentDefinition, promisedBoundary: commitmentBoundary, beneficiary: commitmentBeneficiary, assumptions: commitmentAssumptions, scopeExclusions: commitmentExclusions, state: commitmentState, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC" }), onSuccess: async () => { setCommitmentMessage("Commitment recorded. Forecasts will never rewrite this promise silently."); setCommitmentResult(""); setCommitmentBoundary(""); setCommitmentDefinition(""); setCommitmentBeneficiary(""); setCommitmentAssumptions(""); setCommitmentExclusions(""); await queryClient.invalidateQueries({ queryKey: ["commitments"] }); }, onError: () => setCommitmentMessage("Commitment could not be saved. Keep the promise boundary explicit and try again.") });
  const commitmentStateMutation = useMutation({ mutationFn: ({ commitment, state }: { commitment: Commitment; state: string }) => updateCommitmentState(commitment, state), onSuccess: async () => { setCommitmentMessage("Commitment state updated; its original promise remains in history."); await queryClient.invalidateQueries({ queryKey: ["commitments"] }); }, onError: () => setCommitmentMessage("That commitment changed elsewhere. Reload its current revision before updating it again.") });
  const firstWorkItem = work.data?.[0];
  const selectedPlacementItem = work.data?.find((item) => item.id === (selectedWorkItem || firstWorkItem?.id));
  const requestedHour = Number(startTime.split(":")[0] ?? 9);
  const energyHint = selectedPlacementItem && isDeepWorkCandidate(selectedPlacementItem) && requestedHour >= 12;
  useEffect(() => { if (firstWorkItem && !estimateMinutes) setEstimateMinutes(String(firstWorkItem.remainingMinutes)); }, [firstWorkItem]);
  const estimateMutation = useMutation({ mutationFn: () => updateWorkEstimate({ id: selectedWorkItem || firstWorkItem?.id || "", remainingMinutes: Number(estimateMinutes), reason: estimateReason }), onSuccess: async () => { setMessage("Estimate updated and remembered for future calibration."); setMobileEstimateOpen(false); await queryClient.invalidateQueries({ queryKey: ["work-items"] }); }, onError: () => setMessage("That estimate could not be saved. Check the item and try again.") });
  const timelineAllocations = plan.data?.allocations.slice().sort((left, right) => Number(left.startMinutes) - Number(right.startMinutes) || Number(right.durationMinutes) - Number(left.durationMinutes)).reduce<Array<{ entry: (typeof plan.data.allocations)[number]; lane: number }>>((items, entry) => {
    const start = Number(entry.startMinutes);
    const end = start + Number(entry.durationMinutes);
    let lane = 0;
    while (items.some((item) => item.lane === lane && start < Number(item.entry.startMinutes) + Number(item.entry.durationMinutes) && end > Number(item.entry.startMinutes))) lane += 1;
    return [...items, { entry, lane }];
  }, []) ?? [];
  const timelineLanes = timelineAllocations.length ? Math.max(...timelineAllocations.map((item) => item.lane)) + 1 : 1;
  const requestedPlacement = () => {
    const item = work.data?.find((candidate) => candidate.id === (selectedWorkItem || firstWorkItem?.id));
    if (!item) return null;
    const [hours = 9, minutes = 0] = startTime.split(":").map(Number);
    return { workItemId: item.id, localDate: localDate(), startMinutes: hours * 60 + minutes, durationMinutes: Number(duration) };
  };
  const previewNext = async () => {
    const input = requestedPlacement();
    if (!input) return;
    setPlacing(true); setMessage("");
    try {
      const next = await previewAllocation(input);
      setProposal({ ...next, requestedStartMinutes: input.startMinutes });
      setMessage(next.state === "feasible" ? `Preview ready: ${clock(Number(next.startMinutes))} for ${Number(next.durationMinutes)} minutes.` : next.reason);
    } catch { setMessage("That placement could not be previewed. Reload the plan and try again."); } finally { setPlacing(false); }
  };
  const placeNext = async () => {
    const input = requestedPlacement();
    if (!input || !proposal || proposal.state !== "feasible") return;
    setPlacing(true); setMessage("");
    try {
      await applyAllocationProposal({ proposalId: proposal.id, expectedRevision: proposal.baseRevision, idempotencyKey: `${proposal.id}:apply` });
      setProposal(null); await plan.refetch(); setMobilePlacementOpen(false); setMessage("Placed on today’s accepted schedule; the server rechecked the current schedule.");
    } catch { setProposal(null); setMessage("That proposal is stale or could not be accepted. Preview again to use current schedule state."); } finally { setPlacing(false); }
  };
  const previewBacklog = async () => {
    const ids = (work.data ?? []).filter((item) => Number(item.remainingMinutes) > 0).slice(0, 3).map((item) => item.id);
    if (ids.length < 2) return;
    const [hours = 9, minutes = 0] = startTime.split(":").map(Number);
    setPlacing(true); setMessage("");
    try { setScheduleProposal(await previewSchedule({ localDate: localDate(), startMinutes: hours * 60 + minutes, workItemIds: ids })); }
    catch { setMessage("The selected backlog could not be previewed. Reload the plan and try again."); }
    finally { setPlacing(false); }
  };
  const applyBacklog = async () => {
    if (!scheduleProposal) return;
    setPlacing(true); setMessage("");
    try { const allocations = await applyScheduleProposal({ proposalId: scheduleProposal.id, expectedRevision: scheduleProposal.baseRevision, idempotencyKey: `${scheduleProposal.id}:apply` }); setScheduleProposal(null); await plan.refetch(); setMessage(`Accepted ${allocations.length} sessions after a server recheck.`); }
    catch { setScheduleProposal(null); setMessage("That schedule proposal is stale or could not be accepted. Preview the backlog again."); }
    finally { setPlacing(false); }
  };
  const revealPlacement = () => {
    if (isMobile) {
      setMobilePlacementOpen(true);
      return;
    }
    placementRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  };
  const selectMobileView = (nextView: typeof view) => {
    setView(nextView);
    setMobileMoreOpen(false);
  };
  const placementForm = (mobile = false) => <div ref={mobile ? undefined : placementRef} className={`plan-placement${mobile ? " plan-placement-sheet" : ""}`}>
    <FormField required label="Work item" control={<Select required id={mobile ? "plan-work-item-mobile" : "plan-work-item"} aria-label="Work item" value={selectedWorkItem || firstWorkItem?.id} onChange={(event) => setSelectedWorkItem(event.target.value)} options={(work.data ?? []).map((item) => ({ value: item.id, label: item.title }))} />} />
    <FormField required label="Start" control={<Input required id={mobile ? "plan-start-time-mobile" : "plan-start-time"} aria-label="Start" type="time" value={startTime} onChange={(event) => setStartTime(event.target.value)} />} />
    <FormField required label="Minutes" control={<Input required id={mobile ? "plan-duration-mobile" : "plan-duration"} aria-label="Minutes" type="number" min="1" max="720" step="5" value={duration} onChange={(event) => setDuration(event.target.value)} />} />
    {energyHint && <div className="energy-suggestion" aria-label="Morning energy suggestion"><div><span className="card-kicker">MORNING WINDOW</span><strong>This looks like deep work.</strong><p>The planner heuristic protects higher-focus work earlier in the day, when the morning window is still open.</p></div><Button type="button" variant="secondary" size="sm" onClick={() => setStartTime("09:00")}>Use 09:00</Button></div>}
    <Button type="button" className="secondary-action" variant="secondary" onClick={() => void previewNext()} disabled={placing || work.isLoading || !duration} pending={placing} pendingLabel="Checking…">Preview placement</Button>
    {proposal?.state === "feasible" && <Button type="button" className="primary-action" onClick={() => void placeNext()} disabled={placing} pending={placing} pendingLabel="Placing…">Accept {clock(proposal.startMinutes)}</Button>}
    {(work.data?.filter((item) => Number(item.remainingMinutes) > 0).length ?? 0) >= 2 && <Button type="button" className="secondary-action" variant="secondary" onClick={() => void previewBacklog()} disabled={placing} pending={placing} pendingLabel="Checking…">Preview next 3</Button>}
    {mobile && proposal && <div className={`placement-proposal ${proposal.state}`} role="status"><span className="card-kicker">{proposal.state === "feasible" ? "PROPOSED · NOT ACCEPTED" : "NO FEASIBLE SLOT"}</span><strong>{proposal.state === "feasible" ? `${clock(proposal.startMinutes)} · ${proposal.durationMinutes} min` : "Placement blocked"}</strong>{proposal.state === "feasible" && <p className="placement-diff"><span>Requested {clock(proposal.requestedStartMinutes)}</span><span aria-hidden="true">→</span><span>Proposed {clock(proposal.startMinutes)}</span></p>}<p>{proposal.reason}</p><small>Preview only. Nothing changes until you accept.</small></div>}
  </div>;
  const estimateEditor = (mobile = false) => <form className={`plan-estimate-editor${mobile ? " plan-estimate-sheet" : ""}`} onSubmit={(event) => { event.preventDefault(); estimateMutation.mutate(); }}>
    <div><span className="card-kicker">LEARNING SIGNAL</span><h2>Keep the estimate honest</h2><p>When the shape of work changes, record the new remaining effort and why. The original estimate stays intact for calibration.</p></div>
    <FormField label="Work item" control={<Select aria-label="Estimate work item" value={selectedWorkItem || firstWorkItem?.id} onChange={(event) => { const id = event.target.value; setSelectedWorkItem(id); setEstimateMinutes(String(work.data?.find((item) => item.id === id)?.remainingMinutes ?? 0)); }} options={(work.data ?? []).map((item) => ({ value: item.id, label: item.title }))} />} />
    <FormField required label="Remaining minutes" control={<Input required aria-label="Remaining minutes" type="number" min="0" max="10080" step="5" value={estimateMinutes} onChange={(event) => setEstimateMinutes(event.target.value)} />} />
    <FormField label="Why did it change?" control={<Select aria-label="Estimate change reason" value={estimateReason} onChange={(event) => setEstimateReason(event.target.value)} options={[{ value: "new_information", label: "New information" }, { value: "scope_changed", label: "Scope changed" }, { value: "blocked", label: "Blocked" }, { value: "better_understood", label: "Better understood" }]} />} />
    <Button type="submit" variant="secondary" pending={estimateMutation.isPending} pendingLabel="Remembering…" disabled={estimateMutation.isPending || estimateMinutes === ""}>Save estimate</Button>
  </form>;
  const routinePanel = (mobile = false) => {
    const headingId = mobile ? "plan-routines-mobile-heading" : "plan-routines-heading";
    return <section className={`plan-routines${mobile ? " plan-routines-sheet" : ""}`} aria-labelledby={headingId}>
      <div><span className="card-kicker">RHYTHM</span><h2 id={headingId}>Routines that leave room</h2><p>Fixed appointments and flexible frequency are kept distinct. Occurrences are generated in local time; they do not silently become accepted work blocks.</p></div>
      {routines.data?.length ? <div className="routine-list">{routines.data.map((routine) => <article key={routine.id}><strong>{routine.title}</strong><span>{routine.kind} · {Number(routine.durationMinutes)} min · {routine.timezone}</span></article>)}</div> : <EmptyState className="planner-empty-state" title="No routines yet. Add a small rhythm only when it helps the week." />}
      {routineOccurrences.data?.length ? <div className="routine-occurrences" aria-label="Upcoming routine occurrences">{routineOccurrences.data.slice(0, 5).map((occurrence) => { const routine = routines.data?.find((candidate) => candidate.id === occurrence.routineId); return <div key={`${occurrence.routineId}-${occurrence.localDate}`}><span>{occurrence.localDate} · {Number(occurrence.startMinute) / 60 | 0}:{String(Number(occurrence.startMinute) % 60).padStart(2, "0")} · {Number(occurrence.durationMinutes)} min · {occurrence.title}</span><span className="routine-occurrence-actions"><Button type="button" variant="secondary" size="sm" shape="square" onClick={() => routine && rescheduleMutation.mutate({ routineId: occurrence.routineId, localDate: occurrence.localDate, startMinute: Math.min(1439, Number(occurrence.startMinute) + 30), expectedRevision: routine.revision })} disabled={rescheduleMutation.isPending || skipMutation.isPending || !routine}>Move +30m</Button><Button type="button" variant="secondary" size="sm" shape="square" onClick={() => routine && skipMutation.mutate({ routineId: occurrence.routineId, localDate: occurrence.localDate, expectedRevision: routine.revision })} disabled={skipMutation.isPending || rescheduleMutation.isPending || !routine}>Skip once</Button></span></div>; })}</div> : null}
      <form className="routine-form" onSubmit={(event) => { event.preventDefault(); routineMutation.mutate(); }}><FormField label="Routine title" required control={<Input aria-label="Routine title" value={routineTitle} onChange={(event) => setRoutineTitle(event.target.value)} />} /><FormField label="Type" control={<Select aria-label="Routine type" value={routineKind} onChange={(event) => setRoutineKind(event.target.value as "fixed" | "flexible")} options={[{ value: "fixed", label: "Fixed recurrence" }, { value: "flexible", label: "Flexible frequency" }]} />} /><FormField label="Routine start" control={<Input aria-label="Routine start" type="time" value={routineStart} onChange={(event) => setRoutineStart(event.target.value)} />} /><FormField label="Routine minutes" control={<Input aria-label="Routine minutes" type="number" min="1" max="480" value={routineDuration} onChange={(event) => setRoutineDuration(event.target.value)} />} /><fieldset className="routine-days" aria-label="Eligible weekdays"><legend>Eligible weekdays</legend>{[1,2,3,4,5,6,7].map((day) => <div className="routine-day" key={day}><Checkbox aria-label={`${["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"][day - 1]} eligible`} checked={routineWeekdays.includes(day)} onCheckedChange={(checked) => setRoutineWeekdays((days) => checked ? [...days, day].sort() : days.filter((candidate) => candidate !== day))} /><span aria-hidden="true">{["M","T","W","T","F","S","S"][day - 1]}</span></div>)}</fieldset><Button className="secondary-action" variant="secondary" type="submit" disabled={routineMutation.isPending || !routineTitle || routineWeekdays.length === 0} pending={routineMutation.isPending} pendingLabel="Saving…">Add routine</Button></form>
      {routineMessage && <p role="status" className="plan-message">{routineMessage}</p>}
    </section>;
  };
  return <ObservatoryScene kind="plan"><section className="planner-surface plan-surface" data-testid={selectors.pages.plan} aria-labelledby="plan-heading">
    <AdaptivePageHeader className="planner-page-header" headingId="plan-heading" eyebrow="A day you can keep" title="Plan" description="Place focused work on the schedule. Capacity reflects accepted allocations." leading={<span className="planner-page-mark" aria-hidden="true"><CalendarDays size={21} /></span>} />
    {plan.isLoading && <p role="status">Calculating today’s plan…</p>}
    {plan.isError && <p role="alert">Today’s plan is unavailable right now.</p>}
    {plan.data && <>
      <div className={`plan-view-switcher ${isMobile ? "plan-view-switcher-mobile" : ""}`} role="tablist" aria-label="Planning view"><Button type="button" variant={view === "day" ? "primary" : "secondary"} role="tab" aria-selected={view === "day"} onClick={() => setView("day")}>Day</Button><Button type="button" variant={view === "week" ? "primary" : "secondary"} role="tab" aria-selected={view === "week"} onClick={() => setView("week")}>Week</Button>{!isMobile && <><Button type="button" variant={view === "agenda" ? "primary" : "secondary"} role="tab" aria-selected={view === "agenda"} onClick={() => setView("agenda")}>Agenda</Button><Button type="button" variant={view === "month" ? "primary" : "secondary"} role="tab" aria-selected={view === "month"} onClick={() => setView("month")}>Month</Button><Button type="button" variant={view === "timeline" ? "primary" : "secondary"} role="tab" aria-selected={view === "timeline"} onClick={() => setView("timeline")}>Timeline</Button><Button type="button" variant={view === "capacity" ? "primary" : "secondary"} role="tab" aria-selected={view === "capacity"} onClick={() => setView("capacity")}>Capacity</Button><Button type="button" variant={view === "commitments" ? "primary" : "secondary"} role="tab" aria-selected={view === "commitments"} onClick={() => setView("commitments")}>Commitments</Button><Button type="button" variant={view === "forecast" ? "primary" : "secondary"} role="tab" aria-selected={view === "forecast"} onClick={() => setView("forecast")}>Outlook</Button></>}{isMobile && <Button type="button" variant="secondary" role="tab" aria-haspopup="dialog" aria-expanded={mobileMoreOpen} onClick={() => setMobileMoreOpen(true)}>More</Button>}{!isMobile && work.data && firstWorkItem && <Button type="button" className="plan-desktop-place-action primary-action" onClick={revealPlacement}>+ Place work</Button>}</div>
      {view === "day" && (isMobile ? <div className="plan-mobile-capacity" aria-label={`${Number(plan.data.plannedMinutes)} minutes planned of ${Number(plan.data.plannedMinutes) + Number(plan.data.availableMinutes)} available minutes`}><div><span className="card-kicker">TODAY’S CAPACITY</span><strong>{Number(plan.data.plannedMinutes)} min <small>/ {Number(plan.data.plannedMinutes) + Number(plan.data.availableMinutes)} min</small></strong></div><span className="plan-mobile-capacity-ring" aria-hidden="true" style={{ background: `conic-gradient(var(--color-accent) ${Math.min(100, Math.max(0, Number(plan.data.plannedMinutes) / Math.max(1, Number(plan.data.plannedMinutes) + Number(plan.data.availableMinutes)) * 100))}%, color-mix(in srgb, var(--color-muted-foreground) 18%, transparent) 0)` }}><span>{Math.round(Number(plan.data.plannedMinutes) / Math.max(1, Number(plan.data.plannedMinutes) + Number(plan.data.availableMinutes)) * 100)}%</span></span></div> : <div className="plan-capacity-strip"><div><span className="card-kicker">PLANNED</span><strong>{Number(plan.data.plannedMinutes)} min</strong></div><div><span className="card-kicker">AVAILABLE</span><strong>{Number(plan.data.availableMinutes)} min</strong></div><div><span className="card-kicker">BREATHING ROOM</span><strong>{Number(plan.data.breathingRoomMinutes)} min</strong></div></div>)}
      {isMobile && view === "day" && work.data && firstWorkItem && typeof document !== "undefined" && createPortal(<div className="plan-mobile-action-bar" aria-label="Plan actions"><Button type="button" className="primary-action" onClick={revealPlacement}>+ Place work</Button><Button type="button" variant="secondary" aria-label="More planning views" onClick={() => setMobileMoreOpen(true)}>⋯</Button></div>, document.body)}
      {isMobile && firstWorkItem && <Dialog open={mobilePlacementOpen} title="Plot work on today’s chart" description="Preview the route first. Nothing is accepted until you confirm the proposed time." onClose={() => { setMobilePlacementOpen(false); setProposal(null); }} closeLabel="Close placement" contentClassName="plan-placement-dialog">{placementForm(true)}</Dialog>}
      {isMobile && <Dialog open={mobileMoreOpen} title="More planning views" description="Open one supporting view without crowding the day plan." onClose={() => setMobileMoreOpen(false)} closeLabel="Close planning views" contentClassName="plan-more-dialog"><div className="plan-more-grid">{([['agenda', 'Agenda'], ['month', 'Month'], ['timeline', 'Timeline'], ['capacity', 'Capacity'], ['commitments', 'Commitments'], ['forecast', 'Outlook']] as const).map(([nextView, label]) => <Button key={nextView} type="button" variant={view === nextView ? "primary" : "secondary"} onClick={() => selectMobileView(nextView)}>{label}</Button>)}<Button type="button" variant="secondary" onClick={() => { setMobileMoreOpen(false); setMobileRoutinesOpen(true); }}>Routines</Button><Button type="button" variant="secondary" onClick={() => { setMobileMoreOpen(false); setMobileEstimateOpen(true); }}>Update estimate</Button></div></Dialog>}
      {isMobile && firstWorkItem && <Dialog open={mobileEstimateOpen} title="Update the estimate" description="Record what changed without rewriting the original prediction." onClose={() => setMobileEstimateOpen(false)} closeLabel="Close estimate editor" contentClassName="plan-estimate-dialog">{estimateEditor(true)}</Dialog>}
      {isMobile && <Dialog open={mobileRoutinesOpen} title="Routines" description="Shape recurring rhythms without crowding today’s chart." onClose={() => setMobileRoutinesOpen(false)} closeLabel="Close routines" contentClassName="plan-routines-dialog">{routinePanel(true)}</Dialog>}
      {view === "capacity" && <section className="plan-capacity-view" aria-labelledby="plan-capacity-heading"><div className="plan-capacity-intro"><span className="card-kicker">CAPACITY YOU CAN TRUST</span><h2 id="plan-capacity-heading">Make room before you make promises.</h2><p>Accepted work, provider time, and your breathing-room reserve are shown separately so the day stays honest.</p></div><div className="plan-capacity-meter" aria-label={`${Number(plan.data.plannedMinutes)} of ${Number(plan.data.plannedMinutes) + Number(plan.data.availableMinutes)} planned minutes used`}><div className="plan-capacity-meter-track"><span style={{ width: `${Math.min(100, Math.max(0, Number(plan.data.plannedMinutes) / Math.max(1, Number(plan.data.plannedMinutes) + Number(plan.data.availableMinutes)) * 100))}%` }} /></div><div><span>{Number(plan.data.plannedMinutes)} min accepted</span><span>{Number(plan.data.availableMinutes)} min still available</span></div></div><div className="plan-capacity-grid"><article><span className="card-kicker">ACCEPTED WORK</span><strong>{Number(plan.data.plannedMinutes)} min</strong><p>{plan.data.allocations.length ? `${plan.data.allocations.length} placed ${plan.data.allocations.length === 1 ? "item" : "items"}.` : "Nothing is committed yet."}</p></article><article><span className="card-kicker">PROTECTED RESERVE</span><strong>{Number(plan.data.breathingRoomMinutes)} min</strong><p>This time stays available for the unexpected.</p></article><article><span className="card-kicker">PROVIDER TIME</span><strong>{Number(plan.data.externalBusyMinutes)} min</strong><p>{Number(plan.data.externalEventCount) ? `${Number(plan.data.externalEventCount)} read-only calendar ${Number(plan.data.externalEventCount) === 1 ? "hold" : "holds"}.` : "No provider holds are recorded."}</p></article></div>{plan.data.externalFreshness && <p className="plan-capacity-freshness">Provider freshness: {plan.data.externalFreshness}.</p>}<div className="plan-capacity-list"><div><span className="card-kicker">ACCEPTED ALLOCATIONS</span><strong>{plan.data.allocations.length ? "What is already spoken for" : "A clear day"}</strong></div>{plan.data.allocations.length ? <ul>{plan.data.allocations.slice().sort((left, right) => Number(left.startMinutes) - Number(right.startMinutes)).map((entry) => <li key={entry.id}><span>{clock(Number(entry.startMinutes))} · {entry.title}</span><small>{Number(entry.durationMinutes)} min · {entry.sourceLabel || "Personal Planner"}</small></li>)}</ul> : <p>No accepted work is taking space yet. Return to Day when you are ready to place the next step.</p>}</div></section>}
      {view === "forecast" && <section className="plan-capacity-view plan-forecast-view" aria-labelledby="plan-forecast-heading"><div className="plan-capacity-intro"><span className="card-kicker">OUTLOOK, NOT PROMISE</span><h2 id="plan-forecast-heading">See what the shared resource can carry.</h2><p>These are deterministic central and cautious scenarios over known remaining work. They are not probabilities, and they never rewrite an accepted schedule or commitment.</p></div>{forecast.isLoading && <p role="status">Refreshing the outlook…</p>}{forecast.isError && <p role="alert">The outlook is unavailable right now; accepted planning data is still safe.</p>}{forecast.data && <><div className="plan-forecast-summary"><article><span className="card-kicker">CENTRAL SCENARIO</span><strong>{forecast.data.centralFinish || "Not enough known effort"}</strong><p>{forecast.data.resultState.replace(/_/g, " ")}</p></article><article><span className="card-kicker">CAUTIOUS SCENARIO</span><strong>{forecast.data.cautiousFinish || "Not available"}</strong><p>{forecast.data.riskState.replace(/_/g, " ")}</p></article></div><div className="plan-forecast-explanation"><span className="card-kicker">WHY THIS OUTLOOK</span><p>{forecast.data.explanation}</p><small>Horizon {forecast.data.horizonStart} → {forecast.data.horizonEnd} · {Number(forecast.data.knownWorkMinutes)} min known work · {Number(forecast.data.reserveMinutes)} min protected reserve · {forecast.data.freshness}</small></div>{forecast.data.changeExplanation && <div className="plan-forecast-change" role="status"><span className="card-kicker">WHAT CHANGED</span><p>{forecast.data.changeExplanation}</p></div>}{forecast.data.commitmentOutlooks.length > 0 && <div className="plan-forecast-commitments"><span className="card-kicker">PROMISE CHECK</span><strong>Promise and outlook, side by side</strong><ul>{forecast.data.commitmentOutlooks.map((commitment) => <li key={commitment.id}><div><b>{commitment.result}</b><small>Promised {commitment.promisedBoundary} · forecast {commitment.forecastFinish || "unknown"}</small></div><span data-risk={commitment.riskState}>{commitment.riskState.replace(/_/g, " ")}</span><p>{commitment.explanation}</p></li>)}</ul></div>}{forecastHistory.isLoading && <p role="status">Loading recent outlooks…</p>}{forecastHistory.isError && <p role="alert">Recent outlook history is unavailable.</p>}{!forecastHistory.isLoading && !forecastHistory.isError && <div className="plan-forecast-history"><span className="card-kicker">RECENT OUTLOOKS</span>{forecastHistory.data?.length ? <ul>{forecastHistory.data.map((snapshot) => <li key={snapshot.id}><div><b>{snapshot.generatedAt ? new Date(snapshot.generatedAt).toLocaleString() : "Snapshot"}</b><small>{snapshot.horizonStart} → {snapshot.horizonEnd} · {snapshot.resultState.replace(/_/g, " ")}</small></div><span data-risk={snapshot.riskState}>{snapshot.riskState.replace(/_/g, " ")}</span><p>Central {snapshot.centralFinish || "not available"} · cautious {snapshot.cautiousFinish || "not available"}</p>{snapshot.changeExplanation && <small>{snapshot.changeExplanation}</small>}</li>)}</ul> : <p>No forecast snapshots yet. Generate an outlook to start the history.</p>}</div>}</>}</section>}
      {view === "commitments" && <section className="plan-commitments-view" aria-labelledby="plan-commitments-heading"><div className="plan-capacity-intro"><span className="card-kicker">PROMISES, PRESERVED</span><h2 id="plan-commitments-heading">Keep the promise visible.</h2><p>A commitment is an explicit promise, not a forecast. Revisions preserve the original boundary and acknowledgment stays unknown until someone actually supplies it.</p></div>{commitments.isLoading && <p role="status">Loading commitments…</p>}{commitments.isError && <p role="alert">Commitments are unavailable right now.</p>}{commitments.data?.length ? <div className="plan-commitment-list">{commitments.data.map((commitment) => <article key={commitment.id}><div><span className="card-kicker">{commitment.state.toUpperCase()} · {commitment.risk.toUpperCase()} RISK</span><h3>{commitment.result}</h3><p>Promised {commitment.promisedBoundary} · acknowledgment {commitment.acknowledgmentStatus}.</p>{commitment.definitionOfDone && <p><strong>Done means:</strong> {commitment.definitionOfDone}</p>}{commitment.beneficiary && <p><strong>For:</strong> {commitment.beneficiary}</p>}</div>{(commitment.state === "proposed" || commitment.state === "active") && <Button type="button" variant="secondary" size="sm" onClick={() => commitmentStateMutation.mutate({ commitment, state: commitment.state === "proposed" ? "active" : "fulfilled" })}>{commitment.state === "proposed" ? "Accept promise" : "Mark fulfilled"}</Button>}</article>)}</div> : !commitments.isLoading && <EmptyState className="planner-empty-state" title="No promises recorded yet. Add one only when its boundary is explicit." />}<form className="commitment-form" onSubmit={(event) => { event.preventDefault(); commitmentMutation.mutate(); }}><FormField required label="Promised result" control={<Input required aria-label="Promised result" value={commitmentResult} onChange={(event) => setCommitmentResult(event.target.value)} />} /><FormField required label="Promise boundary" control={<Input required aria-label="Promise boundary" placeholder="2026-10-01 or 2026-10-01T18:00" value={commitmentBoundary} onChange={(event) => setCommitmentBoundary(event.target.value)} />} /><FormField label="What counts as done?" control={<Textarea aria-label="What counts as done?" rows={2} value={commitmentDefinition} onChange={(event) => setCommitmentDefinition(event.target.value)} placeholder="The observable result someone can verify…" />} /><FormField label="Who benefits?" control={<Input aria-label="Who benefits?" value={commitmentBeneficiary} onChange={(event) => setCommitmentBeneficiary(event.target.value)} placeholder="A person, team, or future self" />} /><FormField label="Assumptions" control={<Textarea aria-label="Assumptions" rows={2} value={commitmentAssumptions} onChange={(event) => setCommitmentAssumptions(event.target.value)} placeholder="Conditions this promise depends on…" />} /><FormField label="Out of scope" control={<Textarea aria-label="Out of scope" rows={2} value={commitmentExclusions} onChange={(event) => setCommitmentExclusions(event.target.value)} placeholder="What this promise deliberately does not include…" />} /><FormField label="Starting state" control={<Select aria-label="Starting state" value={commitmentState} onChange={(event) => setCommitmentState(event.target.value as "proposed" | "active")} options={[{ value: "proposed", label: "Proposed" }, { value: "active", label: "Active" }]} />} /><Button type="submit" variant="secondary" pending={commitmentMutation.isPending} pendingLabel="Recording…" disabled={commitmentMutation.isPending || !commitmentResult.trim() || !commitmentBoundary.trim()}>Record commitment</Button></form>{commitmentMessage && <p role="status" className="plan-message">{commitmentMessage}</p>}</section>}
      {view !== "capacity" && view !== "commitments" && view !== "forecast" && plan.data.allocations.length === 0 && <EmptyState className="planner-empty-state" title="No accepted allocations yet. Place the next work item when you are ready." />}
      {!isMobile && view !== "capacity" && view !== "commitments" && view !== "forecast" && work.data && firstWorkItem && placementForm()}
      {!isMobile && view === "day" && work.data && firstWorkItem && estimateEditor()}
      {!isMobile && proposal && <div className={`placement-proposal ${proposal.state}`} role="status"><span className="card-kicker">{proposal.state === "feasible" ? "PROPOSED · NOT ACCEPTED" : "NO FEASIBLE SLOT"}</span><strong>{proposal.state === "feasible" ? `${clock(proposal.startMinutes)} · ${proposal.durationMinutes} min` : "Placement blocked"}</strong>{proposal.state === "feasible" && <p className="placement-diff"><span>Requested {clock(proposal.requestedStartMinutes)}</span><span aria-hidden="true">→</span><span>Proposed {clock(proposal.startMinutes)}</span></p>}<p>{proposal.reason}</p><small>Preview only. Nothing changes until you accept.</small></div>}
      {scheduleProposal && <div className={`placement-proposal schedule-proposal ${scheduleProposal.state}`} role="status"><span className="card-kicker">{scheduleProposal.state.toUpperCase()} · NOT ACCEPTED</span><strong>Next sessions in request order</strong><ul>{scheduleProposal.placements.map((placement) => <li key={placement.workItemId}><span>{placement.title}</span><span>{placement.state === "feasible" ? `${clock(Number(placement.startMinutes))} · ${Number(placement.durationMinutes)} min` : placement.reason}</span></li>)}</ul><p>{scheduleProposal.reason}</p>{scheduleProposal.placements.some((placement) => placement.state === "feasible") && <Button type="button" className="primary-action" onClick={() => void applyBacklog()} disabled={placing} pending={placing} pendingLabel="Accepting…">Accept feasible sessions</Button>}<small>Preview only. Nothing changes until you accept.</small></div>}
      {view === "day" && plan.data.allocations.length > 0 && <div className="plan-timeline" aria-label="Today’s accepted schedule" style={{ minHeight: `${Math.max(15, 3 + timelineLanes * 7)}rem` }}><div className="plan-timeline-scale" aria-hidden="true">{Array.from({ length: 11 }, (_, index) => <span key={index}>{clock(540 + index * 60)}</span>)}</div>{timelineAllocations.map(({ entry, lane }) => {
        const detail = { id: entry.id, title: entry.title, start: clock(Number(entry.startMinutes)), duration: Number(entry.durationMinutes), source: entry.sourceLabel || "Personal Planner" };
        const compact = detail.duration < 60;
        return <Popover key={entry.id} open={selectedAllocation?.id === entry.id} onOpenChange={(open) => setSelectedAllocation(open ? detail : null)} placement="bottom-start" responsive="auto">
          <PopoverParts.Trigger asChild aria-label={`${entry.title}, ${detail.start}, ${detail.duration} minutes`}>
            <article className={`plan-block${compact ? " plan-block-compact" : ""}`} role="group" tabIndex={0} aria-label={`${entry.title}, ${detail.start}, ${detail.duration} minutes`} aria-roledescription="interactive schedule block" title={entry.title} data-full-title={entry.title} style={{ left: `${Math.max(0, (Number(entry.startMinutes) - 540) / 600 * 100)}%`, width: `${Math.max(.5, Number(entry.durationMinutes) / 600 * 100)}%`, top: `${2.35 + lane * 7}rem` }}><span>{detail.start} · {detail.duration} min</span><h2>{compact ? `${detail.duration}m` : entry.title}</h2><small>{detail.source}</small></article>
          </PopoverParts.Trigger>
          <PopoverParts.Content className="timeline-detail plan-detail" aria-label="Accepted allocation details" initialFocus="first"><div><strong>{detail.title}</strong><Button type="button" variant="ghost" size="icon" shape="pill" onClick={() => setSelectedAllocation(null)} aria-label="Close allocation details">×</Button></div><p>{detail.start} · {detail.duration} min</p><small>{detail.source}</small></PopoverParts.Content>
        </Popover>;
      })}</div>}
      {view !== "day" && view !== "capacity" && view !== "commitments" && view !== "forecast" && <RangeView view={view} query={(view === "month" ? monthQuery.data?.allocations : rangeQuery.data?.allocations) ?? []} loading={view === "month" ? monthQuery.isLoading : rangeQuery.isLoading} error={view === "month" ? monthQuery.isError : rangeQuery.isError} />}
      {!isMobile && routinePanel()}
      {message && <p role="status" className="plan-message">{message}</p>}
      <p className="plan-boundary">Accepted placement is durable schedule state. Completing focus does not silently mark an allocation or work item complete.</p>
    </>}
  </section></ObservatoryScene>;
}
