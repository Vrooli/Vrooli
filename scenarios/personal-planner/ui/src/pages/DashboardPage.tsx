import { useEffect, useState, type CSSProperties } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BarChart3, CalendarDays, Sun, Target, Waves } from "lucide-react";
import { selectors } from "../consts/selectors";
import { fetchRoutineOccurrences, fetchTodayAllocations } from "../api/calendar";
import { fetchCurrentFocus, pauseFocus, startFocus } from "../api/focus";
import { createWorkItem, fetchWorkItems } from "../api/work";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { OBSERVATORY_SCENERY_EVENT, readSceneryPreferences, useObservatoryAutoAppearance, useSampledSceneColors } from "../theme/observatoryAppearance";
import { useChromeContribution } from "@vrooli/react-component-library/ChromeTheme/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { TodayAppearanceControl, TodayCaptureForm, TodayCaptureLink, TodayTaskActions, TodayTimelineBlock, TodayTimelineViewToggle, type TodayTimelineItem } from "../components/TodayControls";
import { PlannerDialog as Dialog } from "../components/PlannerDialog";

type TimelineItem = TodayTimelineItem;

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
  const [selectedTimelineItem, setSelectedTimelineItem] = useState<TimelineItem | null>(null);
  const [sceneryPreferences, setSceneryPreferences] = useState(() => readSceneryPreferences());
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
    onError: () => setCaptureOpen(true),
  });
  const today = new Intl.DateTimeFormat(undefined, { weekday: "long", month: "long", day: "numeric" }).format(new Date());
  const nextAllocation = todayPlan?.allocations[0];
  const nextWork = workItems?.find((item) => item.id === nextAllocation?.workItemId) ?? workItems?.[0];
  const hasAcceptedNext = Boolean(nextAllocation && nextWork?.id === nextAllocation.workItemId);
  const focusStarted = focusStartedOptimistically || focus.data?.state === "running";
  const autoAppearance = useObservatoryAutoAppearance();
  const sceneColors = useSampledSceneColors();
  const taskTitle = nextAllocation?.title ?? nextWork?.title ?? "Capture your next useful action";
  const taskDescription = hasAcceptedNext ? `Accepted for ${formatClock(nextAllocation!.startMinutes)} · ${nextAllocation!.durationMinutes} minutes` : nextWork?.description || (workLoading ? "Loading your work…" : workError ? "Work data is unavailable right now." : "Nothing is scheduled yet.");
  const taskSource = nextAllocation?.sourceLabel || nextWork?.sourceLabel || "Personal Planner";
  const taskMinutes = nextAllocation?.durationMinutes ?? nextWork?.remainingMinutes ?? 0;
  const timeline = todayPlan?.allocations.slice().sort((left, right) => left.startMinutes - right.startMinutes || right.durationMinutes - left.durationMinutes).reduce<TimelineItem[]>((items, entry, index) => {
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
  useEffect(() => {
    const refresh = () => setSceneryPreferences(readSceneryPreferences());
    window.addEventListener(OBSERVATORY_SCENERY_EVENT, refresh);
    window.addEventListener("storage", refresh);
    return () => {
      window.removeEventListener(OBSERVATORY_SCENERY_EVENT, refresh);
      window.removeEventListener("storage", refresh);
    };
  }, []);
  const appearanceChoice = choice;
  const appearance = appearanceChoice === "auto" ? autoAppearance : appearanceChoice;
  const chromeColor = typeof document !== "undefined"
    ? getComputedStyle(document.documentElement).getPropertyValue(appearance === "night" ? "--color-background" : "--color-surface-muted").trim() || (appearance === "night" ? "rgb(23 22 52)" : "rgb(235 229 216)")
    : appearance === "night" ? "rgb(23 22 52)" : "rgb(235 229 216)";
  useChromeContribution(
    { statusColor: chromeColor, fillColor: chromeColor },
    { key: "personal-planner-observatory", priority: 10 },
  );
  const sceneryClass = [sceneryPreferences.artFree && "art-free", sceneryPreferences.reducedScenery && "reduced-scenery", sceneryPreferences.subduedNight && "subdued-night"].filter(Boolean).join(" ");
  const sceneStyle = {
    ...(sceneColors.day ? { "--scene-sky-seam-day": sceneColors.day } : {}),
    ...(sceneColors.night ? { "--scene-sky-seam-night": sceneColors.night } : {}),
    ...(sceneColors.aspect ? { "--scene-band-aspect": String(sceneColors.aspect) } : {}),
  } as CSSProperties;
  const setAppearance = (next: string) => setTheme(next as ThemeChoice);
  const openTimelineItem = (item: TimelineItem) => setSelectedTimelineItem(item);

  return (
    <div className={`observatory appearance-${appearance} appearance-choice-${appearanceChoice} ${sceneryClass}`} data-testid={selectors.pages.today} style={sceneStyle}>
      <div className="observatory-sky" aria-hidden="true"><span className="observatory-sky-layer observatory-sky-day" /><span className="observatory-sky-layer observatory-sky-night" /><span className="observatory-sky-stars" /></div>
      <div className="observatory-content">
        <header className="observatory-header"><div><p className="eyebrow">Observatory · {today}</p><h1>Today</h1><p className="date-line">{today}</p></div><div className="appearance-wrap"><TodayAppearanceControl choice={appearanceChoice} onChange={setAppearance} /><TodayCaptureLink onClick={() => setCaptureOpen(true)} /><span className="sample-label">Live plan · Work items</span></div></header>
        <section className="today-grid" aria-label="Current plan"><article className="next-card"><div className="card-kicker">{focusStarted ? "FOCUS IN PROGRESS" : hasAcceptedNext ? "UP NEXT · ACCEPTED" : nextWork ? "UP NEXT · READY TO PLACE" : "YOUR NEXT STEP"}<span>{taskMinutes ? `${taskMinutes} MIN` : "NO TIME SET"}</span></div><h2>{taskTitle}</h2><p className="task-source"><span className="source-dot violet" />{taskSource}</p><p className="task-description">{taskDescription}</p><TodayTaskActions focusStarted={focusStarted} pending={focusMutation.isPending || focus.isLoading} hasWork={Boolean(nextWork)} onFocus={() => focusMutation.mutate()} onOpenDraft={() => setDraftOpen(true)} onCapture={() => setCaptureOpen(true)} />{focusMutation.isError && <p role="alert" className="task-error">That focus transition did not save. Nothing was assumed.</p>}{nextWork && <Dialog open={draftOpen} title="Work item details" onClose={() => setDraftOpen(false)} closeLabel="Close draft" contentClassName="today-dialog-copy"><p>{taskDescription}</p><small>{taskMinutes || "No"} minutes remaining · {taskSource}</small></Dialog>}<Dialog open={captureOpen} title="Capture task" description="Capture a useful next step without changing the accepted schedule." onClose={() => setCaptureOpen(false)} closeLabel="Close capture" contentClassName="capture-dialog-body"><TodayCaptureForm title={captureTitle} description={captureDescription} minutes={captureMinutes} source={captureSource} pending={captureMutation.isPending} error={captureMutation.isError} onTitleChange={setCaptureTitle} onDescriptionChange={setCaptureDescription} onMinutesChange={setCaptureMinutes} onSourceChange={setCaptureSource} onSubmit={(event) => { event.preventDefault(); if (captureTitle.trim()) captureMutation.mutate(); }} /></Dialog></article><div className="capacity-card"><p className="card-kicker">TODAY’S CAPACITY</p><div className="capacity-stats"><div><strong>{planLoading ? "…" : plannedHours}</strong><span>planned</span></div><div><strong>{planLoading ? "…" : availableHours}</strong><span>available</span></div><div><strong>{planLoading ? "…" : breathingHours}</strong><span>breathing room</span></div></div>{Number(todayPlan?.externalEventCount ?? 0) > 0 && <p className="external-event-note" aria-label={`${Number(todayPlan?.externalEventCount)} read-only calendar events occupy ${Number(todayPlan?.externalBusyMinutes)} minutes. Accepted work and provider time are unioned, not double-counted.`}>{Number(todayPlan?.externalEventCount)} calendar holds · {Number(todayPlan?.externalBusyMinutes)} min reserved</p>}{routineMinutes > 0 && <p className="external-event-note routine-demand-note" aria-label={`${routineOccurrences?.length ?? 0} routine commitments occupy ${routineMinutes} minutes.`}>{routineOccurrences?.length ?? 0} routine commitments · {routineMinutes} min reserved</p>}<div className="commitment"><p className="card-kicker">PLAN STATUS</p><div><span>{planError ? "Plan data unavailable" : todayPlan?.allocations.length ? `${todayPlan.allocations.length} work items placed` : "No accepted work placed yet"}</span><b aria-hidden="true">→</b><a href="/plan">Open plan&nbsp; →</a></div></div></div></section>
        <section className="day-plan" aria-labelledby="day-plan-heading"><div className="day-plan-heading"><div><h2 id="day-plan-heading">Your day</h2><span>{timelineView === "focus" ? `${formatClock(timelineStartHour * 60)}–${formatClock(timelineEndHour * 60)}` : "09:00–19:00"}</span></div><TodayTimelineViewToggle view={timelineView} onChange={setTimelineView} /></div><div className="time-ruler" aria-hidden="true" style={{ gridTemplateColumns: `repeat(${timelineSpan + 1}, minmax(0, 1fr))` }}>{Array.from({ length: timelineSpan + 1 }, (_, index) => <span key={index} className={index === Math.round(((new Date().getHours() + new Date().getMinutes() / 60) - timelineStartHour) * 2) / 2 ? "now-label" : ""}>{`${timelineStartHour + index}:00`}</span>)}</div><div className="timeline" role="list" aria-label={`Timeline from ${formatClock(timelineStartHour * 60)} to ${formatClock(timelineEndHour * 60)}`} style={{ minHeight: `${Math.max(9.5, timelineLanes * 6.2)}rem`, "--timeline-step": `${100 / timelineSpan}%` } as CSSProperties}><div className="now-line" aria-hidden="true" style={{ left: `${Math.max(0, Math.min(100, ((new Date().getHours() + new Date().getMinutes() / 60 - timelineStartHour) / timelineSpan) * 100))}%` }}><span /></div>{timeline.map((item) => <TodayTimelineBlock key={item.id} item={item} startHour={timelineStartHour} span={timelineSpan} open={selectedTimelineItem?.id === item.id} onOpen={() => openTimelineItem(item)} onClose={() => setSelectedTimelineItem(null)} />)}</div>{!planLoading && !planError && timeline.length === 0 && <EmptyState className="timeline-empty-state" title="Nothing placed yet. Capture work to see a truthful day plan." />}</section>
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
