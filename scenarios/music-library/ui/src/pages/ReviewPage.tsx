import { useEffect, useMemo, useState } from "react";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { getDecomposition, listDecompositions, sourceAudioUrl, submitReview, type Correction, type Decomposition, type DecompositionSummary } from "../api/decompositions";

type Cue = { start: number; end: number; title: string; detail: string; kind: "cut" | "transcript" | "section" };
function object(value: unknown): Record<string, unknown> { return value && typeof value === "object" ? value as Record<string, unknown> : {}; }
function cuesFrom(doc: Decomposition): Cue[] {
  const result = doc.tool_result;
  const analysis = object(result.analysis);
  const transcript = object(analysis.lyric_alignment);
  const raw = [
    ...(Array.isArray(analysis.sections) ? analysis.sections : []),
    ...(Array.isArray(analysis.cuts) ? analysis.cuts : []),
    ...(Array.isArray(analysis.section_boundaries) ? analysis.section_boundaries : []),
    ...(Array.isArray(analysis.segments) ? analysis.segments : []),
    ...(Array.isArray(analysis.transcript_segments) ? analysis.transcript_segments : []),
    ...(Array.isArray(transcript.segments) ? transcript.segments : []),
  ];
  const duration = Number(object(analysis.media).duration_seconds) || Number(object(result.media).duration_seconds) || Infinity;
  return raw.map(item => {
    const row = object(item);
    const point = Number(row.time_seconds);
    const start = Number(row.start_seconds ?? row.start ?? row.start_time ?? point ?? 0);
    const inferredEnd = Number.isFinite(point) ? Math.min(duration, point + 3) : start;
    const end = Number(row.end_seconds ?? row.end ?? row.end_time ?? inferredEnd);
    const typeName = String(row.kind ?? row.type ?? (row.text ? "transcript" : row.label ? "section" : "cut"));
    const kind: Cue["kind"] = typeName.includes("transcript") ? "transcript" : typeName.includes("section") ? "section" : "cut";
    return { start, end, title: String(row.label ?? row.section ?? row.kind ?? row.type ?? "Audio cue"), detail: String(row.text ?? row.description ?? row.comment ?? ""), kind };
  }).filter(c => Number.isFinite(c.start) && Number.isFinite(c.end) && c.start >= 0 && c.end > c.start).sort((a,b) => a.start-b.start).slice(0, 500);
}
const timecode = (value: number) => `${Math.floor(value / 60)}:${String(Math.floor(value % 60)).padStart(2, "0")}`;

export function ReviewPage() {
  const { t } = useTranslation();
  const [items, setItems] = useState<DecompositionSummary[]>([]);
  const [cursor, setCursor] = useState("");
  const [next, setNext] = useState("");
  const [selected, setSelected] = useState("");
  const [selectedJob, setSelectedJob] = useState("");
  const [detail, setDetail] = useState<Decomposition | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [verdict, setVerdict] = useState("confirmed");
  const [note, setNote] = useState("");
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [label, setLabel] = useState("");
  const [corrections, setCorrections] = useState<Correction[]>([]);
  const [saved, setSaved] = useState(false);

  const loadPage = async (pageCursor = "") => {
    setLoading(true); setError("");
    try { const page = await listDecompositions(pageCursor); setItems(page.items); setCursor(pageCursor); setNext(page.next_cursor ?? ""); if (!selected && page.items[0]) { setSelected(page.items[0].track_sha256); setSelectedJob(page.items[0].analysis_job_id); } }
    catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.review.loadError)); }
    finally { setLoading(false); }
  };
  useEffect(() => { void loadPage(); }, []);
  useEffect(() => { if (!selected || !selectedJob) { setDetail(null); return; } let live = true; setDetail(null); getDecomposition(selected, selectedJob).then(d => { if (live) setDetail(d); }).catch(e => { if (live) setError(e instanceof Error ? e.message : t(strings.pages.review.loadError)); }); return () => { live = false; }; }, [selected, selectedJob]);
  const cues = useMemo(() => detail ? cuesFrom(detail) : [], [detail]);
  const jump = (seconds: number) => { const audio = document.getElementById("source-audio") as HTMLAudioElement | null; if (audio) { audio.currentTime = seconds; void audio.play(); } };
  const addCorrection = () => { const a = Number(start), b = Number(end); if (!Number.isFinite(a) || !Number.isFinite(b) || a < 0 || b <= a || !label.trim()) return; setCorrections([...corrections, { start_seconds: a, end_seconds: b, label: label.trim() }]); setStart(""); setEnd(""); setLabel(""); };
  const save = async () => { if (!detail) return; setSaved(false); setError(""); try { await submitReview(detail.track_sha256, { analysis_job_id: detail.analysis_job_id, verdict, note, timestamped_corrections: corrections }); setSaved(true); setNote(""); setCorrections([]); await loadPage(cursor); const fresh = await getDecomposition(detail.track_sha256); setDetail(fresh); } catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.review.saveError)); } };

  return <main data-testid={selectors.pages.review} aria-labelledby="review-heading" className="mx-auto flex w-full max-w-6xl flex-col gap-5 px-4 pb-28 pt-6 sm:px-6">
    <header><p className="text-sm font-medium text-app-muted-foreground">{t(strings.pages.review.eyebrow)}</p><h1 id="review-heading" className="text-2xl font-semibold">{t(strings.pages.review.title)}</h1><p className="mt-1 max-w-3xl text-app-muted-foreground">{t(strings.pages.review.description)}</p></header>
    {error && <p role="alert" className="rounded-md border border-red-500/40 bg-red-500/10 p-3 text-sm">{error}</p>}{saved && <p role="status" className="rounded-md border border-green-500/40 bg-green-500/10 p-3 text-sm">{t(strings.pages.review.saved)}</p>}
    <div className="grid gap-5 lg:grid-cols-[minmax(16rem,0.8fr)_minmax(0,1.7fr)]">
      <section aria-label={t(strings.pages.review.queueLabel)} className="min-w-0 rounded-xl border border-app-border bg-app-surface p-3">
        <div className="flex items-center justify-between px-1 pb-2"><h2 className="font-semibold">{t(strings.pages.review.queue)}</h2><button className="rounded-md border px-3 py-2 text-sm" onClick={() => void loadPage()}>{t(strings.pages.review.refresh)}</button></div>
        {loading && <p className="p-3 text-sm text-app-muted-foreground">{t(strings.pages.review.loading)}</p>}
        {!loading && !items.length && <p className="p-3 text-sm text-app-muted-foreground">{t(strings.pages.review.empty)}</p>}
        <ul className="flex flex-col gap-1">{items.map(item => <li key={`${item.track_sha256}:${item.analysis_job_id}`}><button onClick={() => { setSelected(item.track_sha256); setSelectedJob(item.analysis_job_id); setSaved(false); }} aria-current={selected === item.track_sha256 && selectedJob === item.analysis_job_id ? "true" : undefined} className={`w-full rounded-lg px-3 py-3 text-left ${selected === item.track_sha256 && selectedJob === item.analysis_job_id ? "bg-app-accent text-app-primary-foreground" : "hover:bg-app-surface-muted"}`}><span className="block font-medium">{item.title || item.artist || item.track_sha256.slice(0, 12)}</span><span className="block truncate text-sm opacity-75">{item.artist || item.uploader || item.track_sha256}</span><span className="mt-1 block text-xs opacity-70">{item.review_count} {t(strings.pages.review.reviewCount)}</span></button></li>)}</ul>
        {next && <button className="mt-3 w-full rounded-md border px-3 py-2" onClick={() => void loadPage(next)}>{t(strings.pages.review.nextPage)}</button>}{cursor && <button className="mt-2 w-full rounded-md px-3 py-2 text-sm" onClick={() => void loadPage()}>{t(strings.pages.review.firstPage)}</button>}
      </section>
      <section className="min-w-0 space-y-5">
        {!detail && <div className="rounded-xl border border-app-border p-5 text-app-muted-foreground">{items.length ? t(strings.pages.review.selectTrack) : t(strings.pages.review.noDetail)}</div>}
        {detail && <>
          <section className="rounded-xl border border-app-border bg-app-surface p-4 sm:p-5"><h2 className="text-xl font-semibold">{detail.source?.title || selected.slice(0, 12)}</h2><p className="mt-1 text-sm text-app-muted-foreground">{detail.source?.artist || detail.source?.uploader || selected}</p><audio id="source-audio" controls preload="metadata" className="mt-4 w-full" src={sourceAudioUrl(detail.track_sha256)} aria-label={t(strings.pages.review.audioLabel)} />
            <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.pages.review.privateAudio)} · {t(strings.pages.review.job)}: {detail.analysis_job_id}</p>
          </section>
          <section className="rounded-xl border border-app-border bg-app-surface p-4 sm:p-5"><div className="flex flex-wrap items-end justify-between gap-2"><div><h2 className="text-lg font-semibold">{t(strings.pages.review.timeline)}</h2><p className="text-sm text-app-muted-foreground">{t(strings.pages.review.timelineHint)}</p></div><span className="text-xs text-app-muted-foreground">{cues.length} {t(strings.pages.review.cues)}</span></div>
            {!cues.length && <p className="mt-4 rounded-md bg-app-surface-muted p-3 text-sm">{t(strings.pages.review.noCues)}</p>}
            <ol className="mt-3 space-y-2">{cues.map((cue, i) => <li key={`${cue.start}-${cue.end}-${i}`} className="rounded-lg border border-app-border p-3"><div className="flex items-start gap-3"><button onClick={() => jump(cue.start)} className="shrink-0 rounded-md border px-3 py-2 font-mono text-sm" aria-label={`${t(strings.pages.review.playFrom)} ${timecode(cue.start)}`}>{timecode(cue.start)}</button><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><strong>{cue.title}</strong><span className="rounded-full bg-app-surface-muted px-2 py-0.5 text-xs">{cue.kind === "cut" ? t(strings.pages.review.lowConfidence) : cue.kind === "transcript" ? t(strings.pages.review.transcriptHypothesis) : t(strings.pages.review.sectionCandidate)}</span><span className="text-xs text-app-muted-foreground">{timecode(cue.start)}–{timecode(cue.end)}</span></div>{cue.detail && <p className="mt-1 break-words text-sm text-app-muted-foreground">{cue.detail}</p>}</div></div></li>)}</ol>
          </section>
          <section className="rounded-xl border border-app-border bg-app-surface p-4 sm:p-5"><h2 className="text-lg font-semibold">{t(strings.pages.review.feedback)}</h2><p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.review.feedbackHint)}</p>
            <fieldset className="mt-4 flex flex-wrap gap-2"><legend className="mb-2 text-sm font-medium">{t(strings.pages.review.verdict)}</legend>{(["confirmed", "corrected", "needs_work"] as const).map(value => <button type="button" key={value} aria-pressed={verdict === value} onClick={() => setVerdict(value)} className={`rounded-full border px-4 py-2 text-sm ${verdict === value ? "bg-app-accent text-app-primary-foreground" : ""}`}>{t(strings.pages.review[value])}</button>)}</fieldset>
            <label className="mt-4 block text-sm font-medium" htmlFor="review-note">{t(strings.pages.review.note)}</label><textarea id="review-note" value={note} onChange={e => setNote(e.target.value)} rows={3} maxLength={4000} className="mt-1 w-full rounded-md border border-app-border bg-app-background p-3" placeholder={t(strings.pages.review.notePlaceholder)} />
            <fieldset className="mt-4"><legend className="text-sm font-medium">{t(strings.pages.review.addCorrection)}</legend><div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-[1fr_1fr_2fr_auto]"><label className="text-xs">{t(strings.pages.review.start)}<input type="number" min="0" step="0.1" value={start} onChange={e => setStart(e.target.value)} className="mt-1 w-full rounded-md border bg-app-background p-2 text-sm" /></label><label className="text-xs">{t(strings.pages.review.end)}<input type="number" min="0" step="0.1" value={end} onChange={e => setEnd(e.target.value)} className="mt-1 w-full rounded-md border bg-app-background p-2 text-sm" /></label><label className="col-span-2 text-xs sm:col-span-1">{t(strings.pages.review.label)}<input value={label} onChange={e => setLabel(e.target.value)} maxLength={240} className="mt-1 w-full rounded-md border bg-app-background p-2 text-sm" /></label><button type="button" onClick={addCorrection} className="col-span-2 self-end rounded-md border px-4 py-2 text-sm sm:col-span-1">{t(strings.pages.review.add)}</button></div></fieldset>
            {!!corrections.length && <ul className="mt-3 space-y-1">{corrections.map((c,i) => <li key={`${c.start_seconds}-${i}`} className="flex items-center justify-between rounded-md bg-app-surface-muted p-2 text-sm"><span>{timecode(c.start_seconds)}–{timecode(c.end_seconds)} · {c.label}</span><button onClick={() => setCorrections(corrections.filter((_, index) => index !== i))} aria-label={t(strings.pages.review.removeCorrection)} className="px-2 py-1">×</button></li>)}</ul>}
            <button onClick={() => void save()} className="mt-5 w-full rounded-md bg-app-primary px-4 py-3 font-semibold text-app-primary-foreground sm:w-auto">{t(strings.pages.review.save)}</button>
          </section>
        </>}
      </section>
    </div>
  </main>;
}
