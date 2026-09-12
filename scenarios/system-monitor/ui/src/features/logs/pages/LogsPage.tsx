import { useEffect, useState } from 'react';
import { fetchBoots, fetchUnits } from '../api';
import { FilterBar } from '../components/FilterBar';
import { LogTable } from '../components/LogTable';
import { RefreshControl } from '../components/RefreshControl';
import { useLogQuery } from '../hooks/useLogQuery';
import { useLogStream } from '../hooks/useLogStream';
import type { BootRecord } from '../types';
import { useTimeRange } from '../../../shared/time/TimeRangeContext';

export const LogsPage = () => {
  const { range, paused: globalPaused } = useTimeRange();
  const query = useLogQuery(range);
  const [units, setUnits] = useState<string[]>([]);
  const [boots, setBoots] = useState<BootRecord[]>([]);

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
    onTick: () => {
      void query.refresh();
    },
  });

  const onPageBack = () => { query.prevPage(); };
  const onPageForward = () => { query.nextPage(); };

  return (
    <section className="logs-page">
      <header
        className="flex-row-center"
        data-sm-style="sm-style-740d1580c7"
      >
        <h2 data-sm-style="sm-style-2a0ca8350a">Logs</h2>
        <RefreshControl
          paused={stream.paused || globalPaused}
          onTogglePause={() => { stream.setPaused(!stream.paused); }}
          onRefresh={() => void query.refresh()}
          isLoading={query.isLoading}
          atTop={stream.atTop}
        />
      </header>

      <FilterBar
        filters={query.filters}
        units={units}
        boots={boots}
        onChange={query.setFilter}
        onReset={query.resetFilters}
      />

      <p className="text-xs text-muted" data-sm-style="sm-style-8251b5082c">
        Shared time range: {range.label}{globalPaused ? ' · live updates paused' : ''}
      </p>

      {/* A failed request and an unavailable source are different facts and
          previously rendered as the same neutral card, leaving the severity
          carried by the sentence alone. */}
      {query.error && (
        <div className="card card--excursion" role="alert">
          <p className="eyebrow card--excursion__label">Log query failed</p>
          <p className="capacity-blind__body">{query.error}</p>
        </div>
      )}

      {!query.available && query.reason && (
        <div className="card capacity-blind" role="status">
          <p className="eyebrow capacity-blind__label">Logs unreadable on this host</p>
          <p className="capacity-blind__body">{query.reason}</p>
        </div>
      )}

      <div data-sm-style="sm-style-323fdcc1e0">
        <LogTable entries={query.entries} onScrollTopChange={stream.setAtTop} />
      </div>

      <div
        className="flex-row-center"
        data-sm-style="sm-style-336c723ae0"
      >
        <button
          type="button"
          className="header-button"
          onClick={onPageBack}
          disabled={query.cursorStack.length === 0}
        >
          ← Newer
        </button>
        <button
          type="button"
          className="header-button"
          onClick={onPageForward}
          disabled={!query.nextCursor}
        >
          Older →
        </button>
        <span className="text-xs text-muted">
          {query.entries.length} entries · page {query.cursorStack.length + 1}
        </span>
      </div>
    </section>
  );
};
