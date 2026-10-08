import { describe, it, expect } from 'vitest';
import type { ChartDataPoint } from '../../types';
import {
  connectionAlerts,
  connectionAlertCount,
  measuredConnections,
  metricConnectionAlertCount,
  CONNECTIONS_CRITICAL,
  CONNECTIONS_WARNING,
  GROWTH_FLOOR
} from './connectionAlerts';

const series = (...values: number[]): ChartDataPoint[] =>
  values.map((value, index) => ({ timestamp: new Date(index * 1000).toISOString(), value }));

describe('connectionAlerts level rule', () => {
  it('stays silent at an ordinary count', () => {
    expect(connectionAlerts(372)).toEqual([]);
  });

  it('warns at the elevated threshold', () => {
    const alerts = connectionAlerts(CONNECTIONS_WARNING);
    expect(alerts).toHaveLength(1);
    expect(alerts[0].severity).toBe('warning');
  });

  it('escalates to critical, without double-counting the warning', () => {
    const alerts = connectionAlerts(CONNECTIONS_CRITICAL);
    expect(alerts).toHaveLength(1);
    expect(alerts[0].severity).toBe('critical');
  });

  // The incident this module exists for.
  it('raises a critical alert for the 2026-08-21 count', () => {
    const alerts = connectionAlerts(57_706);
    expect(alerts.some((alert) => alert.severity === 'critical')).toBe(true);
    expect(alerts[0].reason).toContain('57,706');
  });
});

describe('connectionAlerts growth rule', () => {
  it('flags a sustained climb well before the level threshold', () => {
    const climbing = series(600, 900, 1400, 2100, 3000, 4200);
    expect(connectionAlerts(4200, climbing)).toEqual([
      { severity: 'warning', reason: 'connection count climbing steadily' }
    ]);
  });

  it('ignores growth below the floor, where doubling is ordinary', () => {
    expect(connectionAlerts(40, series(5, 10, 18, 25, 32, 40))).toEqual([]);
  });

  it('ignores a spike that has since recovered', () => {
    const spike = series(600, 900, 4000, 3000, 1200, 700);
    expect(connectionAlerts(700, spike)).toEqual([]);
  });

  it('ignores a steady plateau at an unremarkable level', () => {
    expect(connectionAlerts(800, series(800, 805, 795, 810, 800, 802))).toEqual([]);
  });

  it('requires enough samples before claiming a trend', () => {
    expect(connectionAlerts(2000, series(600, 1200, 2000))).toEqual([]);
  });

  it('tolerates minor jitter within a real climb', () => {
    const jittery = series(600, 880, 860, 1500, 2400, 3600);
    expect(connectionAlertCount(3600, jittery)).toBe(1);
  });

  it('reports both rules when a leak is also at a critical level', () => {
    const climbing = series(4000, 9000, 18_000, 30_000, 45_000, 57_000);
    const alerts = connectionAlerts(57_000, climbing);
    expect(alerts).toHaveLength(2);
    expect(alerts.map((alert) => alert.severity)).toEqual(['critical', 'warning']);
  });

  it('does not treat a flat series at the floor as growth', () => {
    expect(connectionAlerts(GROWTH_FLOOR, series(500, 500, 500, 500, 500, 500))).toEqual([]);
  });
});

describe('unread metrics are not alerts', () => {
  it('returns no alerts for an absent count', () => {
    expect(connectionAlerts(undefined)).toEqual([]);
    expect(connectionAlerts(null)).toEqual([]);
  });

  it('returns no alerts for a non-finite count', () => {
    expect(connectionAlerts(Number.NaN)).toEqual([]);
  });

  it('extracts only a measured MetricValue', () => {
    expect(measuredConnections({ state: { case: 'measured', value: 57_706 } } as never)).toBe(57_706);
    expect(
      measuredConnections({ state: { case: 'unsupportedReason', value: 'no backend' } } as never)
    ).toBeUndefined();
    expect(measuredConnections(undefined)).toBeUndefined();
  });

  // An unsupported metric must not be coerced to 0 and read as a healthy host.
  it('raises nothing for an unsupported metric', () => {
    expect(
      metricConnectionAlertCount({ state: { case: 'unsupportedReason', value: 'no backend' } } as never)
    ).toBe(0);
  });

  it('alerts through the MetricValue wrapper when measured', () => {
    expect(metricConnectionAlertCount({ state: { case: 'measured', value: 57_706 } } as never)).toBe(1);
  });
});
