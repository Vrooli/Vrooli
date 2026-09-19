import { useState, type MouseEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BarChart3, CalendarDays, FilePlus2, FileText, Moon, Play, Sparkles, Sun, SunMoon, Target, Waves } from "lucide-react";
import { selectors } from "../consts/selectors";
import { fetchRoutineOccurrences, fetchTodayAllocations } from "../api/calendar";
import { fetchCurrentFocus, pauseFocus, startFocus } from "../api/focus";
import { createWorkItem, fetchWorkItems } from "../api/work";
import { useTheme } from "../theme/ThemeProvider";
import { useObservatoryAutoAppearance } from "../theme/observatoryAppearance";

/** The Observatory Today surface: a calm, honest read of the live work plan. */
export function DashboardPage() {
  const { choice, setTheme } = useTheme();
  const queryClient = useQueryClient();
  const [focusStartedOptimistically, setFocusStartedOptimistically] = useState(false);
  const [draftOpen, setDraftOpen] = useState(false);
  const [captureOpen, setCaptureOpen] = useState(false);
  const [captureTitle, setCaptureTitle] = useState("");
  const [captureDescription, setCaptureDescription] = useState("");
  const [captureMinutes, setCaptureMinutes] = useState("");
  const [captureSource, setCaptureSource] = useState("");
  const [timelineView, setTimelineView] = useState<"day" | "focus">("day");
  const [selectedTimelineItem, setSelectedTimelineItem] = useState<{ title: string; detail: string; source: string; startLabel: string; anchorLeft: number; anchorTop: number } | null>(null);
  const { data: workItems, isLoading: workLoading, isError: workError } = useQuery({ queryKey: ["work-items"], queryFn: fetchWorkItems });
  const { data: todayPlan, isLoading: planLoading, isError: planError } = useQuery({ queryKey: ["today-allocations"], queryFn: () => fetchTodayAllocations() });
  const todayLocalDate = localDate();
  const { data: routineOccurrences } = useQuery({ queryKey: ["routine-occurrences", todayLocalDate], queryFn: () => fetchRoutineOccurrences(todayLocalDate, todayLocalDate) });
  const focus = useQuery({ queryKey: ["focus-current"], queryFn: fetchCurrentFocus, refetchInterval: 30_000 });
  const focusMutation = useMutation({
    mutationFn: async () => {
      if (focus.data?.state === "running") return pauseFocus(focus.data);
      return startFocus({ workItemId: nextWork?.id, title: nextWork?.title ?? "Spontaneous focus", mode: "open" });
    },
    onSuccess: async (session) => {
      setFocusStartedOptimistically(session.state === "running");
      await queryClient.invalidateQueries({ queryKey: ["focus-current"] });
    },
  });
  const captureMutation = useMutation({
    mutationFn: () => createWorkItem({
      title: captureTitle,
      description: captureDescription,
      remainingMinutes: captureMinutes ? Number(captureMinutes) : 0,
      sourceLabel: captureSource,
    }),
    onSuccess: async () => {
      setCaptureOpen(false);
      setCaptureTitle("");
      setCaptureDescription("");
      setCaptureMinutes("");
      setCaptureSource("");
      await queryClient.invalidateQueries({ queryKey: ["work-items"] });
    },
  });
  const today = new Intl.DateTimeFormat(undefined, { weekday: "long", month: "long", day: "numeric" }).format(new Date());
  const nextAllocation = todayPlan?.allocations[0];
  const nextWork = workItems?.find((item) => item.id === nextAllocation?.workItemId) ?? workItems?.[0];
  const hasAcceptedNext = Boolean(nextAllocation && nextWork?.id === nextAllocation.workItemId);
  const focusStarted = focusStartedOptimistically || focus.data?.state === "running";
  const autoAppearance = useObservatoryAutoAppearance(focusStarted);
  const taskTitle = nextAllocation?.title ?? nextWork?.title ?? "Capture your next useful action";
  const taskDescription = hasAcceptedNext ? `Accepted for ${formatClock(nextAllocation!.startMinutes)} · ${nextAllocation!.durationMinutes} minutes` : nextWork?.description || (workLoading ? "Loading your work…" : workError ? "Work data is unavailable right now." : "Nothing is scheduled yet.");
  const taskSource = nextAllocation?.sourceLabel || nextWork?.sourceLabel || "Personal Planner";
  const taskMinutes = nextAllocation?.durationMinutes ?? nextWork?.remainingMinutes ?? 0;
  const timeline = todayPlan?.allocations.reduce<Array<{
    id: string; label: string; detail: string; owner: string; startLabel: string;
    start: number; width: number; tone: string; end: number; lane: number;
  }>>((items, entry, index) => {
    const item = {
      id: `${entry.workItemId}-${entry.startMinutes}-${index}`,
      label: entry.title,
      detail: `${entry.durationMinutes} min`,
      owner: entry.sourceLabel,
      startLabel: formatClock(entry.startMinutes),
      start: entry.startMinutes / 60,
      width: entry.durationMinutes / 60,
      tone: index === 0 ? "amber" : "lavender",
      end: entry.startMinutes / 60 + entry.durationMinutes / 60,
    };
    return [...items, { ...item, lane: findTimelineLane(item, items) }];
  }, []) ?? [];
  const timelineLanes = timeline.length ? Math.max(...timeline.map((item) => item.lane)) + 1 : 1;
  const timelineStartHour = timelineView === "focus" ? Math.max(9, Math.min(15, Math.floor((timeline[0]?.start ?? 10) - 1))) : 9;
  const timelineEndHour = timelineView === "focus" ? Math.min(19, timelineStartHour + 4) : 19;
  const timelineSpan = timelineEndHour - timelineStartHour;
  const plannedHours = todayPlan ? formatCapacity(Number(todayPlan.plannedMinutes)) : "—";
  const availableHours = todayPlan ? formatCapacity(Number(todayPlan.availableMinutes)) : "—";
  const breathingHours = todayPlan ? formatCapacity(Number(todayPlan.breathingRoomMinutes)) : "—";
  const routineMinutes = routineOccurrences?.reduce((total, occurrence) => total + Number(occurrence.durationMinutes), 0) ?? 0;
  const requestedAppearance = typeof window !== "undefined" ? new URLSearchParams(window.location.search).get("appearance") : null;
  const appearanceChoice = requestedAppearance === "auto" || requestedAppearance === "day" || requestedAppearance === "night"
    ? requestedAppearance
    : choice === "system" ? "auto" : choice === "light" ? "day" : "night";
  const appearance = appearanceChoice === "auto" ? autoAppearance : appearanceChoice;
  const sceneryClass = typeof window !== "undefined" ? [window.localStorage.getItem("planner.art-free") === "true" && "art-free", window.localStorage.getItem("planner.reduced-scenery") === "true" && "reduced-scenery", window.localStorage.getItem("planner.subdued-night") === "true" && "subdued-night"].filter(Boolean).join(" ") : "";
  const setAppearance = (next: string) => setTheme(next === "auto" ? "system" : next === "day" ? "light" : "dark");
  const openTimelineItem = (element: HTMLElement, item: (typeof timeline)[number]) => {
    const timelineElement = element.closest(".timeline");
    const timelineRect = timelineElement?.getBoundingClientRect();
    const itemRect = element.getBoundingClientRect();
    const anchorLeft = timelineRect ? Math.min(Math.max(0, itemRect.left - timelineRect.left), Math.max(0, timelineRect.width - 320)) : 0;
    const anchorTop = timelineRect ? itemRect.bottom - timelineRect.top + 10 : 0;
    setSelectedTimelineItem({ title: item.label, detail: item.detail, source: item.owner, startLabel: item.startLabel, anchorLeft, anchorTop });
  };

  return (
    <div className={`observatory appearance-${appearance} appearance-choice-${appearanceChoice} ${sceneryClass}`} data-testid={selectors.pages.today}>
      <div className="observatory-sky" aria-hidden="true"><span className="observatory-sky-layer observatory-sky-day" /><span className="observatory-sky-layer observatory-sky-night" /><span className="observatory-sky-stars" /></div>
      <div className="observatory-content">
        <header className="observatory-header"><div><p className="eyebrow">Observatory · {today}</p><h1>Today</h1><p className="date-line">{today}</p></div><div className="appearance-wrap"><div className="appearance-toggle" role="group" aria-label="Appearance">{([{ key: "auto", label: "Auto", icon: <SunMoon size={16} aria-hidden="true" /> }, { key: "day", label: "Day", icon: <Sun size={16} aria-hidden="true" /> }, { key: "night", label: "Night", icon: <Moon size={16} aria-hidden="true" /> }] as const).map((option) => <button key={option.key} type="button" data-appearance={option.key} aria-label={option.key === "day" ? "Day appearance" : option.label} title={option.label} className={appearanceChoice === option.key ? "selected" : ""} onClick={() => setAppearance(option.key)}>{option.icon}<span>{option.label}</span></button>)}</div><button type="button" className="capture-link" onClick={() => setCaptureOpen(true)}><FilePlus2 size={16} aria-hidden="true" />Capture task</button><span className="sample-label">Live plan · Work items</span></div></header>
        <section className="today-grid" aria-label="Current plan"><article className="next-card"><div className="card-kicker">{focusStarted ? "FOCUS IN PROGRESS" : hasAcceptedNext ? "UP NEXT · ACCEPTED" : nextWork ? "UP NEXT · READY TO PLACE" : "YOUR NEXT STEP"}<span>{taskMinutes ? `${taskMinutes} MIN` : "NO TIME SET"}</span></div><h2>{taskTitle}</h2><p className="task-source"><span className="source-dot violet" />{taskSource}</p><p className="task-description">{taskDescription}</p><div className="task-actions"><button type="button" className="primary-action" onClick={() => focusMutation.mutate()} disabled={focusMutation.isPending || focus.isLoading}><Play size={18} fill="currentColor" aria-hidden="true" />{focusMutation.isPending ? "Saving…" : focusStarted ? "Pause focus" : "Start focus"}</button>{nextWork ? <button type="button" className="quiet-action" onClick={() => setDraftOpen(true)}><FileText size={18} aria-hidden="true" />Open draft</button> : <button type="button" className="quiet-action" onClick={() => setCaptureOpen(true)}><FilePlus2 size={18} aria-hidden="true" />Capture task</button>}</div>{focusMutation.isError && <p role="alert" className="task-error">That focus transition did not save. Nothing was assumed.</p>}{draftOpen && nextWork && <div className="draft-panel" role="dialog" aria-label="Work item details"><div><strong>{taskTitle}</strong><button type="button" onClick={() => setDraftOpen(false)} aria-label="Close draft">×</button></div><p>{taskDescription}</p><small>{taskMinutes || "No"} minutes remaining · {taskSource}</small></div>}{captureOpen && <form className="capture-panel" role="dialog" aria-label="Capture task" onSubmit={(event) => { event.preventDefault(); if (captureTitle.trim()) captureMutation.mutate(); }}><div className="capture-panel-heading"><strong>Capture a useful next step</strong><button type="button" onClick={() => setCaptureOpen(false)} aria-label="Close capture">×</button></div><label>Task title<input autoFocus value={captureTitle} onChange={(event) => setCaptureTitle(event.target.value)} required /></label><label>Why it matters <span>(optional)</span><textarea value={captureDescription} onChange={(event) => setCaptureDescription(event.target.value)} rows={2} /></label><div className="capture-fields"><label>Minutes <span>(optional)</span><input type="number" min="0" max="1440" step="5" value={captureMinutes} onChange={(event) => setCaptureMinutes(event.target.value)} /></label><label>Source <span>(optional)</span><input value={captureSource} onChange={(event) => setCaptureSource(event.target.value)} /></label></div>{captureMutation.isError && <p role="alert">That task did not save. Nothing was assumed.</p>}<button type="submit" className="primary-action" disabled={captureMutation.isPending || !captureTitle.trim()}>{captureMutation.isPending ? "Saving…" : "Save task"}</button></form>}</article><div className="capacity-card"><p className="card-kicker">TODAY’S CAPACITY</p><div className="capacity-stats"><div><strong>{planLoading ? "…" : plannedHours}</strong><span>planned</span></div><div><strong>{planLoading ? "…" : availableHours}</strong><span>available</span></div><div><strong>{planLoading ? "…" : breathingHours}</strong><span>breathing room</span></div></div>{Number(todayPlan?.externalEventCount ?? 0) > 0 && <p className="external-event-note" aria-label={`${Number(todayPlan?.externalEventCount)} read-only calendar events occupy ${Number(todayPlan?.externalBusyMinutes)} minutes. Accepted work and provider time are unioned, not double-counted.`}>{Number(todayPlan?.externalEventCount)} calendar holds · {Number(todayPlan?.externalBusyMinutes)} min reserved</p>}{routineMinutes > 0 && <p className="external-event-note routine-demand-note" aria-label={`${routineOccurrences?.length ?? 0} routine commitments occupy ${routineMinutes} minutes.`}>{routineOccurrences?.length ?? 0} routine commitments · {routineMinutes} min reserved</p>}<div className="commitment"><p className="card-kicker">PLAN STATUS</p><div><span>{planError ? "Plan data unavailable" : todayPlan?.allocations.length ? `${todayPlan.allocations.length} work items placed` : "No accepted work placed yet"}</span><b aria-hidden="true">→</b><a href="/plan">Open plan&nbsp; →</a></div></div></div></section>
        <section className="day-plan" aria-labelledby="day-plan-heading"><div className="day-plan-heading"><div><h2 id="day-plan-heading">Your day</h2><span>{timelineView === "focus" ? `${formatClock(timelineStartHour * 60)}–${formatClock(timelineEndHour * 60)}` : "09:00–19:00"}</span></div><div className="timeline-view-toggle" role="group" aria-label="Timeline view"><button type="button" className={timelineView === "day" ? "selected" : ""} onClick={() => setTimelineView("day")}><CalendarDays size={14} aria-hidden="true" />Day</button><button type="button" className={timelineView === "focus" ? "selected" : ""} onClick={() => setTimelineView("focus")}><Sparkles size={14} aria-hidden="true" />Focus</button></div></div><div className="time-ruler" aria-hidden="true">{Array.from({ length: timelineSpan + 1 }, (_, index) => <span key={index} className={index === Math.round(((new Date().getHours() + new Date().getMinutes() / 60) - timelineStartHour) * 2) / 2 ? "now-label" : ""}>{`${timelineStartHour + index}:00`}</span>)}</div><div className="timeline" role="list" aria-label={`Timeline from ${formatClock(timelineStartHour * 60)} to ${formatClock(timelineEndHour * 60)}`} style={{ minHeight: `${Math.max(9.5, timelineLanes * 6.2)}rem` }}><div className="now-line" aria-hidden="true" style={{ left: `${Math.max(0, Math.min(100, ((new Date().getHours() + new Date().getMinutes() / 60 - timelineStartHour) / timelineSpan) * 100))}%` }}><span /></div>{timeline.map((item) => <div key={item.id} role="button" tabIndex={0} aria-label={`${item.label}, ${item.startLabel}, ${item.detail}${item.owner ? `, ${item.owner}` : ""}`} data-full-title={item.label} className={`time-block ${item.tone}`} style={{ left: `${((item.start - timelineStartHour) / timelineSpan) * 100}%`, width: `${(item.width / timelineSpan) * 100}%`, top: `calc(2rem + ${item.lane} * 6.2rem)` }} onClick={(event: MouseEvent<HTMLDivElement>) => openTimelineItem(event.currentTarget, item)} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); openTimelineItem(event.currentTarget, item); } }}><span className="time-block-time">{item.startLabel}</span><strong>{item.label}</strong><span>{item.detail}</span>{item.owner && <small><span className="source-dot" />{item.owner}</small>}</div>)}</div>{selectedTimelineItem && <div className="timeline-detail" role="dialog" aria-label="Timeline item details" style={{ left: selectedTimelineItem.anchorLeft, top: selectedTimelineItem.anchorTop }}><div><strong>{selectedTimelineItem.title}</strong><button type="button" onClick={() => setSelectedTimelineItem(null)} aria-label="Close timeline item">×</button></div><p>{selectedTimelineItem.startLabel} · {selectedTimelineItem.detail}</p>{selectedTimelineItem.source && <small>{selectedTimelineItem.source}</small>}</div>}{!planLoading && !planError && timeline.length === 0 && <p className="timeline-empty">Nothing placed yet. Capture work to see a truthful day plan.</p>}</section>
        <footer className="observatory-note"><span aria-hidden="true">—</span><em>A calmer<br />tomorrow lives here.</em><span className="footer-icons" aria-hidden="true"><Sun size={18} /><CalendarDays size={18} /><Target size={18} /><Waves size={18} /><BarChart3 size={18} /></span></footer>
      </div>
    </div>
  );
}

function formatCapacity(minutes: number): string {
  if (minutes < 60) return `${minutes} min`;
  return `${Math.floor(minutes / 60)} h`;
}

function formatClock(minutes: number): string {
  return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`;
}

function localDate(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
}

function findTimelineLane(item: { start: number; end: number }, previous: Array<{ start: number; end: number; lane?: number }>): number {
  let lane = 0;
  while (previous.some((other) => other.lane === lane && item.start < other.end && item.end > other.start)) lane += 1;
  return lane;
}
