import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { fetchBoots, fetchUnits } from '../api';
import { FilterBar } from '../components/FilterBar';
import { LogTable } from '../components/LogTable';
import { PriorityBadge } from '../components/PriorityBadge';
import { RefreshControl } from '../components/RefreshControl';
import { useLogQuery } from '../hooks/useLogQuery';
import type { LogQueryState } from '../hooks/logQueryReducer';
import { useLogStream } from '../hooks/useLogStream';
import type { BootRecord, LogEntry } from '../types';
import { useTimeRange } from '../../../shared/time/TimeRangeContext';
import '../logs.css';

const JsonEvidenceReader = lazy(() => import('../../../shared/components/JsonEvidenceReader').then(module => ({ default: module.JsonEvidenceReader })));

type RouteSnapshot = {
  range: { since: string; until: string };
  query: LogQueryState;
  selected: LogEntry | null;
  paused: boolean;
  atTop: boolean;
  readerOpen: boolean;
  pageScroll: number;
  readerListScroll: number;
  tableScroll: number;
};

// This local-only page is unmounted by route navigation. Keep its reading
// position in memory for the current app session, never in persistent storage.
let routeSnapshot: RouteSnapshot | undefined;
// Small browser/assistive-technology adjustments to the header do not move
// the reader away from the live edge. Larger page scrolls still pause it.
const PAGE_TOP_TOLERANCE = 24;
export const clearLogsRouteSnapshot = () => { routeSnapshot = undefined; };

export const LogsPage = () => {
  const { range, paused: globalPaused } = useTimeRange();
  const [restored] = useState(() => routeSnapshot?.range.since === range.since && routeSnapshot.range.until === range.until ? routeSnapshot : undefined);
  const [rangeReset] = useState(() => !!routeSnapshot && !restored);
  const [previousPause] = useState(() => routeSnapshot?.paused ?? false);
  const query = useLogQuery(range, restored?.query);
  const [units, setUnits] = useState<string[]>([]);
  const [boots, setBoots] = useState<BootRecord[]>([]);
  const [selected, setSelected] = useState<LogEntry | null>(restored?.selected ?? null);
  const [readerOpen, setReaderOpen] = useState(restored?.readerOpen ?? false);
  const [rawOpen, setRawOpen] = useState(false);
  const [resuming, setResuming] = useState(false);
  const [phone, setPhone] = useState(() => typeof window !== 'undefined' && window.matchMedia('(max-width: 48rem)').matches);
  const { cancel, cancelBackground } = query;
  const wasGlobalPaused = useRef(globalPaused);
  const listScroll = useRef(restored?.readerListScroll ?? 0);
  const scrollListToTop = useRef<(() => void) | null>(null);
  const scrollListToOffset = useRef<((offset: number) => void) | null>(null);
  const readListOffset = useRef<(() => number) | null>(null);
  const pageScroll = useRef(restored?.pageScroll ?? 0);
  const tableScroll = useRef(restored?.tableScroll ?? 0);
  const snapshotRef = useRef<RouteSnapshot | undefined>(undefined);

  useEffect(() => {
    const media = window.matchMedia('(max-width: 48rem)');
    const change = () => { setPhone(media.matches); };
    media.addEventListener?.('change', change);
    return () => { media.removeEventListener?.('change', change); };
  }, []);

  useEffect(() => {
    const onPop = () => {
      if (rawOpen) {
        setRawOpen(false);
        requestAnimationFrame(() => document.querySelector<HTMLButtonElement>('.log-detail__raw')?.focus());
        return;
      }
      if (!readerOpen) return;
      setReaderOpen(false);
      requestAnimationFrame(() => {
        window.scrollTo(0, listScroll.current);
        scrollListToOffset.current?.(tableScroll.current);
        requestAnimationFrame(() => document.querySelector<HTMLButtonElement>('.log-row__select[aria-pressed="true"]')?.focus({ preventScroll: true }));
      });
    };
    window.addEventListener('popstate', onPop);
    return () => { window.removeEventListener('popstate', onPop); };
  }, [readerOpen, rawOpen]);

  const selectEntry = (entry: LogEntry) => {
    tableScroll.current = readListOffset.current?.() ?? tableScroll.current;
    setSelected(entry);
    query.cancel();
    stream.setAtTop(false);
    if (phone) {
      listScroll.current = window.scrollY;
      window.history.pushState({ ...window.history.state, vegaLogReader: true }, '');
      setReaderOpen(true);
    }
  };
  const closeReader = () => {
    if (window.history.state?.vegaLogReader) window.history.back();
    else setReaderOpen(false);
  };
  const openRaw = () => {
    window.history.pushState({ ...window.history.state, vegaLogRaw: true }, '');
    setRawOpen(true);
  };
  const closeRaw = () => {
    if (window.history.state?.vegaLogRaw) window.history.back();
    else setRawOpen(false);
  };

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const [u, b] = await Promise.all([fetchUnits(), fetchBoots()]);
        if (cancelled) return;
        if (u.available && u.units) setUnits(u.units.sort());
        if (b.available && b.boots) setBoots(b.boots);
      } catch {
        /* picker autocomplete is best-effort; logs page still works without it */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const stream = useLogStream({
    enabled: !globalPaused,
    initialPaused: restored?.paused ?? previousPause,
    initialAtTop: restored?.atTop,
    onTick: () => {
      void query.refreshBackground();
    },
  });
  const { atTop, setAtTop } = stream;

  useEffect(() => {
    const wasPaused = wasGlobalPaused.current;
    wasGlobalPaused.current = globalPaused;
    if (globalPaused && !wasPaused) cancel();
    if (!globalPaused && wasPaused && atTop && !stream.paused) {
      setResuming(true);
      void query.refreshBackground().finally(() => { setResuming(false); });
    }
  }, [globalPaused, cancel, atTop, stream.paused, query.refreshBackground]);

  const { filters, cursorStack, currentCursor, nextCursor, entries, isLoading, error, available, reason, loadedPage, retryFilters, retryCursorStack } = query;
  snapshotRef.current = { range: { since: range.since, until: range.until }, query: { filters, cursorStack, currentCursor, nextCursor, entries, isLoading, error, available, reason, loadedPage, retryFilters, retryCursorStack }, selected, paused: stream.paused, atTop, readerOpen, pageScroll: pageScroll.current, readerListScroll: listScroll.current, tableScroll: tableScroll.current };
  useEffect(() => () => {
    if (snapshotRef.current) routeSnapshot = { ...snapshotRef.current, pageScroll: pageScroll.current, readerListScroll: listScroll.current, tableScroll: tableScroll.current };
  }, []);
  useEffect(() => {
    if (!restored) return;
    if (restored.readerOpen && phone && !window.history.state?.vegaLogReader) window.history.pushState({ ...window.history.state, vegaLogReader: true }, '');
    requestAnimationFrame(() => { window.scrollTo(0, restored.pageScroll); });
  }, []); // Restore once when this route is mounted.

  useEffect(() => {
    const onScroll = () => {
      pageScroll.current = window.scrollY;
      if (window.scrollY > PAGE_TOP_TOLERANCE && atTop) {
        cancelBackground();
        setAtTop(false);
      }
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => { window.removeEventListener('scroll', onScroll); };
  }, [cancelBackground, atTop, setAtTop]);

  const onPageBack = () => { query.prevPage(); };
  const onPageForward = () => { query.nextPage(); };
  const showingPreviousQuery = !!query.loadedPage && query.loadedPage.filters !== query.filters;
  const selectedOffPage = !!selected && !query.entries.some(entry => entry === selected || (!!entry.cursor && entry.cursor === selected.cursor));

  return (
    <section className="logs-page">
      <div hidden={rawOpen || (phone && readerOpen)}>
      <header
        className="flex-row-center"
        data-sm-style="sm-style-740d1580c7"
      >
        <h2 data-sm-style="sm-style-2a0ca8350a">Logs</h2>
        <RefreshControl
          paused={stream.paused || globalPaused}
          globalPaused={globalPaused}
          onTogglePause={() => {
            if (!stream.paused) query.cancel();
            else if (stream.atTop && !globalPaused) {
              setResuming(true);
              void query.refreshBackground().finally(() => { setResuming(false); });
            }
            stream.setPaused(!stream.paused);
          }}
          onRefresh={() => void query.refresh()}
          isLoading={query.isLoading}
          atTop={stream.atTop}
        />
      </header>

      <details className="logs-filters" open={!phone || undefined}>
      <summary>Filters</summary>
      <FilterBar
        filters={query.filters}
        units={units}
        boots={boots}
        onChange={query.setFilter}
        onReset={query.resetFilters}
      />
      </details>

      <p className="text-xs text-muted" data-sm-style="sm-style-8251b5082c">
        Shared time range: {range.label}; log query: {query.filters.since || 'any start'} to {query.filters.until || 'any end'}
        {globalPaused ? ' · log updates paused' : ''}
      </p>
      <p className="text-xs text-muted">{query.filters.units.join(', ') || 'All units'} · {query.filters.priority ? `Priority ${query.filters.priority}` : 'Any priority'} · {query.filters.boot || 'All boots'} · {query.filters.grep ? `Grep: ${query.filters.grep}` : 'No grep'} · {query.filters.kernel ? 'Kernel only' : 'All sources'} · Limit {query.filters.limit}</p>
      {rangeReset && <p role="status" className="text-xs text-muted">The shared time range changed while away; the previous log query and selection were reset.</p>}
      {(stream.paused || globalPaused || !stream.atTop || query.cursorStack.length > 0) && <div className="logs-follow-status">
        {(stream.paused || globalPaused || !stream.atTop) && <p role="status" className="text-xs text-muted">
          Log updates paused{query.entries[0]?.timestamp ? ` · latest shown: ${query.entries[0].timestamp}` : ''}. Host collection continues.
        </p>}
        {(!stream.atTop || query.cursorStack.length > 0) && <button type="button" className="header-button" onClick={() => {
          void query.returnToLatest().then(ok => {
            if (ok) { scrollListToTop.current?.(); window.scrollTo({ top: 0, behavior: 'auto' }); stream.setAtTop(true); }
          });
        }}>Return to latest</button>}
      </div>}
      {resuming && !stream.paused && stream.atTop && !globalPaused && <p role="status" className="text-xs text-muted">Fetching latest log entries…</p>}

      {/* A failed request and an unavailable source are different facts and
          previously rendered as the same neutral card, leaving the severity
          carried by the sentence alone. */}
      {query.error && (
        <div className="card card--excursion" role="alert">
          <p className="eyebrow card--excursion__label">Could not load this page</p>
          <p className="capacity-blind__body">{query.error}</p>
          <p className="text-xs text-muted">{query.loadedPage ? 'The last loaded page is retained.' : 'The query has not returned a page.'} Retry the query or change the filters.</p>
          {query.cursorStack.length > 0 && <p className="text-xs text-muted">If the cursor no longer works, Return to latest restarts this interval; your position may change.</p>}
          <button type="button" className="header-button" onClick={query.retryFailed}>{query.retryCursorStack === undefined ? 'Retry query' : query.retryCursorStack.length > query.cursorStack.length ? 'Retry older' : 'Retry newer'}</button>
          {query.loadedPage && <button type="button" className="header-button" onClick={query.restoreLoaded}>Back to last loaded page</button>}
        </div>
      )}

      {!query.available && query.reason && (
        <div className="card capacity-blind" role="status">
          <p className="eyebrow capacity-blind__label">Logs unreadable on this host</p>
          <p className="capacity-blind__body">{query.reason}</p>
          {query.loadedPage && <button type="button" className="header-button" onClick={query.restoreLoaded}>Back to last loaded page</button>}
        </div>
      )}
      {showingPreviousQuery && query.loadedPage && <p role="status" className="text-xs text-muted">
        Showing last loaded page {query.loadedPage.cursorStack.length + 1}: {query.loadedPage.filters.units.join(', ') || 'all units'} · priority {query.loadedPage.filters.priority || 'any'} · {query.loadedPage.filters.since || 'any start'} to {query.loadedPage.filters.until || 'any end'} · boot {query.loadedPage.filters.boot || 'all'} · grep {query.loadedPage.filters.grep || 'none'} · {query.loadedPage.filters.kernel ? 'kernel only' : 'all sources'} · limit {query.loadedPage.filters.limit}. The current query is shown above.
      </p>}
      {query.retryFilters && !showingPreviousQuery && <button type="button" className="header-button" onClick={query.retryPreviousQuery}>Retry previous query</button>}

      <div className="logs-reader-layout" data-sm-style="sm-style-323fdcc1e0">
        <div>
        {(query.entries.length > 0 || (query.available && !query.error && !query.isLoading)) && <LogTable entries={query.entries} initialScrollOffset={restored?.tableScroll} scrollToOffsetRef={scrollListToOffset} readScrollOffsetRef={readListOffset} onScrollOffset={offset => { if (!rawOpen && !(phone && readerOpen)) tableScroll.current = offset; }} onScrollTopChange={(atTop) => {
          if (!atTop && stream.atTop) { query.cancelBackground(); stream.setAtTop(false); }
        }} onSelect={selectEntry} selected={selected} scrollToTopRef={scrollListToTop} />}
        </div>
        {!phone && selected && <div>{selectedOffPage && <p className="text-xs text-muted">Selected entry is from a previously loaded page.</p>}<LogDetail entry={selected} onRaw={openRaw} /></div>}
      </div>

      <div
        className="logs-pagination flex-row-center"
        data-sm-style="sm-style-336c723ae0"
      >
        <button
          type="button"
          className="header-button"
          onClick={onPageBack}
          disabled={showingPreviousQuery || query.isLoading || query.cursorStack.length === 0}
        >
          ← Newer
        </button>
        <button
          type="button"
          className="header-button"
          onClick={onPageForward}
          disabled={showingPreviousQuery || query.isLoading || !query.nextCursor}
        >
          Older →
        </button>
        <span className="text-xs text-muted">
          {query.entries.length} entries returned · page {(query.loadedPage?.cursorStack.length ?? 0) + 1} · total unknown
        </span>
        {!query.isLoading && !showingPreviousQuery && query.cursorStack.length === 0 && <span className="text-xs text-muted">Newer unavailable: already at the first loaded page.</span>}
        {!query.isLoading && !showingPreviousQuery && !query.nextCursor && <span className="text-xs text-muted">Older unavailable: no cursor returned.</span>}
      </div>
      </div>
      {phone && readerOpen && !rawOpen && selected && <LogDetail entry={selected} onBack={closeReader} onRaw={openRaw} />}
      {rawOpen && selected?.raw && <Suspense fallback={<p role="status">Loading JSON reader…</p>}>
        <JsonEvidenceReader title="Journal metadata" value={selected.raw} description={`${selected.hostname || 'Local machine'} · ${selected.timestamp || 'Time not reported'}`} onClose={closeRaw} />
      </Suspense>}
    </section>
  );
};

function LogDetail({ entry, onBack, onRaw }: { entry: LogEntry; onBack?: () => void; onRaw: () => void }) {
  const heading = useRef<HTMLHeadingElement>(null);
  const copyRequest = useRef(0);
  const [copyStatus, setCopyStatus] = useState('');
  const validTime = !!entry.timestamp && Number.isFinite(new Date(entry.timestamp).getTime());
  useEffect(() => { if (onBack) heading.current?.focus(); }, [onBack]);
  useEffect(() => { setCopyStatus(''); return () => { copyRequest.current += 1; }; }, [entry]);
  const copyMessage = async () => {
    if (!entry.message) return;
    const request = ++copyRequest.current;
    setCopyStatus('Copying message…');
    try {
      await navigator.clipboard.writeText(entry.message);
      if (copyRequest.current === request) setCopyStatus('Message copied');
    } catch {
      if (copyRequest.current === request) setCopyStatus('Could not copy message');
    }
  };
  const fields: Array<[string, string]> = [
    ['Machine', entry.hostname || 'Not reported'], ['Unit', entry.unit || entry.userUnit || entry.identifier || 'Not reported'],
    ['Process ID', entry.pid == null ? 'Not reported' : String(entry.pid)], ['Boot', entry.bootId || 'Not reported'],
    ['Timestamp', entry.timestamp || 'Not reported'], ['Priority', String(entry.priority)], ['Cursor', entry.cursor || 'Not reported'],
  ];
  return <article className="log-detail card" aria-label="Selected log entry">
    {onBack && <button type="button" className="header-button" onClick={onBack}>← Back to logs</button>}
    <h2 ref={heading} tabIndex={onBack ? -1 : undefined}>Log entry</h2>
    <div className="log-detail__identity">
      <time dateTime={validTime ? entry.timestamp : undefined}>{validTime ? entry.timestamp : entry.timestamp ? 'Time unrecognized' : 'Time not reported'}</time>
      <PriorityBadge priority={entry.priority} />
    </div>
    <p className="log-detail__unit">{entry.unit || entry.userUnit || entry.identifier || 'System'}</p>
    <h3>Message</h3>
    <pre className="log-detail__message">{entry.message || entry.raw || 'No message reported'}</pre>
    <div className="log-detail__actions">
      {entry.message && <button type="button" className="header-button" onClick={() => { void copyMessage(); }}>Copy message</button>}
      {entry.raw?.trimStart().startsWith('{') && <button type="button" className="header-button log-detail__raw" onClick={onRaw}>Raw metadata</button>}
      <span role="status" className="text-xs text-muted">{copyStatus}</span>
    </div>
    <h3>Entry details</h3>
    <dl>{fields.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
  </article>;
}
