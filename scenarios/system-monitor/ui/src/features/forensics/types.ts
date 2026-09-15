/**
 * TypeScript mirrors of the system-monitor forensics JSON contracts.
 * Source of truth: scenarios/system-monitor/api/internal/handlers/forensics.go
 * and scenarios/system-monitor/api/internal/services/forensics/*.go
 */

export interface ForensicsEnvelope<T> {
  available: boolean;
  reason?: string;
  data?: T;
  generatedAt: string;
}

export type PstoreKind = 'dmesg' | 'console' | 'pmsg' | 'ftrace' | 'unknown';

export interface PstoreEntry {
  name: string;
  kind: PstoreKind | string;
  size: number;
  modified: string;
}

export interface PstoreReport {
  path: string;
  entries: PstoreEntry[];
}

export interface BootEntry {
  index: number;
  bootId: string;
  firstEntry: string;
  lastEntry: string;
  clean: boolean;
  reason?: string;
}

export interface BootHistoryReport {
  boots: BootEntry[];
}

export interface MCEReport {
  window: string;
  uncorrected: number;
  corrected: number;
  rawSummary?: string;
}

export interface ForensicsRelevantCheck {
  checkId: string;
  status: string;
  message?: string;
  category?: string;
  details?: Record<string, unknown>;
  lastRunAt?: string;
}

export interface AutohealEnvelope {
  available: boolean;
  reason?: string;
  checks?: ForensicsRelevantCheck[];
}

export interface ForensicsSummary {
  generatedAt: string;
  pstore: ForensicsEnvelope<PstoreReport>;
  bootHistory: ForensicsEnvelope<BootHistoryReport>;
  mce: ForensicsEnvelope<MCEReport>;
  autoheal: AutohealEnvelope;
}
