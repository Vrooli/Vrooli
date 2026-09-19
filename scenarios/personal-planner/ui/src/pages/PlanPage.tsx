import { useQuery } from "@tanstack/react-query";

import { createAllocation, createRoutine, fetchAllocations, fetchRoutineOccurrences, fetchRoutines, fetchTodayAllocations, rescheduleRoutineOccurrence, skipRoutineOccurrence } from "../api/calendar";
import { fetchWorkItems } from "../api/work";
import { RangeView } from "../components/RangeView";
import { selectors } from "../consts/selectors";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

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

export function PlanPage() {
  const queryClient = useQueryClient();
  const plan = useQuery({ queryKey: ["today-allocations"], queryFn: () => fetchTodayAllocations() });
  const work = useQuery({ queryKey: ["work-items"], queryFn: fetchWorkItems });
  const range = dateRange();
  const rangeQuery = useQuery({ queryKey: ["calendar-range", range.start, range.end], queryFn: () => fetchAllocations(range.start, range.end) });
  const routines = useQuery({ queryKey: ["routines"], queryFn: fetchRoutines });
  const routineOccurrences = useQuery({ queryKey: ["routine-occurrences", range.start, range.end], queryFn: () => fetchRoutineOccurrences(range.start, range.end) });
  const [view, setView] = useState<"day" | "week" | "agenda">("day");
  const [placing, setPlacing] = useState(false);
  const [message, setMessage] = useState("");
  const [selectedWorkItem, setSelectedWorkItem] = useState("");
  const [startTime, setStartTime] = useState("09:00");
  const [duration, setDuration] = useState("45");
  const [routineTitle, setRoutineTitle] = useState("");
  const [routineKind, setRoutineKind] = useState<"fixed" | "flexible">("fixed");
  const [routineWeekdays, setRoutineWeekdays] = useState<number[]>([1]);
  const [routineStart, setRoutineStart] = useState("09:00");
  const [routineDuration, setRoutineDuration] = useState("30");
  const [routineMessage, setRoutineMessage] = useState("");
  const skipMutation = useMutation({ mutationFn: skipRoutineOccurrence, onSuccess: async () => { setRoutineMessage("Occurrence skipped; the series remains unchanged."); await queryClient.invalidateQueries({ queryKey: ["routine-occurrences"] }); await queryClient.invalidateQueries({ queryKey: ["routines"] }); }, onError: () => setRoutineMessage("That occurrence could not be skipped. Reload the routine and try again.") });
  const rescheduleMutation = useMutation({ mutationFn: rescheduleRoutineOccurrence, onSuccess: async () => { setRoutineMessage("Occurrence moved 30 minutes later; the series remains unchanged."); await queryClient.invalidateQueries({ queryKey: ["routine-occurrences"] }); await queryClient.invalidateQueries({ queryKey: ["routines"] }); }, onError: () => setRoutineMessage("That occurrence could not be moved. Reload the routine and try again.") });
  const routineMutation = useMutation({ mutationFn: () => { const [hours = 9, minutes = 0] = routineStart.split(":").map(Number); return createRoutine({ title: routineTitle, kind: routineKind, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC", startDate: localDate(), endDate: "", weekdays: routineWeekdays, startMinute: hours * 60 + minutes, durationMinutes: Number(routineDuration), frequencyPerWeek: routineKind === "fixed" ? routineWeekdays.length : Math.min(2, routineWeekdays.length) }); }, onSuccess: async () => { setRoutineMessage("Routine saved; future occurrences are now visible in the accepted planning horizon."); setRoutineTitle(""); await queryClient.invalidateQueries({ queryKey: ["routines"] }); await queryClient.invalidateQueries({ queryKey: ["routine-occurrences"] }); }, onError: () => setRoutineMessage("Routine could not be saved. Check its time and eligible weekdays.") });
  const firstWorkItem = work.data?.[0];
  const placeNext = async () => {
    const item = work.data?.find((candidate) => candidate.id === (selectedWorkItem || firstWorkItem?.id));
    if (!item) return;
    setPlacing(true); setMessage("");
    try {
      const [hours = 9, minutes = 0] = startTime.split(":").map(Number);
      const requestedDuration = Number(duration);
      await createAllocation({ workItemId: item.id, localDate: localDate(), startMinutes: hours * 60 + minutes, durationMinutes: requestedDuration });
      await plan.refetch(); setMessage("Placed on today’s accepted schedule.");
    } catch { setMessage("That placement could not be accepted. Check for an overlap or try again."); } finally { setPlacing(false); }
  };
  return <section className="planner-surface plan-surface" data-testid={selectors.pages.plan} aria-labelledby="plan-heading">
    <p className="eyebrow">A day you can keep</p><h1 id="plan-heading">Plan</h1>
    <p className="planner-surface-description">Place focused work on an accepted schedule. Capacity below is measured from allocations you have explicitly placed.</p>
    {plan.isLoading && <p role="status">Calculating today’s plan…</p>}
    {plan.isError && <p role="alert">Today’s plan is unavailable right now.</p>}
    {plan.data && <>
      <div className="plan-view-switcher" role="tablist" aria-label="Planning view"><button type="button" role="tab" aria-selected={view === "day"} onClick={() => setView("day")}>Day</button><button type="button" role="tab" aria-selected={view === "week"} onClick={() => setView("week")}>Week</button><button type="button" role="tab" aria-selected={view === "agenda"} onClick={() => setView("agenda")}>Agenda</button></div>
      {view === "day" && <div className="plan-capacity-strip"><div><span className="card-kicker">PLANNED</span><strong>{Number(plan.data.plannedMinutes)} min</strong></div><div><span className="card-kicker">AVAILABLE</span><strong>{Number(plan.data.availableMinutes)} min</strong></div><div><span className="card-kicker">BREATHING ROOM</span><strong>{Number(plan.data.breathingRoomMinutes)} min</strong></div></div>}
      {plan.data.allocations.length === 0 && <div className="planner-empty"><p>No accepted allocations yet. Place the next work item when you are ready.</p></div>}
      {work.data && firstWorkItem && <div className="plan-placement">
        <div><label htmlFor="plan-work-item">Work item</label><select id="plan-work-item" value={selectedWorkItem || firstWorkItem.id} onChange={(event) => setSelectedWorkItem(event.target.value)}>{work.data.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</select></div>
        <div><label htmlFor="plan-start-time">Start</label><input id="plan-start-time" type="time" value={startTime} onChange={(event) => setStartTime(event.target.value)} /></div>
        <div><label htmlFor="plan-duration">Minutes</label><input id="plan-duration" type="number" min="1" max="720" step="5" value={duration} onChange={(event) => setDuration(event.target.value)} /></div>
        <button type="button" className="primary-action" onClick={() => void placeNext()} disabled={placing || work.isLoading || !duration}>{placing ? "Placing…" : "Accept placement"}</button>
      </div>}
      {view === "day" && plan.data.allocations.length > 0 && <div className="plan-timeline" aria-label="Today’s accepted schedule">{plan.data.allocations.map((entry) => <article className="plan-block" key={entry.id} style={{ marginLeft: `${Math.max(0, (Number(entry.startMinutes) - 540) / 600 * 100)}%`, width: `${Math.max(8, Number(entry.durationMinutes) / 600 * 100)}%` }}><span>{clock(Number(entry.startMinutes))} · {Number(entry.durationMinutes)} min</span><h2>{entry.title}</h2><small>{entry.sourceLabel || "Personal Planner"}</small></article>)}</div>}
      {view !== "day" && <RangeView view={view} query={rangeQuery.data?.allocations ?? []} loading={rangeQuery.isLoading} error={rangeQuery.isError} />}
      <section className="plan-routines" aria-labelledby="plan-routines-heading">
        <div><span className="card-kicker">RHYTHM</span><h2 id="plan-routines-heading">Routines that leave room</h2><p>Fixed appointments and flexible frequency are kept distinct. Occurrences are generated in local time; they do not silently become accepted work blocks.</p></div>
        {routines.data?.length ? <div className="routine-list">{routines.data.map((routine) => <article key={routine.id}><strong>{routine.title}</strong><span>{routine.kind} · {Number(routine.durationMinutes)} min · {routine.timezone}</span></article>)}</div> : <p className="planner-empty">No routines yet. Add a small rhythm only when it helps the week.</p>}
        {routineOccurrences.data?.length ? <div className="routine-occurrences" aria-label="Upcoming routine occurrences">{routineOccurrences.data.slice(0, 5).map((occurrence) => { const routine = routines.data?.find((candidate) => candidate.id === occurrence.routineId); return <div key={`${occurrence.routineId}-${occurrence.localDate}`}><span>{occurrence.localDate} · {Number(occurrence.startMinute) / 60 | 0}:{String(Number(occurrence.startMinute) % 60).padStart(2, "0")} · {Number(occurrence.durationMinutes)} min · {occurrence.title}</span><span className="routine-occurrence-actions"><button type="button" onClick={() => routine && rescheduleMutation.mutate({ routineId: occurrence.routineId, localDate: occurrence.localDate, startMinute: Math.min(1439, Number(occurrence.startMinute) + 30), expectedRevision: routine.revision })} disabled={rescheduleMutation.isPending || skipMutation.isPending || !routine}>Move +30m</button><button type="button" onClick={() => routine && skipMutation.mutate({ routineId: occurrence.routineId, localDate: occurrence.localDate, expectedRevision: routine.revision })} disabled={skipMutation.isPending || rescheduleMutation.isPending || !routine}>Skip once</button></span></div>; })}</div> : null}
        <form className="routine-form" onSubmit={(event) => { event.preventDefault(); routineMutation.mutate(); }}><label>Routine title<input value={routineTitle} onChange={(event) => setRoutineTitle(event.target.value)} required /></label><label>Type<select value={routineKind} onChange={(event) => setRoutineKind(event.target.value as "fixed" | "flexible")}><option value="fixed">Fixed recurrence</option><option value="flexible">Flexible frequency</option></select></label><label>Routine start<input aria-label="Routine start" type="time" value={routineStart} onChange={(event) => setRoutineStart(event.target.value)} /></label><label>Routine minutes<input aria-label="Routine minutes" type="number" min="1" max="480" value={routineDuration} onChange={(event) => setRoutineDuration(event.target.value)} /></label><div className="routine-days" aria-label="Eligible weekdays">{[1,2,3,4,5,6,7].map((day) => <label key={day}><input type="checkbox" checked={routineWeekdays.includes(day)} onChange={() => setRoutineWeekdays((days) => days.includes(day) ? days.filter((candidate) => candidate !== day) : [...days, day].sort())} />{["M","T","W","T","F","S","S"][day - 1]}</label>)}</div><button className="secondary-action" type="submit" disabled={routineMutation.isPending || !routineTitle || routineWeekdays.length === 0}>{routineMutation.isPending ? "Saving…" : "Add routine"}</button></form>
        {routineMessage && <p role="status" className="plan-message">{routineMessage}</p>}
      </section>
      {message && <p role="status" className="plan-message">{message}</p>}
      <p className="plan-boundary">Accepted placement is durable schedule state. Completing focus does not silently mark an allocation or work item complete.</p>
    </>}
  </section>;
}
