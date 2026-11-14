/**
 * Context and compaction controls. With a session id it shows that session's
 * size and offers "Compact when idle" and an optional threshold watcher; with
 * none it lists every open Claude and Codex session with its size, then
 * compacts only the ones the operator ticks. The server decides each time
 * whether a session may be typed into, and every outcome, including each skip
 * and its reason, is shown here.
 */
import { useCallback, useEffect, useState } from "react";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { compactIdleSessions, getCompactCandidates, getSessionCompaction, setSessionCompaction, type CompactCandidate, type CompactEvent, type CompactResult, type SessionCompaction } from "../api/usageMode";
import { useContextCompactionStore } from "../stores/useContextCompactionStore";

const THRESHOLDS = [0, 150_000, 200_000, 300_000];

function stateKey(state: string) {
  const keys = strings.contextCompaction.state;
  return state === "working" ? keys.working : state === "idle" ? keys.idle : state === "waiting" ? keys.waiting : keys.unknown;
}

function kTokens(value: number | undefined): number {
  return Math.round((value ?? 0) / 1000);
}

function EventRow({ event }: { event: CompactEvent }) {
  const { t } = useTranslation();
  const outcome = event.outcome === "compacted" ? strings.contextCompaction.outcomeCompacted : event.outcome === "failed" ? strings.contextCompaction.outcomeFailed : strings.contextCompaction.outcomeSkipped;
  const by = event.trigger === "manual" ? strings.contextCompaction.by.manual : event.trigger === "armed" ? strings.contextCompaction.by.armed
    : event.trigger === "threshold" ? strings.contextCompaction.by.threshold : event.trigger === "bulk" ? strings.contextCompaction.by.bulk : null;
  return (
    <li data-testid="compaction-event" data-outcome={event.outcome} className="rounded-lg border border-wc-default px-3 py-2 text-sm">
      <span className="font-semibold text-wc-text-primary">{t(outcome)}</span>
      {by ? <span className="text-wc-text-muted"> · {t(by)}</span> : null}
      {event.tokensBefore ? <span className="text-wc-text-muted"> · {t(strings.contextCompaction.before, { count: kTokens(event.tokensBefore) })}</span> : null}
      {event.tokensAfter ? <span className="text-wc-text-muted"> · {t(strings.contextCompaction.after, { count: kTokens(event.tokensAfter) })}</span> : null}
      {event.reason ? <p className="break-words text-xs text-wc-text-muted" data-testid="compaction-reason">{event.reason}</p> : null}
    </li>
  );
}

function SessionView({ sessionId }: { sessionId: string }) {
  const { t } = useTranslation();
  const [state, setState] = useState<SessionCompaction | null>(null);
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const apply = useCallback(async (body: Parameters<typeof setSessionCompaction>[1]) => {
    setBusy(true); setFailed(false);
    try { setState(await setSessionCompaction(sessionId, body)); } catch { setFailed(true); } finally { setBusy(false); }
  }, [sessionId]);
  useEffect(() => {
    const abort = new AbortController();
    getSessionCompaction(sessionId, abort.signal).then((next) => { if (!abort.signal.aborted) setState(next); }).catch(() => { if (!abort.signal.aborted) setFailed(true); });
    return () => { abort.abort(); };
  }, [sessionId]);
  const armed = state?.watch.armed ?? false;
  const threshold = state?.watch.thresholdTokens ?? 0;
  return (
    <div className="space-y-4 p-4" data-testid="compaction-session">
      <p className="text-sm text-wc-text-muted">{t(strings.contextCompaction.explain)}</p>
      <section className="space-y-2">
        <Button type="button" variant="primary" size="md" className="min-h-11 w-full" disabled={busy || armed} data-testid="compaction-when-idle" onClick={() => { void apply({ action: "compact_when_idle" }); }}>{t(strings.contextCompaction.compactNow)}</Button>
        {armed ? (
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-wc-text-secondary" data-testid="compaction-armed">{t(strings.contextCompaction.armed)}</p>
            <Button type="button" variant="secondary" size="md" className="min-h-11" disabled={busy} onClick={() => { void apply({ action: "cancel" }); }}>{t(strings.contextCompaction.cancelArmed)}</Button>
          </div>
        ) : null}
      </section>
      <section className="space-y-2" aria-label={t(strings.contextCompaction.thresholdLabel)}>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">{t(strings.contextCompaction.thresholdLabel)}</h3>
        <div role="radiogroup" className="flex flex-wrap gap-2">
          {THRESHOLDS.map((value) => (
            <button key={value} type="button" role="radio" aria-checked={threshold === value} disabled={busy} data-testid={`compaction-threshold-${String(value)}`}
              onClick={() => { void apply({ action: "set_threshold", thresholdTokens: value }); }}
              className={`min-h-11 rounded-lg border px-3 text-sm ${threshold === value ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary" : "border-wc-default text-wc-text-secondary"}`}>
              {value === 0 ? t(strings.contextCompaction.thresholdOff) : t(strings.contextCompaction.thresholdValue, { count: kTokens(value) })}
            </button>))}
        </div>
        <p className="text-xs text-wc-text-muted">{t(strings.contextCompaction.thresholdHint)}</p>
      </section>
      <section className="space-y-2">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">{t(strings.contextCompaction.history)}</h3>
        {state && state.history.length > 0
          ? <ul className="m-0 list-none space-y-2 p-0">{[...state.history].reverse().map((event) => <EventRow key={`${event.at}-${event.trigger}`} event={event} />)}</ul>
          : <p className="text-sm text-wc-text-muted">{t(strings.contextCompaction.noHistory)}</p>}
      </section>
      {failed ? <p role="alert" data-testid="compaction-error" className="text-sm text-wc-error-text">{t(strings.contextCompaction.error)}</p> : null}
    </div>
  );
}

function AllView() {
  const { t } = useTranslation();
  const [candidates, setCandidates] = useState<CompactCandidate[] | null>(null);
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [results, setResults] = useState<CompactResult[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const load = useCallback(async (signal?: AbortSignal) => {
    try {
      const rows = await getCompactCandidates(signal);
      if (signal?.aborted) return;
      setCandidates(rows);
      setSelected(Object.fromEntries(rows.filter((row) => row.eligible).map((row) => [row.sessionId, true])));
    } catch { if (!signal?.aborted) setFailed(true); }
  }, []);
  useEffect(() => {
    const abort = new AbortController();
    void load(abort.signal);
    return () => { abort.abort(); };
  }, [load]);
  const chosen = (candidates ?? []).filter((row) => selected[row.sessionId]).map((row) => row.sessionId);
  const run = async () => {
    setBusy(true); setFailed(false);
    try { setResults(await compactIdleSessions(chosen)); } catch { setFailed(true); } finally { setBusy(false); }
  };
  const label = (id: string) => candidates?.find((row) => row.sessionId === id)?.label ?? id.slice(0, 8);
  return (
    <div className="space-y-4 p-4" data-testid="compaction-all">
      <p className="text-sm text-wc-text-muted">{t(strings.contextCompaction.explain)}</p>
      {candidates === null && !failed ? <p className="text-sm text-wc-text-muted">{t(strings.contextCompaction.loading)}</p> : null}
      {candidates?.length === 0 ? <p className="text-sm text-wc-text-muted">{t(strings.contextCompaction.none)}</p> : null}
      {results ? (
        <ul className="m-0 list-none space-y-2 p-0" data-testid="compaction-results">
          {results.map((result) => (
            <li key={result.sessionId} data-testid="compaction-result" data-outcome={result.outcome} className="space-y-1">
              <p className="text-sm font-semibold text-wc-text-primary">{label(result.sessionId)}</p>
              <ul className="m-0 list-none p-0"><EventRow event={result} /></ul>
            </li>))}
        </ul>
      ) : (
        <>
          <ul className="m-0 list-none space-y-2 p-0">
            {(candidates ?? []).map((row) => (
              <li key={row.sessionId} data-testid="compaction-candidate" data-eligible={row.eligible} className="flex items-start gap-3 rounded-lg border border-wc-default px-3 py-2">
                <input type="checkbox" className="mt-1 h-5 w-5" aria-label={row.label} checked={Boolean(selected[row.sessionId])} disabled={!row.eligible}
                  onChange={(event) => { setSelected((prev) => ({ ...prev, [row.sessionId]: event.target.checked })); }} />
                <div className="min-w-0 flex-1">
                  <p className="break-words text-sm font-semibold text-wc-text-primary">{row.label} <span className="font-normal text-wc-text-muted">· {row.harness} · {t(stateKey(row.state))}{row.goalActive ? ` · ${t(strings.contextCompaction.goal)}` : ""}</span></p>
                  <p className="text-xs text-wc-text-muted">{row.contextTokens > 0 ? t(strings.contextCompaction.tokens, { count: kTokens(row.contextTokens) }) : t(strings.contextCompaction.unknownSize)}</p>
                  {row.skipReason ? <p className="break-words text-xs text-wc-text-secondary" data-testid="compaction-skip-reason">{row.skipReason}</p> : null}
                </div>
              </li>))}
          </ul>
          <Button type="button" variant="primary" size="md" className="min-h-11 w-full" disabled={busy || chosen.length === 0} data-testid="compaction-run" onClick={() => { void run(); }}>{t(strings.contextCompaction.compactSelected, { count: chosen.length })}</Button>
        </>
      )}
      {failed ? <p role="alert" data-testid="compaction-error" className="text-sm text-wc-error-text">{t(strings.contextCompaction.error)}</p> : null}
    </div>
  );
}

/** Mounted once in the app; opened through the store. */
export default function ContextCompactionSheet() {
  const { t } = useTranslation();
  const { open, sessionId, hide } = useContextCompactionStore();
  if (!open) return null;
  return (
    <BottomSheet avoidKeyboard open onClose={hide} title={t(sessionId ? strings.contextCompaction.title : strings.contextCompaction.allTitle)} closeLabel={t(strings.contextCompaction.close)} testId="context-compaction-sheet">
      {sessionId ? <SessionView sessionId={sessionId} /> : <AllView />}
    </BottomSheet>
  );
}
