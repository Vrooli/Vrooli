import { describe, expect, it } from 'vitest';
import { parseDiskReadActivity } from './diskReadActivity';

describe('parseDiskReadActivity', () => {
  it('keeps typed status, coverage, separate read semantics, and storage rows', () => {
    const parsed = parseDiskReadActivity({
      status: 'partial', platform: 'linux', window_seconds: 2,
      first_enumerated: 20, last_enumerated: 19, permission_denied_observations: 4,
      matched: 12, rows_truncated: true, semantics: 'host-wide sample',
      rows: [{ pid: 1, name: 'reader', owner: 'app', returned_bytes_per_second: 900, storage_bytes_per_second: 300, interval_seconds: 2 }],
      storage_rows: [{ pid: 2, name: 'writer', owner: 'app', returned_bytes_per_second: 800, storage_bytes_per_second: 700, interval_seconds: 2 }],
    });
    expect(parsed.status).toBe('partial');
    expect(parsed.permissionDenied).toBe(4);
    expect(parsed.rows[0]?.returnedBytesPerSecond).toBe(900);
    expect(parsed.storageRows[0]?.storageBytesPerSecond).toBe(700);
    expect(parsed.rowsTruncated).toBe(true);
  });

  it('rejects malformed top-level payloads', () => {
    expect(() => parseDiskReadActivity({ status: 'failed' })).toThrow('Invalid process read activity response');
  });
});
