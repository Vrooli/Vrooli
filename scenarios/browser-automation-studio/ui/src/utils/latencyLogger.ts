/**
 * Latency Logger Utility
 *
 * Collects latency samples and computes statistics for A/B testing
 * different frame streaming approaches.
 *
 * Used for the direct WebSocket research spike to measure:
 * - Current API-relay latency (baseline)
 * - Direct playwright-driver connection latency
 *
 * @module utils/latencyLogger
 */

export interface LatencyStats {
  count: number;
  min: number;
  max: number;
  median: number;
  p95: number;
  avg: number;
}

export class LatencyLogger {
  private samples: number[] = [];
  private maxSamples: number;
  private label: string;

  constructor(label: string = 'Latency', maxSamples: number = 100) {
    this.label = label;
    this.maxSamples = maxSamples;
  }

  /**
   * Record a latency sample in milliseconds.
   */
  record(latencyMs: number): void {
    this.samples.push(latencyMs);
    if (this.samples.length > this.maxSamples) {
      this.samples.shift();
    }
  }

  /**
   * Get computed statistics from collected samples.
   */
  getStats(): LatencyStats | null {
    if (this.samples.length === 0) {
      return null;
    }

    const sorted = [...this.samples].sort((a, b) => a - b);
    const count = sorted.length;

    return {
      count,
      min: sorted[0],
      max: sorted[count - 1],
      median: sorted[Math.floor(count / 2)],
      p95: sorted[Math.floor(count * 0.95)] ?? sorted[count - 1],
      avg: Math.round(sorted.reduce((a, b) => a + b, 0) / count * 100) / 100,
    };
  }

  /**
   * Log stats to console in a table format.
   */
  logStats(): void {
    const stats = this.getStats();
    if (!stats) {
      console.log(`[${this.label}] No samples collected yet`);
      return;
    }

    console.log(`[${this.label}] Latency Statistics:`);
    console.table({
      'Sample Count': stats.count,
      'Min (ms)': stats.min.toFixed(2),
      'Max (ms)': stats.max.toFixed(2),
      'Median (ms)': stats.median.toFixed(2),
      'P95 (ms)': stats.p95.toFixed(2),
      'Avg (ms)': stats.avg.toFixed(2),
    });
  }

  /**
   * Clear all collected samples.
   */
  clear(): void {
    this.samples = [];
  }

  /**
   * Get the number of samples collected.
   */
  getSampleCount(): number {
    return this.samples.length;
  }

  /**
   * Get the label for this logger.
   */
  getLabel(): string {
    return this.label;
  }
}

/**
 * Create a new latency logger instance.
 */
export function createLatencyLogger(label?: string, maxSamples?: number): LatencyLogger {
  return new LatencyLogger(label, maxSamples);
}
