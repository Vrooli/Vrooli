// Shared HistoryWindow type — mirrors stats.HistoryWindow on the API side.
// Every operational-stats response carries one of these so the UI can refuse
// to render misleading aggregates over thin samples.

export interface HistoryWindow {
  earliest_event_at: string;
  history_days: number;
  has_history: boolean;
  min_sample_meaningful: number;
}

export interface HistoryCoverage {
  historyFloor: string;
  outsideHistoryRunCount: number;
}
