import { PerfCollector } from '../../../src/performance';
import { createTestConfig } from '../../helpers/test-config';

describe('PerfCollector', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2025-01-01T00:00:00Z'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('records frames and maintains a ring buffer', () => {
    const collector = new PerfCollector('session-1', {
      bufferSize: 2,
      logSummaryInterval: 0,
      targetFps: 10,
    });

    collector.recordFrame({
      captureMs: 10,
      compareMs: 2,
      wsSendMs: 3,
      frameBytes: 100,
      skipped: false,
    });
    collector.recordFrame({
      captureMs: 20,
      compareMs: 4,
      wsSendMs: 6,
      frameBytes: 200,
      skipped: false,
    });
    collector.recordSkipped(5, 1);

    expect(collector.getFrameCount()).toBe(3);
    expect(collector.getSequenceNum()).toBe(3);

    const recent = collector.getRecentFrames(10);
    expect(recent).toHaveLength(2);
    expect(recent[0]?.frame_id).toBe('session-1-2');
    expect(recent[1]?.skipped).toBe(true);
  });

  it('builds a valid frame header with length prefix', () => {
    const collector = new PerfCollector('session-2', {
      bufferSize: 10,
      logSummaryInterval: 0,
      targetFps: 30,
    });

    const headerBuffer = collector.buildFrameHeader(12, 3, 4, 256);
    const headerLength = headerBuffer.readUInt32BE(0);
    const headerJson = headerBuffer.subarray(4, 4 + headerLength).toString('utf8');
    const header = JSON.parse(headerJson) as { frame_id: string; capture_ms: number; frame_bytes: number };

    expect(header.frame_id).toBe('session-2-1');
    expect(header.capture_ms).toBe(12);
    expect(header.frame_bytes).toBe(256);
  });

  it('respects log summary interval settings', () => {
    const collector = new PerfCollector('session-3', {
      bufferSize: 5,
      logSummaryInterval: 2,
      targetFps: 10,
    });

    collector.recordFrame({ captureMs: 10, compareMs: 1, wsSendMs: 1, frameBytes: 1, skipped: false });
    expect(collector.shouldLogSummary()).toBe(false);

    collector.recordFrame({ captureMs: 10, compareMs: 1, wsSendMs: 1, frameBytes: 1, skipped: false });
    expect(collector.shouldLogSummary()).toBe(true);
  });

  it('returns empty stats when no frames recorded', () => {
    const collector = new PerfCollector('session-empty', {
      bufferSize: 5,
      logSummaryInterval: 0,
      targetFps: 15,
    });

    jest.setSystemTime(new Date('2025-01-01T00:00:02Z'));
    const stats = collector.getAggregatedStats();

    expect(stats.frame_count).toBe(0);
    expect(stats.skipped_count).toBe(0);
    expect(stats.actual_fps).toBe(0);
    expect(stats.target_fps).toBe(15);
    expect(stats.primary_bottleneck).toBe('none');
    expect(stats.bottleneck_description).toContain('No frames recorded');
  });

  it('identifies capture bottleneck when capture p90 exceeds target threshold', () => {
    const collector = new PerfCollector('session-capture', {
      bufferSize: 5,
      logSummaryInterval: 0,
      targetFps: 10,
    });

    collector.recordFrame({ captureMs: 90, compareMs: 1, wsSendMs: 2, frameBytes: 100, skipped: false });
    collector.recordFrame({ captureMs: 95, compareMs: 1, wsSendMs: 2, frameBytes: 100, skipped: false });
    collector.recordFrame({ captureMs: 100, compareMs: 1, wsSendMs: 2, frameBytes: 100, skipped: false });

    jest.setSystemTime(new Date('2025-01-01T00:00:01Z'));
    const stats = collector.getAggregatedStats();

    expect(stats.primary_bottleneck).toBe('capture');
    expect(stats.bottleneck_description).toContain('Screenshot capture');
  });

  it('identifies slow capture when p50 exceeds 100ms without violating p90 threshold', () => {
    const collector = new PerfCollector('session-capture-slow', {
      bufferSize: 5,
      logSummaryInterval: 0,
      targetFps: 2,
    });

    collector.recordFrame({ captureMs: 120, compareMs: 2, wsSendMs: 10, frameBytes: 100, skipped: false });
    collector.recordFrame({ captureMs: 130, compareMs: 2, wsSendMs: 10, frameBytes: 100, skipped: false });
    collector.recordFrame({ captureMs: 140, compareMs: 2, wsSendMs: 10, frameBytes: 100, skipped: false });

    jest.setSystemTime(new Date('2025-01-01T00:00:02Z'));
    const stats = collector.getAggregatedStats();

    expect(stats.primary_bottleneck).toBe('capture');
    expect(stats.bottleneck_description).toContain('>100ms');
  });

  it('identifies network bottleneck when e2e latency is high', () => {
    const collector = new PerfCollector('session-network', {
      bufferSize: 5,
      logSummaryInterval: 0,
      targetFps: 10,
    });

    collector.recordFrame({ captureMs: 20, compareMs: 5, wsSendMs: 200, frameBytes: 150, skipped: false });
    collector.recordFrame({ captureMs: 18, compareMs: 4, wsSendMs: 210, frameBytes: 150, skipped: false });
    collector.recordFrame({ captureMs: 22, compareMs: 6, wsSendMs: 220, frameBytes: 150, skipped: false });

    jest.setSystemTime(new Date('2025-01-01T00:00:01Z'));
    const stats = collector.getAggregatedStats();

    expect(stats.primary_bottleneck).toBe('network');
    expect(stats.bottleneck_description).toContain('network');
  });

  it('returns no bottleneck when timings are within bounds', () => {
    const collector = new PerfCollector('session-none', {
      bufferSize: 5,
      logSummaryInterval: 0,
      targetFps: 10,
    });

    collector.recordFrame({ captureMs: 30, compareMs: 2, wsSendMs: 3, frameBytes: 100, skipped: false });
    collector.recordFrame({ captureMs: 35, compareMs: 2, wsSendMs: 3, frameBytes: 100, skipped: false });

    jest.setSystemTime(new Date('2025-01-01T00:00:01Z'));
    const stats = collector.getAggregatedStats();

    expect(stats.primary_bottleneck).toBe('none');
    expect(stats.bottleneck_description).toContain('No significant bottlenecks');
  });

  it('builds collector from config', () => {
    const config = createTestConfig({ performance: { bufferSize: 12, logSummaryInterval: 5 } });
    const collector = PerfCollector.fromConfig('session-config', config, 24);

    collector.recordFrame({ captureMs: 10, compareMs: 1, wsSendMs: 2, frameBytes: 100, skipped: false });
    jest.setSystemTime(new Date('2025-01-01T00:00:01Z'));

    const stats = collector.getAggregatedStats();
    expect(stats.target_fps).toBe(24);
    expect(stats.frame_count).toBe(1);
  });

  it('resets internal counters', () => {
    const collector = new PerfCollector('session-reset', {
      bufferSize: 3,
      logSummaryInterval: 0,
      targetFps: 10,
    });

    collector.recordFrame({ captureMs: 10, compareMs: 1, wsSendMs: 2, frameBytes: 100, skipped: false });
    collector.reset();

    const stats = collector.getAggregatedStats();
    expect(stats.frame_count).toBe(0);
    expect(collector.getSequenceNum()).toBe(0);
  });
});
