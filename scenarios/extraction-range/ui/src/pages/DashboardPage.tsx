import { useCallback, useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { buildApiUrl } from "@vrooli/api-base";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import { Activity, ArrowLeft, CheckCircle2, ChevronRight, ExternalLink, LoaderCircle, RefreshCw, Trophy } from "lucide-react";

import { API_BASE } from "../api/client";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type Level = { id: string; family: string; family_name: string; difficulty: number; title: string; task: string; tags: string[]; target_url: string };
type Score = { tool: string; level_id: string; family: string; correct: number; submitted: number; expected: number; precision: number; recall: number; f1: number; passed: boolean };
type Payload = { levels: Level[] };
type ScorePayload = { scores: Score[] };
const url = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(url(path), { cache: "no-store" });
  if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
  return response.json() as Promise<T>;
}

function useNarrowScreen() {
  const [narrow, setNarrow] = useState(() => typeof window !== "undefined" && window.matchMedia("(max-width: 1279px)").matches);
  useEffect(() => {
    const query = window.matchMedia("(max-width: 1279px)");
    const update = () => setNarrow(query.matches);
    update(); query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return narrow;
}

function StateMessage({ kind, children, retry }: { kind: "loading" | "error" | "empty"; children: ReactNode; retry?: () => void }) {
  return <div className={`range-state range-state-${kind}`} role={kind === "error" ? "alert" : "status"}>
    {kind === "loading" && <LoaderCircle size={18} className="range-spin" aria-hidden="true" />}
    <span>{children}</span>{retry && <button className="range-link-button" onClick={retry}>Try again</button>}
  </div>;
}

/** The range is an operator workbench: select a challenge, submit an extractor result, then compare persistent bests. */
export function DashboardPage() {
  const { t } = useTranslation();
  const narrow = useNarrowScreen();
  const [levels, setLevels] = useState<Level[]>([]);
  const [scores, setScores] = useState<Score[]>([]);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");
  const [family, setFamily] = useState("all");
  const [selectedId, setSelectedId] = useState("");
  const [tool, setTool] = useState("");
  const [recordsText, setRecordsText] = useState("[]");
  const [submitError, setSubmitError] = useState("");
  const [result, setResult] = useState<Score | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [mobileView, setMobileView] = useState<"challenges" | "detail" | "scores">("challenges");

  const refresh = useCallback(async () => {
    setLoading(true); setFailure("");
    try {
      const [catalog, board] = await Promise.all([getJSON<Payload>("/levels"), getJSON<ScorePayload>("/scoreboard")]);
      setLevels(catalog.levels ?? []); setScores(board.scores ?? []);
      setSelectedId((current) => current || catalog.levels?.[0]?.id || "");
    } catch (error) { setFailure(error instanceof Error ? error.message : "Could not load the range."); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { void refresh(); }, [refresh]);

  const families = useMemo(() => [...new Map(levels.map((item) => [item.family, item.family_name])).entries()], [levels]);
  const visibleLevels = useMemo(() => levels.filter((item) => family === "all" || item.family === family), [levels, family]);
  const selected = levels.find((item) => item.id === selectedId) ?? visibleLevels[0];
  const visibleScores = scores.filter((item) => family === "all" || item.family === family);
  const passedCount = scores.filter((score) => score.passed).length;
  const resetSubmission = () => { setRecordsText("[]"); setSubmitError(""); setResult(null); };
  const selectLevel = (id: string) => { setSelectedId(id); resetSubmission(); if (narrow) setMobileView("detail"); };
  const chooseFamily = (value: string) => { setFamily(value); setSelectedId(""); resetSubmission(); };

  async function submit(event: FormEvent) {
    event.preventDefault(); setSubmitError(""); setResult(null);
    if (!selected) return;
    const toolName = tool.trim();
    if (!toolName) { setSubmitError("Add a tool name so this result can be attributed on the scoreboard."); return; }
    let records: unknown;
    try { records = JSON.parse(recordsText); } catch { setSubmitError("Results must be valid JSON."); return; }
    if (!Array.isArray(records) || records.some((record) => !record || typeof record !== "object" || Array.isArray(record))) {
      setSubmitError("Enter a JSON array of record objects, for example [{\"id\":\"A1\",\"name\":\"Sample\"}]."); return;
    }
    setSubmitting(true);
    try {
      const response = await fetch(url("/score"), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tool: toolName, level_id: selected.id, records }) });
      if (!response.ok) throw new Error(await response.text());
      const scored = await response.json() as Score; setResult(scored); await refresh();
    } catch (error) { setSubmitError(error instanceof Error ? error.message : "Submission failed. Check the payload and retry."); }
    finally { setSubmitting(false); }
  }

  const catalogPanel = <section className="range-panel range-catalog" aria-label="Challenge catalog" data-testid={selectors.range.catalog}>
    <div className="range-panel-heading"><div><p className="range-eyebrow">PRACTICE RANGE</p><h2>Challenges <span className="range-count">{visibleLevels.length}</span></h2></div>
      <label className="range-filter"><span>Family</span><select value={family} onChange={(event) => chooseFamily(event.target.value)} aria-label="Filter challenges by family" data-testid={selectors.range.familyFilter}><option value="all">All families</option>{families.map(([id, name]) => <option key={id} value={id}>{name}</option>)}</select></label>
    </div>
    <div className="range-level-list" role="listbox" aria-label="Challenges" data-testid={selectors.range.levelList}>
      {visibleLevels.map((level) => <button key={level.id} role="option" aria-selected={level.id === selected?.id} className={`range-level-row${level.id === selected?.id ? " is-selected" : ""}`} onClick={() => selectLevel(level.id)} data-testid={selectors.range.level({ key: level.id })}>
        <span className="range-level-index">{String(level.difficulty).padStart(2, "0")}</span><span className="range-level-copy"><strong>{level.title}</strong><small>{level.family_name} · {level.id}</small></span><ChevronRight size={16} aria-hidden="true" />
      </button>)}
      {!loading && visibleLevels.length === 0 && <StateMessage kind="empty">No challenges in this family yet.</StateMessage>}
    </div>
  </section>;

  const challengePanel = <section className="range-panel range-challenge" aria-label="Selected challenge" data-testid={selectors.range.challenge}>
    {selected ? <>
      <div className="range-challenge-top"><div><p className="range-eyebrow">{selected.family_name} <span aria-hidden="true">/</span> LEVEL {String(selected.difficulty).padStart(2, "0")}</p><h2>{selected.title}</h2><p className="range-task">{selected.task}</p></div>
        <a className="range-open-link" href={url(selected.target_url)} target="_blank" rel="noreferrer"><ExternalLink size={15} aria-hidden="true" /> Open practice page<span className="sr-only"> in a new tab</span></a></div>
      <div className="range-tags">{selected.tags?.map((tag) => <span key={tag}>{tag}</span>)}</div>
      <div className="range-submit-head"><div><h3>Submit extracted records</h3><p>Send the completed result to the known-answer grader.</p></div></div>
      <form className="range-submit-form" onSubmit={submit}>
        <label className="range-tool-field">Tool or version<input value={tool} onChange={(event) => setTool(event.target.value)} placeholder="e.g. scrapy-toolkit 0.3" maxLength={120} data-testid={selectors.range.toolInput} /></label>
        <label className="range-payload-field">Extracted records <span>JSON array</span><textarea value={recordsText} onChange={(event) => setRecordsText(event.target.value)} spellCheck={false} rows={8} aria-describedby="range-payload-help" data-testid={selectors.range.recordsInput} /></label>
        <p id="range-payload-help" className="range-help">Use the exact field names and values from the practice page. The grader reports precision, recall and F1.</p>
        {submitError && <p className="range-inline-error" role="alert">{submitError}</p>}
        {result && <div className={`range-result ${result.passed ? "is-pass" : "is-partial"}`} role="status"><CheckCircle2 size={17} aria-hidden="true"/><strong>{result.passed ? "Passed" : "Scored"}</strong><span>{result.correct}/{result.expected} correct · F1 {(result.f1 * 100).toFixed(1)}%</span></div>}
        <button className="range-submit" type="submit" disabled={submitting} data-testid={selectors.range.submitButton}>{submitting ? <LoaderCircle size={17} className="range-spin" /> : <Activity size={17} />} {submitting ? "Scoring…" : "Score extraction"}</button>
      </form>
    </> : <StateMessage kind="empty">Select a challenge to view its instructions.</StateMessage>}
  </section>;

  const scoreboardPanel = <section className="range-panel range-scoreboard" aria-label="Persistent scoreboard" data-testid={selectors.range.scoreboard}>
    <div className="range-panel-heading"><div><p className="range-eyebrow">BEST RESULT PER TOOL</p><h2><Trophy size={19} aria-hidden="true" /> Scoreboard</h2></div><button className="range-icon-button" onClick={() => void refresh()} aria-label="Refresh scoreboard" title="Refresh scoreboard"><RefreshCw size={16} /></button></div>
    <div className="range-score-summary"><strong>{passedCount}</strong><span>passing results</span><small>{scores.length} saved best scores</small></div>
    <div className="range-score-list" data-testid={selectors.range.scoreList}>
      {visibleScores.map((score) => <article className="range-score-row" key={`${score.tool}:${score.level_id}`}>
        <div className="range-score-title"><strong>{score.tool}</strong><span>{score.level_id} · {score.family}</span></div>
        <div className="range-score-metric"><strong>{(score.f1 * 100).toFixed(1)}%</strong><span>F1</span></div>
        <div className={`range-pass-chip ${score.passed ? "is-pass" : ""}`}>{score.passed ? "Pass" : `${score.correct}/${score.expected}`}</div>
      </article>)}
      {!loading && visibleScores.length === 0 && <StateMessage kind="empty">No scores saved{family === "all" ? " yet. Submit a result to start the board." : " for this family."}</StateMessage>}
    </div>
    <p className="range-persistence-note">Best scores are saved across scenario restarts.</p>
  </section>;

  if (loading && levels.length === 0) return <section className="range-workbench" aria-label="Extraction workbench" data-testid={selectors.pages.dashboard}><header className="range-header"><div><p className="range-eyebrow">EXTRACTION RANGE</p><h1>{t(strings.pages.dashboard.title)}</h1></div></header><AsyncPanel surfaceId="workbench-data" state="loading" loading={<StateMessage kind="loading">Loading challenge catalog and scoreboard…</StateMessage>} /></section>;
  if (failure && levels.length === 0) return <section className="range-workbench" aria-label="Extraction workbench" data-testid={selectors.pages.dashboard}><header className="range-header"><div><p className="range-eyebrow">EXTRACTION RANGE</p><h1>{t(strings.pages.dashboard.title)}</h1></div></header><AsyncPanel surfaceId="workbench-data" state="error" onRetry={() => void refresh()} error={<StateMessage kind="error" retry={() => void refresh()}>The range could not be reached. {failure}</StateMessage>} /></section>;

  return <section className="range-workbench" aria-label="Extraction workbench" data-testid={selectors.pages.dashboard}>
    <AsyncPanel surfaceId="workbench-data" state={failure ? "partial" : levels.length === 0 ? "empty" : "ready"} partial={<StateMessage kind="error" retry={() => void refresh()}>Refresh failed. Showing the last loaded data. {failure}</StateMessage>} empty={<StateMessage kind="empty">No challenges are available. Try refreshing the range.</StateMessage>}>
    <header className="range-header"><div><p className="range-eyebrow">EXTRACTION RANGE <span className="range-status-dot" /> API CONNECTED</p><h1>Extraction workbench</h1><p>Run your extractor against owned challenges, then compare its best result.</p></div>
      <div className="range-header-stats"><span><strong>{levels.length}</strong> challenges</span><span><strong>{families.length}</strong> families</span><span><strong>{passedCount}</strong> passes</span></div>
    </header>
    {narrow && <div className="range-mobile-switch" role="tablist" aria-label="Workbench view"><button role="tab" aria-selected={mobileView === "challenges" || mobileView === "detail"} onClick={() => setMobileView("challenges")}>Challenges <span>{visibleLevels.length}</span></button><button role="tab" aria-selected={mobileView === "scores"} onClick={() => setMobileView("scores")}>Scoreboard <span>{visibleScores.length}</span></button></div>}
    <div className={narrow ? "range-mobile-stage" : "range-desktop-grid"}>
      {(!narrow || mobileView === "challenges") && catalogPanel}
      {(!narrow || mobileView === "detail") && <>{narrow && <button className="range-back" onClick={() => setMobileView("challenges")}><ArrowLeft size={16} /> All challenges</button>}{challengePanel}</>}
      {(!narrow || mobileView === "scores") && scoreboardPanel}
    </div>
    </AsyncPanel>
  </section>;
}
