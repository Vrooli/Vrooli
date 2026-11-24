import { createHash } from 'node:crypto';
import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';

export const BROWSER_PROCESS_TREE_CONTRACT = 'bas.browser-process-tree.v2';
export const BROWSER_PROCESS_TREE_FILE = 'performance.process-tree.json';
const SAMPLE_INTERVAL_MS = 100;
const MIN_DURATION_MS = 500;
const MAX_SAMPLE_GAP_MS = 500;
const MAX_DURATION_MS = 120_000;

export interface BrowserProcessTreeArtifact {
  contract: typeof BROWSER_PROCESS_TREE_CONTRACT;
  platform: string;
  cpu_clock_ticks_per_second: null;
  cpu_clock_source: 'unavailable-from-current-owner';
  process_lifetime_accounting: 'sampled-snapshot-only';
  process_lifetime_coverage_complete: false;
  process_lifetime_sample_interval_ms: typeof SAMPLE_INTERVAL_MS;
  capture_id: string;
  session_id: string;
  lease_id_sha256: string;
  scope: 'bas-driver-plus-exclusive-broker-owned-browser-process-tree';
  status: 'measured' | 'unknown';
  reason?: string;
  root_pid: number | null;
  root_start_ticks: string | null;
  host_boot_id: string | null;
  process_identity_set_sha256: string | null;
  sample_count: number;
  process_count: number | null;
  interval_ms: number | null;
  window_start_monotonic_ms: number | null;
  window_end_monotonic_ms: number | null;
  cpu_time_ms: number | null;
  cpu_percent_avg: number | null;
  peak_rss_bytes: number | null;
}

interface ProcessPoint {
  pid: number;
  ppid: number;
  startTicks: string;
}

interface TreePoint {
  atMs: number;
  processes: Map<number, ProcessPoint>;
}

interface ProcFS {
  readFile(file: string): string;
  readDir(directory: string): string[];
}

const hostProc: ProcFS = {
  readFile: (file) => readFileSync(file, 'utf8'),
  readDir: (directory) => readdirSync(directory),
};

export class BrowserProcessTreeCapture {
  private readonly points: TreePoint[] = [];
  private readonly rootPid: number | null;
  private readonly rootStartTicks: string | null;
  private readonly bootId: string | null;
  private readonly leaseDigest: string;
  private readonly timer?: NodeJS.Timeout;
  private failure?: string;
  private finished = false;

  private constructor(
    private readonly captureId: string,
    private readonly sessionId: string,
    private readonly proc: ProcFS,
    private readonly platform: string,
    private readonly now: () => number,
    rootPid: number | null,
    rootStartTicks: string | null,
    bootId: string | null,
    leaseId: string,
    private readonly isExclusive: () => boolean,
    private readonly isLeaseActive: () => boolean
  ) {
    this.rootPid = rootPid;
    this.rootStartTicks = rootStartTicks;
    this.bootId = bootId;
    this.leaseDigest = createHash('sha256').update(leaseId).digest('hex');
    this.takePoint();
    this.timer = setInterval(() => this.takePoint(), SAMPLE_INTERVAL_MS);
    this.timer.unref?.();
  }

  static start(options: {
    captureId: string;
    sessionId: string;
    leaseId: string;
    rootPid: number;
    isExclusive: () => boolean;
    proc?: ProcFS;
    platform?: string;
    now?: () => number;
    isLeaseActive?: () => boolean;
  }): BrowserProcessTreeCapture {
    const proc = options.proc ?? hostProc;
    const platform = options.platform ?? process.platform;
    let rootPid: number | null = null;
    let rootStartTicks: string | null = null;
    let bootId: string | null = null;
    const isExclusive = options.isExclusive;
    let startupFailure: string | undefined;
    try {
      rootPid =
        Number.isSafeInteger(options.rootPid) && options.rootPid > 0 ? options.rootPid : null;
      if (platform !== 'linux') startupFailure = `process-tree sampler unsupported on ${platform}`;
      else if (rootPid === null) startupFailure = 'BAS browser process identity is unavailable';
      else {
        const stat = parseStat(proc.readFile(path.join('/proc', String(rootPid), 'stat')));
        rootStartTicks = stat.startTicks;
        bootId = proc.readFile('/proc/sys/kernel/random/boot_id').trim();
        if (!bootId) startupFailure = 'host process boot identity is unavailable';
      }
    } catch {
      startupFailure = 'BAS browser process start identity could not be read';
    }
    const capture = new BrowserProcessTreeCapture(
      options.captureId,
      options.sessionId,
      proc,
      platform,
      options.now ?? (() => performance.now()),
      rootPid,
      rootStartTicks,
      bootId,
      options.leaseId,
      isExclusive,
      options.isLeaseActive ?? (() => true)
    );
    capture.failure = startupFailure;
    return capture;
  }

  finish(): BrowserProcessTreeArtifact {
    if (!this.finished) {
      this.takePoint();
      if (this.timer) clearInterval(this.timer);
      this.finished = true;
    }
    const base: BrowserProcessTreeArtifact = {
      contract: BROWSER_PROCESS_TREE_CONTRACT,
      platform: this.platform,
      cpu_clock_ticks_per_second: null,
      cpu_clock_source: 'unavailable-from-current-owner',
      process_lifetime_accounting: 'sampled-snapshot-only',
      process_lifetime_coverage_complete: false,
      process_lifetime_sample_interval_ms: SAMPLE_INTERVAL_MS,
      capture_id: this.captureId,
      session_id: this.sessionId,
      lease_id_sha256: this.leaseDigest,
      scope: 'bas-driver-plus-exclusive-broker-owned-browser-process-tree',
      status: 'unknown',
      root_pid: this.rootPid,
      root_start_ticks: this.rootStartTicks,
      host_boot_id: this.bootId,
      process_identity_set_sha256: null,
      sample_count: this.points.length,
      process_count: null,
      interval_ms: null,
      window_start_monotonic_ms: this.points[0]?.atMs ?? null,
      window_end_monotonic_ms: this.points[this.points.length - 1]?.atMs ?? null,
      cpu_time_ms: null,
      cpu_percent_avg: null,
      peak_rss_bytes: null,
    };
    const reason = this.unknownReason();
    if (reason) return { ...base, reason };
    const first = this.points[0];
    const last = this.points[this.points.length - 1];
    if (!first || !last) return { ...base, reason: 'process-tree sample interval is incomplete' };
    const elapsed = last.atMs - first.atMs;
    const firstProcesses = first.processes;
    for (const point of this.points) {
      if (!sameIdentitySet(firstProcesses, point.processes)) {
        return {
          ...base,
          reason: 'process tree membership or start identities changed during capture',
        };
      }
      for (const [pid, process] of point.processes) {
        const initial = firstProcesses.get(pid);
        if (!initial || process.startTicks !== initial.startTicks) {
          return {
            ...base,
            reason: `process ${pid} identity or CPU counter changed during capture`,
          };
        }
      }
    }
    const identityList = [...firstProcesses.values()]
      .map((process) => `${process.pid}:${process.startTicks}`)
      .sort()
      .join('\n');
    return {
      ...base,
      process_identity_set_sha256: createHash('sha256').update(identityList).digest('hex'),
      process_count: firstProcesses.size,
      interval_ms: elapsed,
      window_start_monotonic_ms: first.atMs,
      window_end_monotonic_ms: last.atMs,
      reason: 'clock rate is unavailable and snapshot polling cannot account for short-lived process lifetimes',
    };
  }

  private takePoint(): void {
    if (this.finished || this.failure || this.platform !== 'linux' || this.rootPid === null) return;
    try {
      if (!this.isLeaseActive()) {
        this.failure = 'BAS session lease was released or transferred during capture';
        return;
      }
      if (!this.isExclusive()) {
        this.failure = 'shared browser process cannot be attributed to one BAS session';
        return;
      }
      const processes = readTree(this.proc, this.rootPid);
      const root = processes.get(this.rootPid);
      if (!root || root.startTicks !== this.rootStartTicks) {
        this.failure = 'BAS browser root process was replaced or is missing';
        return;
      }
      const atMs = this.now();
      const previous = this.points[this.points.length - 1];
      const first = this.points[0];
      if (first && atMs - first.atMs > MAX_DURATION_MS) {
        this.failure = 'process-tree capture exceeded the 120-second measurement bound';
        if (this.timer) clearInterval(this.timer);
        return;
      }
      if (previous && atMs === previous.atMs) return;
      if (previous && atMs < previous.atMs) throw new Error('monotonic clock moved backwards');
      this.points.push({ atMs, processes });
    } catch {
      this.failure = 'process-tree coverage is incomplete or a process disappeared during sampling';
    }
  }

  private unknownReason(): string | undefined {
    if (this.failure) return this.failure;
    if (!this.captureId || !this.sessionId || !this.leaseDigest)
      return 'capture, session, or lease binding is missing';
    if (!this.rootPid || !this.rootStartTicks || !this.bootId)
      return 'browser process start identity is unavailable';
    if (this.points.length < 3) return 'process-tree interval has fewer than three samples';
    const first = this.points[0];
    const last = this.points[this.points.length - 1];
    if (!first || !last) return 'process-tree interval is incomplete';
    const elapsed = last.atMs - first.atMs;
    if (!Number.isFinite(elapsed) || elapsed < MIN_DURATION_MS)
      return 'process-tree interval is shorter than 500 ms';
    for (let i = 1; i < this.points.length; i++) {
      const current = this.points[i];
      const previous = this.points[i - 1];
      if (!current || !previous) return 'process-tree interval is incomplete';
      const gap = current.atMs - previous.atMs;
      if (!Number.isFinite(gap) || gap <= 0 || gap > MAX_SAMPLE_GAP_MS)
        return 'process-tree sample interval is invalid or incomplete';
    }
    return undefined;
  }
}

function readTree(proc: ProcFS, rootPid: number): Map<number, ProcessPoint> {
  const result = new Map<number, ProcessPoint>();
  const queue: Array<{ pid: number; expectedParent?: number }> = [{ pid: rootPid }];
  while (queue.length > 0) {
    const { pid, expectedParent } = queue.shift()!;
    if (result.has(pid)) continue;
    const base = path.join('/proc', String(pid));
    const stat = parseStat(proc.readFile(path.join(base, 'stat')));
    if (expectedParent !== undefined && stat.ppid !== expectedParent) {
      throw new Error('child process parent identity changed during tree walk');
    }
    result.set(pid, {
      pid,
      ppid: stat.ppid,
      startTicks: stat.startTicks,
    });
    const taskDir = path.join(base, 'task');
    const threads = proc.readDir(taskDir).filter((entry) => /^\d+$/.test(entry));
    if (threads.length === 0) throw new Error('process has no readable task entries');
    const children = new Set<number>();
    for (const threadId of threads) {
      const raw = proc.readFile(path.join(taskDir, threadId, 'children')).trim();
      for (const item of raw ? raw.split(/\s+/) : []) {
        const childPid = Number(item);
        if (!Number.isSafeInteger(childPid) || childPid <= 0) throw new Error('invalid child pid');
        children.add(childPid);
      }
    }
    queue.push(...[...children].map((childPid) => ({ pid: childPid, expectedParent: pid })));
  }
  return result;
}

function parseStat(raw: string): { ppid: number; startTicks: string } {
  const end = raw.lastIndexOf(')');
  if (end < 0) throw new Error('malformed proc stat');
  const fields = raw
    .slice(end + 1)
    .trim()
    .split(/\s+/);
  if (fields.length < 20) throw new Error('short proc stat');
  const ppid = Number(fields[1]);
  const startTicks = fields[19];
  if (
    !Number.isSafeInteger(ppid) ||
    typeof startTicks !== 'string' ||
    !/^\d+$/.test(startTicks)
  ) {
    throw new Error('invalid proc stat fields');
  }
  return { ppid, startTicks };
}

function sameIdentitySet(a: Map<number, ProcessPoint>, b: Map<number, ProcessPoint>): boolean {
  if (a.size !== b.size) return false;
  for (const [pid, process] of a) if (b.get(pid)?.startTicks !== process.startTicks) return false;
  return true;
}
