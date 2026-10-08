import { createContext, useContext, useMemo, useState, type ReactNode } from 'react';

export interface TimeRange {
  key: string;
  label: string;
  seconds: number;
  since: string;
  until: string;
}

export const TIME_RANGE_OPTIONS: readonly TimeRange[] = [
  { key: '15m', label: '15 minutes', seconds: 15 * 60, since: '15m ago', until: 'now' },
  { key: '1h', label: '1 hour', seconds: 60 * 60, since: '1h ago', until: 'now' },
  { key: '6h', label: '6 hours', seconds: 6 * 60 * 60, since: '6h ago', until: 'now' },
  { key: '24h', label: '24 hours', seconds: 24 * 60 * 60, since: '24h ago', until: 'now' },
];

interface TimeRangeContextValue {
  range: TimeRange;
  setRange: (key: string) => void;
  paused: boolean;
  setPaused: (paused: boolean) => void;
}

const defaultTimeRange: TimeRange = (() => {
  const option = TIME_RANGE_OPTIONS.find((candidate) => candidate.key === '1h');
  if (!option) throw new Error('Time range options must include the default 1h window');
  return option;
})();
const TimeRangeContext = createContext<TimeRangeContextValue>({
  range: defaultTimeRange,
  setRange: () => undefined,
  paused: false,
  setPaused: () => undefined,
});

export function TimeRangeProvider({ children }: { children: ReactNode }) {
  const [range, setRangeValue] = useState<TimeRange>(defaultTimeRange);
  const [paused, setPaused] = useState(false);
  const value = useMemo<TimeRangeContextValue>(() => ({
    range,
    setRange: (key) => {
      const next = TIME_RANGE_OPTIONS.find((option) => option.key === key);
      if (next) setRangeValue(next);
    },
    paused,
    setPaused,
  }), [paused, range]);
  return <TimeRangeContext.Provider value={value}>{children}</TimeRangeContext.Provider>;
}

export function useTimeRange(): TimeRangeContextValue {
  return useContext(TimeRangeContext);
}
