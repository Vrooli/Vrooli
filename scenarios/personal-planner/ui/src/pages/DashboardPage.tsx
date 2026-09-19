import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BarChart3, CalendarDays, FilePlus2, FileText, Play, Sun, Target, Waves } from "lucide-react";
import { selectors } from "../consts/selectors";
import { fetchTodayAllocations } from "../api/calendar";
import { fetchCurrentFocus, pauseFocus, startFocus } from "../api/focus";
import { createWorkItem, fetchWorkItems } from "../api/work";
import { useTheme } from "../theme/ThemeProvider";

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
  const { data: workItems, isLoading: workLoading, isError: workError } = useQuery({ queryKey: ["work-items"], queryFn: fetchWorkItems });
  const { data: todayPlan, isLoading: planLoading, isError: planError } = useQuery({ queryKey: ["today-allocations"], queryFn: () => fetchTodayAllocations() });
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
  const nextWork = workItems?.[0];
  const focusStarted = focusStartedOptimistically || focus.data?.state === "running";
  const taskTitle = nextWork?.title ?? "Capture your next useful action";
  const taskDescription = nextWork?.description || (workLoading ? "Loading your work…" : workError ? "Work data is unavailable right now." : "Nothing is scheduled yet.");
  const taskSource = nextWork?.sourceLabel || "Personal Planner";
  const taskMinutes = nextWork?.remainingMinutes ?? 0;
  const timeline = todayPlan?.allocations.map((entry, index) => ({
    label: entry.title,
    detail: `${entry.durationMinutes} min`,
    owner: entry.sourceLabel,
    startLabel: formatClock(entry.startMinutes),
    start: entry.startMinutes / 60,
    width: entry.durationMinutes / 60,
    tone: index === 0 ? "amber" : "lavender",
  })) ?? [];
  const plannedHours = todayPlan ? formatCapacity(Number(todayPlan.plannedMinutes)) : "—";
  const availableHours = todayPlan ? formatCapacity(Number(todayPlan.availableMinutes)) : "—";
  const breathingHours = todayPlan ? formatCapacity(Number(todayPlan.breathingRoomMinutes)) : "—";
  const appearance = choice === "system" ? "auto" : choice === "light" ? "day" : "night";
  const sceneryClass = typeof window !== "undefined" ? [window.localStorage.getItem("planner.art-free") === "true" && "art-free", window.localStorage.getItem("planner.reduced-scenery") === "true" && "reduced-scenery", window.localStorage.getItem("planner.subdued-night") === "true" && "subdued-night"].filter(Boolean).join(" ") : "";
  const setAppearance = (next: string) => setTheme(next === "auto" ? "system" : next === "day" ? "light" : "dark");

  return (
    <div className={`observatory appearance-${appearance} ${sceneryClass}`} data-testid={selectors.pages.today}>
      <div className="observatory-sky" aria-hidden="true" />
      <div className="observatory-content">
        <header className="observatory-header"><div><p className="eyebrow">Observatory · {today}</p><h1>Today</h1><p className="date-line">{today}</p></div><div className="appearance-wrap"><div className="appearance-toggle" role="group" aria-label="Appearance">{["auto", "day", "night"].map((option) => <button key={option} type="button" data-appearance={option} className={appearance === option ? "selected" : ""} onClick={() => setAppearance(option)}>{option.charAt(0).toUpperCase() + option.slice(1)}</button>)}</div><button type="button" className="capture-link" onClick={() => setCaptureOpen(true)}><FilePlus2 size={16} aria-hidden="true" />Capture task</button><span className="sample-label">Live plan · Work items</span></div></header>
        <section className="today-grid" aria-label="Current plan"><article className="next-card"><div className="card-kicker">{focusStarted ? "FOCUS IN PROGRESS" : nextWork ? "UP NEXT · READY TO PLACE" : "YOUR NEXT STEP"}<span>{taskMinutes ? `${taskMinutes} MIN` : "NO TIME SET"}</span></div><h2>{taskTitle}</h2><p className="task-source"><span className="source-dot violet" />{taskSource}</p><p className="task-description">{taskDescription}</p><div className="task-actions"><button type="button" className="primary-action" onClick={() => focusMutation.mutate()} disabled={focusMutation.isPending || focus.isLoading}><Play size={18} fill="currentColor" aria-hidden="true" />{focusMutation.isPending ? "Saving…" : focusStarted ? "Pause focus" : "Start focus"}</button>{nextWork ? <button type="button" className="quiet-action" onClick={() => setDraftOpen(true)}><FileText size={18} aria-hidden="true" />Open draft</button> : <button type="button" className="quiet-action" onClick={() => setCaptureOpen(true)}><FilePlus2 size={18} aria-hidden="true" />Capture task</button>}</div>{focusMutation.isError && <p role="alert" className="task-error">That focus transition did not save. Nothing was assumed.</p>}{draftOpen && nextWork && <div className="draft-panel" role="dialog" aria-label="Work item details"><div><strong>{taskTitle}</strong><button type="button" onClick={() => setDraftOpen(false)} aria-label="Close draft">×</button></div><p>{taskDescription}</p><small>{taskMinutes || "No"} minutes remaining · {taskSource}</small></div>}{captureOpen && <form className="capture-panel" role="dialog" aria-label="Capture task" onSubmit={(event) => { event.preventDefault(); if (captureTitle.trim()) captureMutation.mutate(); }}><div className="capture-panel-heading"><strong>Capture a useful next step</strong><button type="button" onClick={() => setCaptureOpen(false)} aria-label="Close capture">×</button></div><label>Task title<input autoFocus value={captureTitle} onChange={(event) => setCaptureTitle(event.target.value)} required /></label><label>Why it matters <span>(optional)</span><textarea value={captureDescription} onChange={(event) => setCaptureDescription(event.target.value)} rows={2} /></label><div className="capture-fields"><label>Minutes <span>(optional)</span><input type="number" min="0" max="1440" step="5" value={captureMinutes} onChange={(event) => setCaptureMinutes(event.target.value)} /></label><label>Source <span>(optional)</span><input value={captureSource} onChange={(event) => setCaptureSource(event.target.value)} /></label></div>{captureMutation.isError && <p role="alert">That task did not save. Nothing was assumed.</p>}<button type="submit" className="primary-action" disabled={captureMutation.isPending || !captureTitle.trim()}>{captureMutation.isPending ? "Saving…" : "Save task"}</button></form>}</article><div className="capacity-card"><p className="card-kicker">TODAY’S CAPACITY</p><div className="capacity-stats"><div><strong>{planLoading ? "…" : plannedHours}</strong><span>planned</span></div><div><strong>{planLoading ? "…" : availableHours}</strong><span>available</span></div><div><strong>{planLoading ? "…" : breathingHours}</strong><span>breathing room</span></div></div>{Number(todayPlan?.externalEventCount ?? 0) > 0 && <p className="external-event-note">{Number(todayPlan?.externalEventCount)} read-only calendar events occupy {Number(todayPlan?.externalBusyMinutes)} minutes. Accepted work and provider time are unioned, not double-counted.</p>}<div className="commitment"><p className="card-kicker">PLAN STATUS</p><div><span>{planError ? "Plan data unavailable" : todayPlan?.allocations.length ? `${todayPlan.allocations.length} work items placed` : "No accepted work placed yet"}</span><b aria-hidden="true">→</b><a href="/plan">Open plan&nbsp; →</a></div></div></div></section>
        <section className="day-plan" aria-labelledby="day-plan-heading"><div className="day-plan-heading"><h2 id="day-plan-heading">Your day</h2><span>09:00</span></div><div className="time-ruler" aria-hidden="true">{Array.from({ length: 11 }, (_, index) => <span key={index} className={index === 1 ? "now-label" : ""}>{`${9 + index}:00`}</span>)}</div><div className="timeline" role="list" aria-label="Timeline from 09:00 to 19:00"><div className="now-line" aria-hidden="true"><span /></div>{timeline.map((item) => <div key={item.label} role="listitem" className={`time-block ${item.tone}`} style={{ left: `${(item.start - 9) * 10}%`, width: `${item.width * 10}%` }}><span className="time-block-time">{item.startLabel}</span><strong>{item.label}</strong><span>{item.detail}</span>{item.owner && <small><span className="source-dot" />{item.owner}</small>}</div>)}</div>{!planLoading && !planError && timeline.length === 0 && <p className="timeline-empty">Nothing placed yet. Capture work to see a truthful day plan.</p>}</section>
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
