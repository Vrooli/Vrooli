import { useEffect, useMemo, useRef, useState } from "react";
import { Timer } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { AdaptivePageHeader } from "../components/AdaptivePageHeader";
import { Select } from "@vrooli/react-component-library/Select/1";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import { ObservatoryScene } from "../components/ObservatoryScene";
import { PlannerDialog as Dialog } from "../components/PlannerDialog";
import { useBreakpoint } from "../hooks/useBreakpoint";
import { DesktopFocusStartGuidance, MobileFocusStartGuidance } from "../components/FocusStartGuidance";

import { correctActual, endFocus, fetchActualCorrections, fetchActuals, fetchCurrentFocus, pauseFocus, recordManualActual, recordPauseReason, resumeFocus, saveSessionNote, startFocus, type Actual, type FocusSession, type PauseEvent } from "../api/focus";
import { fetchWorkItems } from "../api/work";
import { selectors } from "../consts/selectors";

function formatDuration(seconds: number): string {
  const minutes = Math.floor(Math.max(0, seconds) / 60);
  const remainder = Math.max(0, seconds) % 60;
  return `${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}

function activeSeconds(session: FocusSession, now: number): number {
  if (session.state !== "running") return Number(session.activeSeconds);
  return Number(session.activeSeconds) + Math.max(0, now - Number(session.activeStartedAtUnixSeconds));
}

export function FocusPage() {
  const queryClient = useQueryClient();
  const { isMobile } = useBreakpoint();
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  const current = useQuery({ queryKey: ["focus-current"], queryFn: fetchCurrentFocus, refetchInterval: 30_000 });
  const work = useQuery({ queryKey: ["work-items"], queryFn: fetchWorkItems });
  const actualDate = todayLocalDate();
  const actuals = useQuery({ queryKey: ["focus-actuals", actualDate], queryFn: () => fetchActuals(actualDate) });
  const corrections = useQuery({ queryKey: ["focus-corrections", actualDate], queryFn: () => fetchActualCorrections({ localDate: actualDate }) });
  const [selectedWorkId, setSelectedWorkId] = useState("");
  const [actualTitle, setActualTitle] = useState("");
  const [actualMinutes, setActualMinutes] = useState("");
  const [actualNote, setActualNote] = useState("");
  const [editingActual, setEditingActual] = useState<Actual | null>(null);
  const [correctionMinutes, setCorrectionMinutes] = useState("");
  const [correctionNote, setCorrectionNote] = useState("");
  const [actualsOpen, setActualsOpen] = useState(false);
  const [endedSession, setEndedSession] = useState<FocusSession | null>(null);
  const [sessionNote, setSessionNote] = useState("");
  const [timerMode, setTimerMode] = useState<"countdown" | "open" | "pomodoro">("countdown");
  const [plannedMinutes, setPlannedMinutes] = useState(45);
  const [pomodoroCycles, setPomodoroCycles] = useState(4);
  const [pomodoroCycle, setPomodoroCycle] = useState(1);
  const [breakUntil, setBreakUntil] = useState<number | null>(null);
  const [pauseReason, setPauseReason] = useState<PauseEvent["reason"]>("interrupted");
  const announcedOvertime = useRef(false);
  const selected = work.data?.find((item) => item.id === selectedWorkId) ?? work.data?.[0];
  const session = current.data;

  useEffect(() => {
    if ((!session || session.state !== "running") && !breakUntil) return;
    const timer = window.setInterval(() => setNow(Math.floor(Date.now() / 1000)), 1000);
    return () => window.clearInterval(timer);
  }, [session, breakUntil]);
  useEffect(() => {
    if (selected?.remainingMinutes && !session) setPlannedMinutes(Math.max(1, Number(selected.remainingMinutes)));
  }, [selected?.id, selected?.remainingMinutes, session]);

  const mutation = useMutation({
    mutationFn: async (action: "start" | "pause" | "resume" | "end" | "break") => {
      if (action === "start") {
        const minutes = timerMode === "open" ? 0 : plannedMinutes;
        if (timerMode !== "open") window.localStorage.setItem(`personal-planner.focus-target:${selected?.id ?? "open"}`, String(minutes * 60));
        return startFocus({ workItemId: selected?.id, title: selected?.title ?? "Spontaneous focus", mode: timerMode === "open" ? "open" : timerMode });
      }
      if (!session) throw new Error("No current focus session");
      if (action === "pause") {
        const paused = await pauseFocus(session);
        await recordPauseReason({ sessionId: session.id, reason: pauseReason });
        return paused;
      }
      if (action === "resume") {
        if (breakUntil) window.localStorage.setItem(`personal-planner.focus-target:${session.workItemId || "open"}`, String(Number(session.activeSeconds) + plannedMinutes * 60));
        return resumeFocus(session);
      }
      if (action === "break") return pauseFocus(session);
      return endFocus(session);
    },
    onSuccess: async (result, action) => { if (action === "end") setEndedSession(result); if (action === "break") { setBreakUntil(Math.floor(Date.now() / 1000) + 5 * 60); setPomodoroCycle((cycle) => Math.min(pomodoroCycles, cycle + 1)); } if (action === "resume") setBreakUntil(null); await queryClient.invalidateQueries({ queryKey: ["focus-current"] }); },
  });
  const noteMutation = useMutation({ mutationFn: () => { if (!endedSession) throw new Error("No ended session"); return saveSessionNote({ sessionId: endedSession.id, note: sessionNote }); }, onSuccess: () => setSessionNote("") });
  const actualMutation = useMutation({
    mutationFn: () => recordManualActual({ title: actualTitle, localDate: actualDate, reportedMinutes: Number(actualMinutes), workItemId: selected?.id, note: actualNote }),
    onSuccess: async () => { setActualTitle(""); setActualMinutes(""); setActualNote(""); await queryClient.invalidateQueries({ queryKey: ["focus-actuals", actualDate] }); },
  });
  const correctionMutation = useMutation({
    mutationFn: () => {
      if (!editingActual) throw new Error("No actual selected");
      return correctActual(editingActual, { reportedMinutes: Number(correctionMinutes), note: correctionNote });
    },
    onSuccess: async () => { setEditingActual(null); setCorrectionMinutes(""); setCorrectionNote(""); await queryClient.invalidateQueries({ queryKey: ["focus-actuals", actualDate] }); await queryClient.invalidateQueries({ queryKey: ["focus-corrections", actualDate] }); },
  });

  const elapsed = useMemo(() => session ? activeSeconds(session, now) : 0, [session, now]);
  const targetSeconds = session && (session.mode === "countdown" || session.mode === "pomodoro") ? Number(window.localStorage.getItem(`personal-planner.focus-target:${session.workItemId || "open"}`) ?? 0) : 0;
  const overtime = targetSeconds > 0 && elapsed > targetSeconds;
  const timerSeconds = targetSeconds > 0 ? Math.abs(targetSeconds - elapsed) : elapsed;
  const breakRemaining = breakUntil ? Math.max(0, breakUntil - now) : 0;

  useEffect(() => {
    if (!overtime) {
      announcedOvertime.current = false;
      return;
    }
    if (announcedOvertime.current || typeof window === "undefined") return;
    announcedOvertime.current = true;
    const AudioContextCtor = window.AudioContext ?? (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!AudioContextCtor) return;
    try {
      const audio = new AudioContextCtor();
      const oscillator = audio.createOscillator();
      const gain = audio.createGain();
      oscillator.frequency.value = 660;
      gain.gain.setValueAtTime(0.0001, audio.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.08, audio.currentTime + 0.02);
      gain.gain.exponentialRampToValueAtTime(0.0001, audio.currentTime + 0.22);
      oscillator.connect(gain).connect(audio.destination);
      oscillator.start();
      oscillator.stop(audio.currentTime + 0.24);
      oscillator.addEventListener("ended", () => void audio.close(), { once: true });
    } catch {
      // The visual/textual overtime state remains authoritative on locked-down
      // browsers and environments without an audio output.
    }
  }, [overtime]);

  return (
    <ObservatoryScene kind="focus" className={session ? "focus-active" : "focus-standby"}><section className="focus-surface" data-testid={selectors.pages.focus} aria-labelledby="focus-heading">
      <AdaptivePageHeader className="planner-page-header" headingId="focus-heading" eyebrow="Deep work" title="Focus" description="One protected window, with the time recorded honestly." leading={<span className="planner-page-mark" aria-hidden="true"><Timer size={21} /></span>} />
      {current.isLoading && <p role="status">Checking for an active session…</p>}
      {current.isError && <p role="alert">The focus session is unavailable right now.</p>}
      {!current.isLoading && !current.isError && !session && (
        <div className="focus-start-card">
          <span className="card-kicker">READY WHEN YOU ARE</span>
          <h2>{selected?.title ?? "Spontaneous focus"}</h2>
          <p>{selected ? `${selected.remainingMinutes} minutes remain on this work item.` : "Start without a task and classify the time later."}</p>
          {isMobile ? <MobileFocusStartGuidance /> : <DesktopFocusStartGuidance />}
          <div className="focus-mode-row" role="group" aria-label="Timer mode"><Button type="button" variant={timerMode === "countdown" ? "primary" : "secondary"} onClick={() => setTimerMode("countdown")}>Countdown</Button><Button type="button" variant={timerMode === "pomodoro" ? "primary" : "secondary"} onClick={() => setTimerMode("pomodoro")}>Pomodoro</Button><Button type="button" variant={timerMode === "open" ? "primary" : "secondary"} onClick={() => setTimerMode("open")}>Open timer</Button>{timerMode !== "open" && <Select aria-label="Planned duration" value={String(plannedMinutes)} onChange={(event) => setPlannedMinutes(Number(event.target.value))} options={[25, 45, 90].map((minutes) => ({ value: String(minutes), label: `${minutes} minutes` }))} />}{timerMode === "pomodoro" && <Select aria-label="Pomodoro cycles" value={String(pomodoroCycles)} onChange={(event) => setPomodoroCycles(Number(event.target.value))} options={[2, 4, 6].map((cycles) => ({ value: String(cycles), label: `${cycles} cycles` }))} />}</div>
          {work.data && work.data.length > 0 && <FormField required label="Work item" control={<Select required aria-label="Work item" value={selectedWorkId || work.data[0]?.id || ""} onChange={(event) => setSelectedWorkId(event.target.value)} options={work.data.map((item) => ({ value: item.id, label: item.title }))} />} />}
          <Button className="primary-action" type="button" onClick={() => mutation.mutate("start")} disabled={mutation.isPending} pending={mutation.isPending} pendingLabel="Starting…">Start focus</Button>
        </div>
      )}
      {session && (
        <div className="focus-session-card">
          <div className="focus-session-heading"><span className="card-kicker">{session.state === "running" ? "IN FOCUS" : "PAUSED"} · {session.mode}</span><span className="focus-session-live">{session.state === "running" ? "Recording now" : "Timer held"}</span></div>
          <h2>{session.title}</h2>
          <div className={`focus-timer${overtime ? " is-overtime" : ""}`} aria-live="polite">{breakUntil ? formatDuration(breakRemaining) : targetSeconds > 0 ? `${overtime ? "+" : "−"}${formatDuration(timerSeconds)}` : formatDuration(elapsed)}</div>
          <p className="focus-evidence">{breakUntil ? `Break ${pomodoroCycle - 1} of ${pomodoroCycles} · resume when ready.` : overtime ? session.mode === "pomodoro" ? `Cycle ${pomodoroCycle} complete — take a short break or keep going honestly.` : "Target reached — overtime is being recorded honestly." : targetSeconds > 0 ? `${formatDuration(targetSeconds - elapsed)} remaining in the planned window.` : "Open session; stop when the work stops."} Active time recorded: {formatDuration(Number(session.activeSeconds))}. Paused time is not counted as work.</p>
          <dl className="focus-session-facts"><div><dt>Mode</dt><dd>{session.mode}</dd></div><div><dt>State</dt><dd>{session.state}</dd></div><div><dt>Evidence</dt><dd>Server saved</dd></div></dl>
          <div className="focus-actions">
            {session.state === "running" && !(session.mode === "pomodoro" && overtime) && <Select aria-label="Pause reason" value={pauseReason} onChange={(event) => setPauseReason(event.target.value as PauseEvent["reason"])} options={[{ value: "interrupted", label: "Interrupted" }, { value: "blocked", label: "Blocked" }, { value: "distracted", label: "Distracted" }, { value: "rest", label: "Rest" }, { value: "other", label: "Other" }]} />}
            {session.state === "running" ? <Button className="primary-action" type="button" onClick={() => mutation.mutate(session.mode === "pomodoro" && overtime ? "break" : "pause")} disabled={mutation.isPending} pending={mutation.isPending} pendingLabel="Pausing…">{session.mode === "pomodoro" && overtime ? "Start 5-minute break" : "Pause"}</Button> : <Button className="primary-action" type="button" onClick={() => mutation.mutate("resume")} disabled={mutation.isPending} pending={mutation.isPending} pendingLabel="Resuming…">{breakUntil ? "Resume cycle" : "Resume"}</Button>}
            <Button className="quiet-action" variant="secondary" type="button" onClick={() => mutation.mutate("end")} disabled={mutation.isPending}>End session</Button>
          </div>
          {mutation.isError && <p role="alert">That transition did not save. Nothing was assumed.</p>}
        </div>
      )}
      {endedSession && !session && <section className="focus-note-card" aria-labelledby="focus-note-heading"><span className="card-kicker">CLOSE THE LOOP</span><h2 id="focus-note-heading">What did you get done?</h2><p>Leave one useful observation for Review. It stays attached to this completed focus session.</p><form onSubmit={(event) => { event.preventDefault(); if (sessionNote.trim()) noteMutation.mutate(); }}><FormField label="End-of-session note" required control={<Input aria-label="End-of-session note" value={sessionNote} onChange={(event) => setSessionNote(event.target.value)} placeholder="A small result, learning, or next step" />} /><Button type="submit" disabled={noteMutation.isPending || sessionNote.trim() === ""} pending={noteMutation.isPending} pendingLabel="Saving…">Save note</Button></form>{noteMutation.isSuccess && <p role="status">Saved to Review.</p>}{noteMutation.isError && <p role="alert">The note could not be saved.</p>}</section>}
      {!session && <section className="actuals-launch" aria-labelledby="actuals-launch-heading">
        <div><span className="card-kicker">HONEST CATCH-UP</span><h2 id="actuals-launch-heading">Need to account for earlier work?</h2><p>Keep remembered time separate from the live timer and label it as approximate.</p></div>
        <Button type="button" variant="secondary" onClick={() => setActualsOpen(true)}>Open actuals ledger</Button>
      </section>}
      <Dialog open={actualsOpen} title="Actuals ledger" description="Record remembered time as an approximation. Corrections stay visible instead of rewriting history." onClose={() => { setActualsOpen(false); setEditingActual(null); }} closeLabel="Close actuals ledger" contentClassName="actuals-dialog">
      <section className="actuals-card" aria-label="Actuals ledger entries">
        <div className="actuals-heading"><div><span className="card-kicker">HONEST CATCH-UP</span><h2 id="actuals-heading">Record time you already spent</h2></div><span className="actuals-date">{actualDate}</span></div>
        <p className="actuals-intro">A rough memory is useful when it stays labeled as a report, not a fabricated timer interval.</p>
        <form className="actual-form" onSubmit={(event) => { event.preventDefault(); actualMutation.mutate(); }}>
          <FormField label="What did you work on?" required control={<Input aria-label="What did you work on?" value={actualTitle} onChange={(event) => setActualTitle(event.target.value)} placeholder="e.g. Review the launch brief" />} />
          <FormField label="Minutes" required control={<Input aria-label="Minutes" type="number" min="1" step="1" value={actualMinutes} onChange={(event) => setActualMinutes(event.target.value)} />} />
          <FormField className="actual-note-field" label="Context (optional)" control={<Input aria-label="Context (optional)" value={actualNote} onChange={(event) => setActualNote(event.target.value)} placeholder="Approximate, interrupted, or complete" />} />
          <Button className="quiet-action" variant="secondary" type="submit" disabled={actualMutation.isPending} pending={actualMutation.isPending} pendingLabel="Saving…">Record actual</Button>
        </form>
        {actualMutation.isError && <p className="actual-error" role="alert">That actual did not save. Nothing was assumed.</p>}
        {actuals.isError && <p className="actual-error" role="alert">Recorded actuals are unavailable right now.</p>}
        {corrections.isError && <p className="actual-error" role="alert">Correction history is unavailable right now; the current actuals remain intact.</p>}
        {actuals.data && actuals.data.length > 0 && <div className="actual-list" aria-label="Recorded actual activity">{actuals.data.map((actual) => { const history = corrections.data?.filter((correction) => correction.actualId === actual.id) ?? []; return <article className="actual-row" key={actual.id}><div><strong>{actual.title}</strong><span>{Number(actual.reportedMinutes)} min · {actual.certainty === "timed_observed" ? "observed" : "approximate"}</span>{actual.note && <small>{actual.note}</small>}{history.length > 0 && <details><summary>{history.length} correction{history.length === 1 ? "" : "s"} preserved</summary><ul>{history.map((correction) => <li key={correction.id}>{Number(correction.previousMinutes)} → {Number(correction.newMinutes)} min{correction.reason ? ` · ${correction.reason}` : ""}</li>)}</ul></details>}</div><Button type="button" className="quiet-action" variant="secondary" onClick={() => { setEditingActual(actual); setCorrectionMinutes(String(actual.reportedMinutes)); setCorrectionNote(actual.note); }}>Correct</Button></article>; })}</div>}
        {editingActual && <form className="actual-correction" onSubmit={(event) => { event.preventDefault(); correctionMutation.mutate(); }}><strong>Correct “{editingActual.title}”</strong><FormField label="Minutes" required control={<Input aria-label="Minutes" type="number" min="1" value={correctionMinutes} onChange={(event) => setCorrectionMinutes(event.target.value)} />} /><FormField label="Why did it change?" required control={<Input aria-label="Why did it change?" value={correctionNote} onChange={(event) => setCorrectionNote(event.target.value)} />} /><div className="focus-actions"><Button className="primary-action" type="submit" disabled={correctionMutation.isPending} pending={correctionMutation.isPending} pendingLabel="Saving…">Save correction</Button><Button className="quiet-action" variant="secondary" type="button" onClick={() => setEditingActual(null)}>Cancel</Button></div></form>}
      </section>
      </Dialog>
    </section></ObservatoryScene>
  );
}

function todayLocalDate(): string {
  const date = new Date();
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
