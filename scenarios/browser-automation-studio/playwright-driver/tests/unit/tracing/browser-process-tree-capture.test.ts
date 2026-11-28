import { BrowserProcessTreeCapture } from '../../../src/tracing/browser-process-tree-capture';

function procFixture() {
  const files = new Map<string, string>();
  const dirs = new Map<string, string[]>();
  const addProcess = (pid: number, ppid: number, start: number, cpu: number, rssKib: number) => {
    const fields = Array.from({ length: 19 }, () => '0');
    fields[0] = String(ppid);
    fields[10] = String(cpu);
    fields[11] = '0';
    fields[18] = String(start);
    files.set(`/proc/${pid}/stat`, `${pid} (chromium process) S ${fields.join(' ')}\n`);
    files.set(`/proc/${pid}/status`, `Name:\tchromium\nVmRSS:\t${rssKib} kB\n`);
    dirs.set(`/proc/${pid}/task`, [String(pid)]);
    dirs.set(`/proc/${pid}/task/${pid}`, []);
    files.set(`/proc/${pid}/task/${pid}/children`, '');
  };
  addProcess(10, 1, 100, 10, 100);
  addProcess(11, 10, 200, 20, 50);
  files.set('/proc/10/task/10/children', '11');
  files.set('/proc/sys/kernel/random/boot_id', 'host-boot-id\n');
  const proc = {
    readFile: (file: string) => {
      const value = files.get(file);
      if (value === undefined) throw new Error(`missing ${file}`);
      return value;
    },
    readDir: (directory: string) => {
      const value = dirs.get(directory);
      if (value === undefined) throw new Error(`missing ${directory}`);
      return value;
    },
  };
  return { files, dirs, proc };
}

function stat(pid: number, ppid: number, start: number, cpu: number): string {
  const fields = Array.from({ length: 19 }, () => '0');
  fields[0] = String(ppid);
  fields[10] = String(cpu);
  fields[11] = '0';
  fields[18] = String(start);
  return `${pid} (chromium process) S ${fields.join(' ')}\n`;
}

describe('BrowserProcessTreeCapture', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  test('emits a v2 qualification contract but leaves CPU and RSS unknown without clock/lifetime proof', () => {
    const { proc, files } = procFixture();
    const capture = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-1',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    for (let tick = 1; tick <= 6; tick++) {
      files.set('/proc/10/stat', stat(10, 1, 100, 10 + tick));
      files.set('/proc/11/stat', stat(11, 10, 200, 20 + tick));
      jest.advanceTimersByTime(100);
    }
    const result = capture.finish();
    expect(result.status).toBe('unknown');
    expect(result.contract).toBe('bas.browser-process-tree.v2');
    expect(result.platform).toBe('linux');
    expect(result.cpu_clock_ticks_per_second).toBeNull();
    expect(result.cpu_clock_source).toBe('unavailable-from-current-owner');
    expect(result.process_lifetime_accounting).toBe('sampled-snapshot-only');
    expect(result.process_lifetime_coverage_complete).toBe(false);
    expect(result.process_lifetime_sample_interval_ms).toBe(100);
    expect(result.reason).toContain('short-lived process lifetimes');
    expect(result.capture_id).toBe('capture-1');
    expect(result.session_id).toBe('session-1');
    expect(result.lease_id_sha256).toMatch(/^[a-f0-9]{64}$/);
    expect(result.root_pid).toBe(10);
    expect(result.root_start_ticks).toBe('100');
    expect(result.process_identity_set_sha256).toMatch(/^[a-f0-9]{64}$/);
    expect(result.process_count).toBe(2);
    expect(result.cpu_time_ms).toBeNull();
    expect(result.cpu_percent_avg).toBeNull();
    expect(result.peak_rss_bytes).toBeNull();
  });

  test('unknowns shared browser ownership, short intervals, PID reuse, and incomplete descendants', () => {
    const shared = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => false,
      captureId: 'capture-1',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: procFixture().proc,
      platform: 'linux',
      now: () => Date.now(),
    }).finish();
    expect(shared.status).toBe('unknown');
    expect(shared.reason).toContain('shared browser process');

    const short = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-2',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: procFixture().proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    jest.advanceTimersByTime(200);
    expect(short.finish().reason).toContain('shorter than 500 ms');

    const reusedFixture = procFixture();
    const reused = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-3',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: reusedFixture.proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    jest.advanceTimersByTime(300);
    reusedFixture.files.set('/proc/11/stat', stat(11, 10, 999, 22));
    jest.advanceTimersByTime(300);
    expect(reused.finish().status).toBe('unknown');

    const missingFixture = procFixture();
    const missing = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-4',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: missingFixture.proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    jest.advanceTimersByTime(100);
    missingFixture.files.delete('/proc/10/task/10/children');
    jest.advanceTimersByTime(100);
    const incomplete = missing.finish();
    expect(incomplete.status).toBe('unknown');
    expect(incomplete.reason).toContain('coverage is incomplete');
  });

  test('v2 does not claim coverage of a child that exists only between snapshots', () => {
    const { proc, files, dirs } = procFixture();
    const capture = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-transient-child',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    const transientFields = Array.from({ length: 19 }, () => '0');
    transientFields[0] = '10';
    transientFields[10] = '500';
    transientFields[18] = '300';
    files.set('/proc/12/stat', `12 (short child) S ${transientFields.join(' ')}\n`);
    files.set('/proc/12/status', 'VmRSS:\t9000 kB\n');
    dirs.set('/proc/12/task', ['12']);
    dirs.set('/proc/12/task/12', []);
    files.set('/proc/12/task/12/children', '');
    files.set('/proc/10/task/10/children', '11 12');
    // It exits wholly before the next 100ms sampler pass.
    for (const path of ['/proc/12/stat', '/proc/12/status', '/proc/12/task/12/children']) files.delete(path);
    dirs.delete('/proc/12/task/12');
    dirs.delete('/proc/12/task');
    files.set('/proc/10/task/10/children', '11');
    jest.advanceTimersByTime(600);
    const result = capture.finish();
    expect(result.status).toBe('unknown');
    expect(result.process_lifetime_coverage_complete).toBe(false);
    expect(result.process_lifetime_accounting).toBe('sampled-snapshot-only');
    expect(result.process_count).toBe(2);
    expect(result.cpu_time_ms).toBeNull();
    expect(result.peak_rss_bytes).toBeNull();
  });

  test('fails closed on parent mismatch, child churn, lease revocation, and sampling gaps', () => {
    const mismatched = procFixture();
    const parentMismatch = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-parent',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: mismatched.proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    mismatched.files.set('/proc/11/stat', stat(11, 99, 200, 20));
    jest.advanceTimersByTime(100);
    expect(parentMismatch.finish().status).toBe('unknown');
    expect(parentMismatch.finish().reason).toContain('coverage is incomplete');

    const churn = procFixture();
    const childChurn = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-churn',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: churn.proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    jest.advanceTimersByTime(100);
    churn.files.set('/proc/10/task/10/children', '');
    jest.advanceTimersByTime(100);
    churn.files.set('/proc/10/task/10/children', '11');
    jest.advanceTimersByTime(400);
    expect(childChurn.finish().status).toBe('unknown');
    expect(childChurn.finish().reason).toContain('membership');

    let leaseActive = true;
    const revoked = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      isLeaseActive: () => leaseActive,
      captureId: 'capture-revoked',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: procFixture().proc,
      platform: 'linux',
      now: () => Date.now(),
    });
    jest.advanceTimersByTime(100);
    leaseActive = false;
    jest.advanceTimersByTime(100);
    leaseActive = true;
    jest.advanceTimersByTime(500);
    expect(revoked.finish().status).toBe('unknown');
    expect(revoked.finish().reason).toContain('session lease was released');

    const times = [1000, 1700, 1800, 1900];
    const gap = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-gap',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: procFixture().proc,
      platform: 'linux',
      now: () => times.shift() ?? 2000,
    });
    jest.advanceTimersByTime(300);
    expect(gap.finish().status).toBe('unknown');
    expect(gap.finish().reason).toContain('sample interval');
  });

  test('projects unsupported platforms as unknown with zero live dependency', () => {
    const result = BrowserProcessTreeCapture.start({
      rootPid: 10,
      isExclusive: () => true,
      captureId: 'capture-5',
      sessionId: 'session-1',
      leaseId: 'lease-1',
      proc: procFixture().proc,
      platform: 'darwin',
    }).finish();
    expect(result.status).toBe('unknown');
    expect(result.reason).toContain('unsupported on darwin');
    expect(result.cpu_time_ms).toBeNull();
    expect(result.peak_rss_bytes).toBeNull();
  });
});
