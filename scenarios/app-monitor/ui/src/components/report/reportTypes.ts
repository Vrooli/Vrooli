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

export interface ReportAppStatusSnapshot {
  appId: string;
  scenario: string;
  statusLabel: string;
  severity: 'ok' | 'warn' | 'error';
  runtime: string | null;
  processCount: number | null;
  details: string[];
  capturedAt: string | null;
}

export type ReportCaptureType = 'page' | 'element';

export interface ReportCaptureMetadata {
  selector?: string | null;
  tagName?: string | null;
  elementId?: string | null;
  classes?: string[];
  label?: string | null;
  ariaDescription?: string | null;
  title?: string | null;
  role?: string | null;
  text?: string | null;
  ancestorIndex?: number | null;
  ancestorCount?: number | null;
  ancestorTrail?: Array<string | null>;
  boundingBox?: { x: number; y: number; width: number; height: number } | null;
}

interface ReportCaptureBase {
  id: string;
  type: ReportCaptureType;
  width: number;
  height: number;
  data: string;
  createdAt: number;
  filename?: string | null;
  clip?: { x: number; y: number; width: number; height: number } | null;
  mode?: string | null;
}

export interface ReportPageCapture extends ReportCaptureBase {
  type: 'page';
}

export interface ReportElementCapture extends ReportCaptureBase {
  type: 'element';
  note: string;
  metadata: ReportCaptureMetadata;
}

export type ReportCapture = ReportPageCapture | ReportElementCapture;

type ConsoleSeverity = 'error' | 'warn' | 'info' | 'log' | 'debug' | 'trace';

export type { ConsoleSeverity };
