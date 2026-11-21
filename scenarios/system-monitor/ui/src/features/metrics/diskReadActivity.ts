import { apiFetch } from '../../shared/api/apiFetch';
import { isRecord, num, str } from '../../shared/utils/typeGuards';

export interface DiskReadActivityRow {
  pid: number;
  name: string;
  owner: string;
  returnedBytesPerSecond: number;
  storageBytesPerSecond: number;
  intervalSeconds: number;
  observedChildExits: number;
}

export interface DiskReadActivity {
  status: 'measured' | 'partial' | 'failed' | 'not_yet_sampled' | 'unsupported' | string;
  reason?: string;
  platform: string;
  windowSeconds?: number;
  firstObservedAt?: string;
  lastObservedAt?: string;
  firstEnumerated?: number;
  lastEnumerated?: number;
  unreadable?: number;
  permissionDenied?: number;
  disappeared?: number;
  unmatched?: number;
  truncated?: boolean;
  matched?: number;
  active?: number;
  rowsTruncated?: boolean;
  rows: DiskReadActivityRow[];
  storageRows: DiskReadActivityRow[];
  semantics: string;
}

const parseRows = (value: unknown): DiskReadActivityRow[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap(entry => {
    if (!isRecord(entry)) return [];
    const pid = num(entry.pid);
    const name = str(entry.name);
    const owner = str(entry.owner);
    const returnedBytesPerSecond = num(entry.returned_bytes_per_second);
    const storageBytesPerSecond = num(entry.storage_bytes_per_second);
    const intervalSeconds = num(entry.interval_seconds);
    if (pid === undefined || !name || owner === undefined || returnedBytesPerSecond === undefined || storageBytesPerSecond === undefined || intervalSeconds === undefined) return [];
    return [{
      pid,
      name,
      owner,
      returnedBytesPerSecond,
      storageBytesPerSecond,
      intervalSeconds,
      observedChildExits: num(entry.observed_child_exits) ?? 0,
    }];
  });
};

export const parseDiskReadActivity = (value: unknown): DiskReadActivity => {
  if (!isRecord(value) || !str(value.status) || !str(value.platform) || !str(value.semantics)) {
    throw new TypeError('Invalid process read activity response');
  }
  return {
    status: value.status,
    platform: value.platform,
    semantics: value.semantics,
    reason: str(value.reason),
    windowSeconds: num(value.window_seconds),
    firstObservedAt: str(value.first_observed_at),
    lastObservedAt: str(value.last_observed_at),
    firstEnumerated: num(value.first_enumerated),
    lastEnumerated: num(value.last_enumerated),
    unreadable: num(value.unreadable_observations),
    permissionDenied: num(value.permission_denied_observations),
    disappeared: num(value.disappeared_observations),
    unmatched: num(value.unmatched),
    truncated: value.truncated === true,
    matched: num(value.matched),
    active: num(value.active),
    rowsTruncated: value.rows_truncated === true,
    rows: parseRows(value.rows),
    storageRows: parseRows(value.storage_rows),
  };
};

export const fetchDiskReadActivity = (signal?: AbortSignal): Promise<DiskReadActivity> =>
  apiFetch('/metrics/disk/read-activity', parseDiskReadActivity, { signal });
