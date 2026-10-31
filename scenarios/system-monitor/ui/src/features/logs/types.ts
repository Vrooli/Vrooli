/**
 * TypeScript mirrors of the system-monitor logs JSON contracts.
 * Source: scenarios/system-monitor/api/internal/handlers/logs.go and
 *         scenarios/system-monitor/api/internal/services/journal/reader.go
 */

export interface LogEntry {
  timestamp: string;
  realtime: number;
  priority: number;
  unit?: string;
  userUnit?: string;
  identifier?: string;
  hostname?: string;
  pid?: number;
  bootId?: string;
  cursor?: string;
  message: string;
  raw?: string;
}

export interface BootRecord {
  index: number;
  bootId: string;
  firstEntry: string;
  lastEntry: string;
}

export type LogDirection = 'forward' | 'reverse';

export interface LogsResponse {
  available: boolean;
  reason?: string;
  entries?: LogEntry[];
  nextCursor?: string;
  direction?: LogDirection;
  limit?: number;
  generatedAt: string;
}

export interface UnitsResponse {
  available: boolean;
  reason?: string;
  units?: string[];
  generatedAt: string;
}

export interface BootsResponse {
  available: boolean;
  reason?: string;
  boots?: BootRecord[];
  generatedAt: string;
}

/**
 * User-visible filter set. Maps 1:1 onto the QueryOpts the backend accepts,
 * minus pagination knobs (cursor/limit/direction live in the reducer state).
 */
export interface LogQueryFilters {
  units: string[];
  kernel: boolean;
  since: string;
  until: string;
  priority: string;
  grep: string;
  boot: string;
  limit: number;
}

export const DEFAULT_LIMIT = 200;
export const MAX_LIMIT = 500;

export const emptyFilters: LogQueryFilters = {
  units: [],
  kernel: false,
  since: '',
  until: '',
  priority: '',
  grep: '',
  boot: '',
  limit: DEFAULT_LIMIT,
};
