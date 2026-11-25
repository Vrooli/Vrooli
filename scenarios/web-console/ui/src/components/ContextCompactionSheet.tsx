/**
 * Context and compaction controls. With a session id it shows that session's
 * size and offers "Compact when idle" and an optional threshold watcher; with
 * none it lists every supported live session with its context record, then
 * compacts only the eligible ones the operator ticks. The server decides each time
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
import { useHandoffRequestStore } from "../stores/useHandoffRequestStore";
import { pinnedStanceError } from "@vrooli/goal-brief";
import { ContextBudgetSummary } from "./ContextBudgetSummary";

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
      {event.pinnedStanceSent ? <span className="text-wc-text-muted"> · {t(strings.contextCompaction.pinnedStanceSent)}</span> : null}
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
  const [stance, setStance] = useState("");
  const apply = useCallback(async (body: Parameters<typeof setSessionCompaction>[1]) => {
    setBusy(true); setFailed(false);
    try { setState(await setSessionCompaction(sessionId, body)); } catch { setFailed(true); } finally { setBusy(false); }
  }, [sessionId]);
  useEffect(() => {
    let stopped = false;
    let current: AbortController | null = null;
    const refresh = () => {
      current?.abort();
      const controller = new AbortController();
      current = controller;
      getSessionCompaction(sessionId, controller.signal).then((next) => { if (!stopped && !controller.signal.aborted) setState(next); }).catch(() => { if (!stopped && !controller.signal.aborted) setFailed(true); });
    };
    refresh();
    const timer = window.setInterval(refresh, 5000);
    return () => { stopped = true; window.clearInterval(timer); current?.abort(); };
  }, [sessionId]);
  const armed = state?.watch.armed ?? false;
  const threshold = state?.watch.thresholdTokens ?? 0;
  const pinnedStanceMaxBytes = state?.pinnedStanceMaxBytes;
  const stanceError = pinnedStanceMaxBytes === undefined ? null : pinnedStanceError(stance, pinnedStanceMaxBytes);
  const hasStanceError = stance.trim() !== "" && stanceError !== null;
  const stanceBytes = new TextEncoder().encode(stance).byteLength;
  return (
    <div className="space-y-4 p-4" data-testid="compaction-session">
      <p className="text-sm text-wc-text-muted">{t(strings.contextCompaction.explain)}</p>
      <section className="space-y-2 rounded-lg border border-wc-default p-3" aria-label={t(strings.contextCompaction.stanceLabel)}>
        <label htmlFor="context-pin-stance" className="block text-xs font-semibold uppercase tracking-wider text-wc-text-muted">{t(strings.contextCompaction.stanceLabel)}</label>
        <input id="context-pin-stance" type="text" maxLength={pinnedStanceMaxBytes === undefined ? undefined : pinnedStanceMaxBytes * 2} value={stance} disabled={busy || pinnedStanceMaxBytes === undefined} onChange={(event) => { setStance(event.target.value); }}
          className="min-h-11 w-full rounded border border-wc-default bg-wc-surface px-3 text-sm text-wc-text-primary" placeholder={t(strings.contextCompaction.stancePlaceholder)} data-testid="context-pin-stance-input" />
        <p className="text-[11px] text-wc-text-muted" data-testid="context-pin-stance-byte-count">{stanceBytes}/{pinnedStanceMaxBytes ?? "…"} {t(strings.contextCompaction.stanceBytes)}</p>
        {hasStanceError && pinnedStanceMaxBytes !== undefined ? <p className="text-xs text-amber-200" data-testid="context-pin-stance-error" role="status">{t(strings.contextCompaction.stanceLimit, { limit: pinnedStanceMaxBytes })}</p> : null}
        {state?.pinnedStance ? <div className="space-y-1 text-xs text-wc-text-secondary" data-testid="context-pinned-stance"><p>{t(strings.contextCompaction.stancePinned)}</p><blockquote className="break-words border-s-2 border-wc-accent ps-2">{state.pinnedStanceText}</blockquote></div> : <p className="text-xs text-wc-text-muted">{t(strings.contextCompaction.stanceHint)}</p>}
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="secondary" size="md" className="min-h-11" disabled={busy || pinnedStanceMaxBytes === undefined || stance.trim() === "" || stanceError !== null} data-testid="context-pin-stance"
            onClick={() => { void apply({ action: "pin_stance", stance }); }}>{t(strings.contextCompaction.pinStance)}</Button>
          {state?.pinnedStance ? <Button type="button" variant="secondary" size="md" className="min-h-11" disabled={busy} data-testid="context-unpin-stance"
            onClick={() => { setStance(""); void apply({ action: "unpin_stance" }); }}>{t(strings.contextCompaction.unpinStance)}</Button> : null}
          <Button type="button" variant="secondary" size="md" className="min-h-11" data-testid="compaction-handoff-now"
            onClick={() => { useHandoffRequestStore.getState().open(sessionId); }}>{t(strings.contextCompaction.handOffNow)}</Button>
        </div>
      </section>
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
              <ContextBudgetSummary row={row} />
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
