import { apiFetch } from '../../shared/api/apiFetch';
import type {
  BootHistoryReport,
  ForensicsEnvelope,
  ForensicsSummary,
  MCEReport,
  PstoreReport,
} from './types';

export function fetchForensicsSummary(signal?: AbortSignal): Promise<ForensicsSummary> {
  return apiFetch<ForensicsSummary>('/forensics/summary', { signal });
}

export function fetchPstore(signal?: AbortSignal): Promise<ForensicsEnvelope<PstoreReport>> {
  return apiFetch<ForensicsEnvelope<PstoreReport>>('/forensics/pstore', { signal });
}

export function fetchBootHistory(signal?: AbortSignal): Promise<ForensicsEnvelope<BootHistoryReport>> {
  return apiFetch<ForensicsEnvelope<BootHistoryReport>>('/forensics/boot-history', { signal });
}

export function fetchMCE(signal?: AbortSignal): Promise<ForensicsEnvelope<MCEReport>> {
  return apiFetch<ForensicsEnvelope<MCEReport>>('/forensics/mce', { signal });
}
