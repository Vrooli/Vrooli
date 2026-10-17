import { useEffect, useState } from 'react';
import type { BootRecord, LogQueryFilters } from '../types';
import { BootPicker } from './BootPicker';
import { TimeRangePicker } from './TimeRangePicker';
import { parseGrep } from '../utils/parseGrep';

interface FilterBarProps {
  filters: LogQueryFilters;
  units: string[];
  boots: BootRecord[];
  onChange: (patch: Partial<LogQueryFilters>) => void;
  onReset: () => void;
}

const PRIORITY_OPTIONS = [
  { value: '', label: 'Any priority' },
  { value: '0', label: '0 emerg' },
  { value: '1', label: '1 alert' },
  { value: '2', label: '2 crit' },
  { value: '3', label: '3 err' },
  { value: '4', label: '4 warn' },
  { value: '5', label: '5 notice' },
  { value: '6', label: '6 info' },
  { value: '7', label: '7 debug' },
];

export const FilterBar = ({ filters, units, boots, onChange, onReset }: FilterBarProps) => {
  const [grepDraft, setGrepDraft] = useState(filters.grep);
  const [grepError, setGrepError] = useState<string | undefined>(undefined);

  useEffect(() => {
    setGrepDraft(filters.grep);
  }, [filters.grep]);

  const commitGrep = () => {
    const parsed = parseGrep(grepDraft);
    setGrepError(parsed.error);
    if (!parsed.error) onChange({ grep: parsed.pattern });
  };

  const onUnitChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    onChange({ units: value ? [value] : [] });
  };

  return (
    <div
      className="filter-bar card"
      data-sm-style="sm-style-436d528fbb"
    >
      <label
        className="text-xs text-muted"
        data-sm-style="sm-style-e240e6f52c"
      >
        Unit
        <select value={filters.units[0] ?? ''} onChange={onUnitChange}>
          <option value="">All units</option>
          {units.map((u) => (
            <option key={u} value={u}>
              {u}
            </option>
          ))}
        </select>
      </label>

      <label
        className="text-xs text-muted"
        data-sm-style="sm-style-975684b32c"
      >
        Priority
        <select
          value={filters.priority}
          onChange={(e) => { onChange({ priority: e.target.value }); }}
        >
          {PRIORITY_OPTIONS.map((p) => (
            <option key={p.value} value={p.value}>
              {p.label}
            </option>
          ))}
        </select>
      </label>

      <TimeRangePicker
        since={filters.since}
        until={filters.until}
        onChange={(patch) => { onChange(patch); }}
      />

      <BootPicker boots={boots} value={filters.boot} onChange={(boot) => { onChange({ boot }); }} />

      <label
        className="text-xs text-muted"
        data-sm-style="sm-style-616fcc4c59"
      >
        Grep (regex)
        <input
          type="text"
          value={grepDraft}
          placeholder="oom_killer"
          onChange={(e) => { setGrepDraft(e.target.value); }}
          onBlur={commitGrep}
          onKeyDown={(e) => {
            if (e.key === 'Enter') commitGrep();
          }}
        />
        {grepError && (
          <span className="text-xs" data-sm-style="sm-style-6d06f948c5">
            {grepError}
          </span>
        )}
      </label>

      <label className="text-xs text-muted flex-row-center gap-sm">
        <input
          type="checkbox"
          checked={filters.kernel}
          onChange={(e) => { onChange({ kernel: e.target.checked }); }}
        />
        Kernel only
      </label>

      <button type="button" className="header-button" onClick={onReset}>
        Reset
      </button>
    </div>
  );
};
