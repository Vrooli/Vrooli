import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { correctActual, endFocus, fetchActualCorrections, fetchActuals, fetchCurrentFocus, pauseFocus, recordManualActual, resumeFocus, startFocus, type Actual, type FocusSession } from "../api/focus";
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
  const selected = work.data?.find((item) => item.id === selectedWorkId) ?? work.data?.[0];
  const session = current.data;

  useEffect(() => {
    if (!session || session.state !== "running") return;
    const timer = window.setInterval(() => setNow(Math.floor(Date.now() / 1000)), 1000);
    return () => window.clearInterval(timer);
  }, [session]);

  const mutation = useMutation({
    mutationFn: async (action: "start" | "pause" | "resume" | "end") => {
      if (action === "start") return startFocus({ workItemId: selected?.id, title: selected?.title ?? "Spontaneous focus", mode: "open" });
      if (!session) throw new Error("No current focus session");
      if (action === "pause") return pauseFocus(session);
      if (action === "resume") return resumeFocus(session);
      return endFocus(session);
    },
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["focus-current"] }); },
  });
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

  return (
    <section className="focus-surface" data-testid={selectors.pages.focus} aria-labelledby="focus-heading">
      <p className="eyebrow">Deep work</p>
      <h1 id="focus-heading">Focus</h1>
      <p className="planner-surface-description">One protected window, with the time recorded honestly.</p>
      {current.isLoading && <p role="status">Checking for an active session…</p>}
      {current.isError && <p role="alert">The focus session is unavailable right now.</p>}
      {!current.isLoading && !current.isError && !session && (
        <div className="focus-start-card">
          <span className="card-kicker">READY WHEN YOU ARE</span>
          <h2>{selected?.title ?? "Spontaneous focus"}</h2>
          <p>{selected ? `${selected.remainingMinutes} minutes remain on this work item.` : "Start without a task and classify the time later."}</p>
          {work.data && work.data.length > 0 && <label>Work item<select value={selectedWorkId || work.data[0]?.id || ""} onChange={(event) => setSelectedWorkId(event.target.value)}>{work.data.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</select></label>}
          <button className="primary-action" type="button" onClick={() => mutation.mutate("start")} disabled={mutation.isPending}>Start focus</button>
        </div>
      )}
      {session && (
        <div className="focus-session-card">
          <span className="card-kicker">{session.state === "running" ? "IN FOCUS" : "PAUSED"} · {session.mode}</span>
          <h2>{session.title}</h2>
          <div className="focus-timer" aria-live="polite">{formatDuration(elapsed)}</div>
          <p className="focus-evidence">Active time recorded: {formatDuration(Number(session.activeSeconds))}. Paused time is not counted as work.</p>
          <div className="focus-actions">
            {session.state === "running" ? <button className="primary-action" type="button" onClick={() => mutation.mutate("pause")} disabled={mutation.isPending}>Pause</button> : <button className="primary-action" type="button" onClick={() => mutation.mutate("resume")} disabled={mutation.isPending}>Resume</button>}
            <button className="quiet-action" type="button" onClick={() => mutation.mutate("end")} disabled={mutation.isPending}>End session</button>
          </div>
          {mutation.isError && <p role="alert">That transition did not save. Nothing was assumed.</p>}
        </div>
      )}
      <section className="actuals-card" aria-labelledby="actuals-heading">
        <div className="actuals-heading"><div><span className="card-kicker">HONEST CATCH-UP</span><h2 id="actuals-heading">Record time you already spent</h2></div><span className="actuals-date">{actualDate}</span></div>
        <p className="actuals-intro">A rough memory is useful when it stays labeled as a report, not a fabricated timer interval.</p>
        <form className="actual-form" onSubmit={(event) => { event.preventDefault(); actualMutation.mutate(); }}>
          <label>What did you work on?<input value={actualTitle} onChange={(event) => setActualTitle(event.target.value)} required placeholder="e.g. Review the launch brief" /></label>
          <label>Minutes<input type="number" min="1" step="1" value={actualMinutes} onChange={(event) => setActualMinutes(event.target.value)} required /></label>
          <label className="actual-note-field">Context (optional)<input value={actualNote} onChange={(event) => setActualNote(event.target.value)} placeholder="Approximate, interrupted, or complete" /></label>
          <button className="quiet-action" type="submit" disabled={actualMutation.isPending}>{actualMutation.isPending ? "Saving…" : "Record actual"}</button>
        </form>
        {actualMutation.isError && <p className="actual-error" role="alert">That actual did not save. Nothing was assumed.</p>}
        {actuals.isError && <p className="actual-error" role="alert">Recorded actuals are unavailable right now.</p>}
        {corrections.isError && <p className="actual-error" role="alert">Correction history is unavailable right now; the current actuals remain intact.</p>}
        {actuals.data && actuals.data.length > 0 && <div className="actual-list" aria-label="Recorded actual activity">{actuals.data.map((actual) => { const history = corrections.data?.filter((correction) => correction.actualId === actual.id) ?? []; return <article className="actual-row" key={actual.id}><div><strong>{actual.title}</strong><span>{Number(actual.reportedMinutes)} min · {actual.certainty === "timed_observed" ? "observed" : "approximate"}</span>{actual.note && <small>{actual.note}</small>}{history.length > 0 && <details><summary>{history.length} correction{history.length === 1 ? "" : "s"} preserved</summary><ul>{history.map((correction) => <li key={correction.id}>{Number(correction.previousMinutes)} → {Number(correction.newMinutes)} min{correction.reason ? ` · ${correction.reason}` : ""}</li>)}</ul></details>}</div><button type="button" className="quiet-action" onClick={() => { setEditingActual(actual); setCorrectionMinutes(String(actual.reportedMinutes)); setCorrectionNote(actual.note); }}>Correct</button></article>; })}</div>}
        {editingActual && <form className="actual-correction" onSubmit={(event) => { event.preventDefault(); correctionMutation.mutate(); }}><strong>Correct “{editingActual.title}”</strong><label>Minutes<input type="number" min="1" value={correctionMinutes} onChange={(event) => setCorrectionMinutes(event.target.value)} required /></label><label>Why did it change?<input value={correctionNote} onChange={(event) => setCorrectionNote(event.target.value)} required /></label><div className="focus-actions"><button className="primary-action" type="submit" disabled={correctionMutation.isPending}>Save correction</button><button className="quiet-action" type="button" onClick={() => setEditingActual(null)}>Cancel</button></div></form>}
      </section>
    </section>
  );
}

function todayLocalDate(): string {
  const date = new Date();
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
