import type {
  ReportIssueConsoleLogEntry,
  ReportIssueNetworkEntry,
} from '@/services/api';

export interface ReportConsoleEntry {
  display: string;
  severity: ConsoleSeverity;
  payload: ReportIssueConsoleLogEntry;
  timestamp: string;
  source: string;
  body: string;
}

export interface ReportNetworkEntry {
  display: string;
  payload: ReportIssueNetworkEntry;
  timestamp: string;
  method: string;
  statusLabel: string;
  durationLabel: string | null;
  errorText: string | null;
}

export interface ReportAppLogStream {
  key: string;
  label: string;
  type: 'lifecycle' | 'background';
  lines: string[];
  total: number;
  command?: string;
}

export interface ReportHealthCheckEntry {
  id: string;
  name: string;
  status: 'pass' | 'warn' | 'fail';
  endpoint: string | null;
  latencyMs: number | null;
  message: string | null;
  code: string | null;
  response: string | null;
}

type ConsoleSeverity = 'error' | 'warn' | 'info' | 'log' | 'debug' | 'trace';

export type { ConsoleSeverity };
