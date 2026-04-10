import { describe, it, expect } from 'vitest';
import {
  buildMetricValues,
  computeTriggerProgress,
  getProgressColor,
  formatMetricValue,
  formatTriggerReadout,
  type SystemMetricSources,
} from './triggerMetrics';

// ---------------------------------------------------------------------------
// buildMetricValues
// ---------------------------------------------------------------------------

describe('buildMetricValues', () => {
  it('maps all available sources to trigger IDs', () => {
    const sources: SystemMetricSources = {
      cpuUsage: 42,
      memoryUsage: 30,
      tcpConnections: 150,
      diskUsagePercent: 75,
      anomalousProcessCount: 3,
    };

    const result = buildMetricValues(sources);

    expect(result).toEqual({
      high_cpu: 42,
      memory_pressure: 70, // 100 - 30
      network_connections: 150,
      disk_space: 75,
      process_anomaly: 3,
    });
  });

  it('inverts memory_usage to available memory', () => {
    expect(buildMetricValues({ memoryUsage: 80 })).toEqual({ memory_pressure: 20 });
    expect(buildMetricValues({ memoryUsage: 0 })).toEqual({ memory_pressure: 100 });
    expect(buildMetricValues({ memoryUsage: 100 })).toEqual({ memory_pressure: 0 });
  });

  it('omits keys for undefined sources', () => {
    const result = buildMetricValues({ cpuUsage: 50 });
    expect(result).toEqual({ high_cpu: 50 });
    expect(result).not.toHaveProperty('memory_pressure');
    expect(result).not.toHaveProperty('disk_space');
    expect(result).not.toHaveProperty('process_anomaly');
    expect(result).not.toHaveProperty('network_connections');
  });

  it('returns empty object for empty sources', () => {
    expect(buildMetricValues({})).toEqual({});
  });

  it('handles zero values correctly', () => {
    const result = buildMetricValues({
      cpuUsage: 0,
      tcpConnections: 0,
      anomalousProcessCount: 0,
    });
    expect(result).toEqual({
      high_cpu: 0,
      network_connections: 0,
      process_anomaly: 0,
    });
  });
});

// ---------------------------------------------------------------------------
// computeTriggerProgress
// ---------------------------------------------------------------------------

describe('computeTriggerProgress', () => {
  describe('above condition', () => {
    it('returns 0 when value is 0', () => {
      expect(computeTriggerProgress(0, 95, 'above', '%')).toBe(0);
    });

    it('returns ratio of value to threshold', () => {
      expect(computeTriggerProgress(47.5, 95, 'above', '%')).toBe(0.5);
    });

    it('returns 1 when value equals threshold', () => {
      expect(computeTriggerProgress(95, 95, 'above', '%')).toBe(1);
    });

    it('clamps to 1 when value exceeds threshold', () => {
      expect(computeTriggerProgress(120, 95, 'above', '%')).toBe(1);
    });

    it('works with non-percentage units', () => {
      expect(computeTriggerProgress(1000, 2000, 'above', ' connections')).toBe(0.5);
    });
  });

  describe('below condition (percentage)', () => {
    // memory_pressure: threshold=10%, safeRef=100%
    // progress = 1 - (value - 10) / (100 - 10) = 1 - (value - 10) / 90

    it('returns 0 when value is at safe reference (100%)', () => {
      expect(computeTriggerProgress(100, 10, 'below', '%')).toBeCloseTo(0, 5);
    });

    it('returns 1 when value equals threshold', () => {
      expect(computeTriggerProgress(10, 10, 'below', '%')).toBe(1);
    });

    it('returns ~0.5 at midpoint between safe and threshold', () => {
      // midpoint between 10 and 100 is 55
      expect(computeTriggerProgress(55, 10, 'below', '%')).toBe(0.5);
    });

    it('returns low progress for value far from threshold', () => {
      // value=71, threshold=10: progress = 1 - (71-10)/90 = 1 - 0.678 ≈ 0.322
      const progress = computeTriggerProgress(71, 10, 'below', '%');
      expect(progress).toBeCloseTo(0.322, 2);
      expect(progress).toBeGreaterThan(0);
      expect(progress).toBeLessThan(0.5);
    });

    it('clamps to 1 when value drops below threshold', () => {
      expect(computeTriggerProgress(5, 10, 'below', '%')).toBe(1);
    });
  });

  describe('below condition (count units)', () => {
    // For non-percentage: safeRef = threshold * 10
    // e.g. threshold=25: safeRef=250, range=225

    it('returns 0 when value is at safeRef', () => {
      expect(computeTriggerProgress(250, 25, 'below', ' processes')).toBeCloseTo(0, 5);
    });

    it('returns 1 when value equals threshold', () => {
      expect(computeTriggerProgress(25, 25, 'below', ' processes')).toBe(1);
    });
  });

  describe('edge cases', () => {
    it('returns 0 for undefined currentValue', () => {
      expect(computeTriggerProgress(undefined, 95, 'above', '%')).toBe(0);
    });

    it('returns 0 for zero threshold', () => {
      expect(computeTriggerProgress(50, 0, 'above', '%')).toBe(0);
    });

    it('returns 0 for negative threshold', () => {
      expect(computeTriggerProgress(50, -10, 'above', '%')).toBe(0);
    });

    it('handles negative current value (clamped to 0)', () => {
      expect(computeTriggerProgress(-5, 95, 'above', '%')).toBe(0);
    });
  });
});

// ---------------------------------------------------------------------------
// getProgressColor
// ---------------------------------------------------------------------------

describe('getProgressColor', () => {
  it('returns success for progress < 0.5', () => {
    expect(getProgressColor(0)).toBe('var(--color-success)');
    expect(getProgressColor(0.49)).toBe('var(--color-success)');
  });

  it('returns warning for progress 0.5–0.8', () => {
    expect(getProgressColor(0.5)).toBe('var(--color-warning)');
    expect(getProgressColor(0.79)).toBe('var(--color-warning)');
  });

  it('returns error for progress >= 0.8', () => {
    expect(getProgressColor(0.8)).toBe('var(--color-error)');
    expect(getProgressColor(1.0)).toBe('var(--color-error)');
  });
});

// ---------------------------------------------------------------------------
// formatMetricValue
// ---------------------------------------------------------------------------

describe('formatMetricValue', () => {
  it('formats percentage values as rounded integers', () => {
    expect(formatMetricValue(42.7, '%')).toBe('43%');
    expect(formatMetricValue(0, '%')).toBe('0%');
    expect(formatMetricValue(99.4, '%')).toBe('99%');
  });

  it('formats integer values without decimals', () => {
    expect(formatMetricValue(150, ' connections')).toBe('150');
    expect(formatMetricValue(0, ' processes')).toBe('0');
  });

  it('formats non-integer values with one decimal', () => {
    expect(formatMetricValue(3.7, ' processes')).toBe('3.7');
    expect(formatMetricValue(0.5, ' unit')).toBe('0.5');
  });
});

// ---------------------------------------------------------------------------
// formatTriggerReadout
// ---------------------------------------------------------------------------

describe('formatTriggerReadout', () => {
  it('shows current/threshold for available value', () => {
    expect(formatTriggerReadout(42, 95, '%')).toBe('42% / 95%');
  });

  it('shows em-dash for undefined value', () => {
    expect(formatTriggerReadout(undefined, 90, '%')).toBe('— / 90%');
  });

  it('handles connection units', () => {
    expect(formatTriggerReadout(150, 2000, ' connections')).toBe('150 / 2000 connections');
  });

  it('handles process units', () => {
    expect(formatTriggerReadout(3, 25, ' processes')).toBe('3 / 25 processes');
  });

  it('formats non-integer current values', () => {
    expect(formatTriggerReadout(3.7, 25, ' processes')).toBe('3.7 / 25 processes');
  });
});
