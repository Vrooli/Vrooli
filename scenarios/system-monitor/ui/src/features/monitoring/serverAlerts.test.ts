import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { describe, expect, it } from 'vitest';
import { AlertSchema, ListAlertsResponseSchema, RuleStateSchema } from '@vrooli/proto-types/system-monitor/v1/alerts/alerts_pb';
import { toAlertFeed } from './serverAlerts';

const evaluatedAt = timestampFromDate(new Date('2026-10-07T12:00:00Z'));

describe('toAlertFeed', () => {
  it('maps server condition alerts onto the panel model', () => {
    const response = create(ListAlertsResponseSchema, {
      monitoringActive: true,
      evaluatedAt,
      sustainSeconds: 300,
      alerts: [create(AlertSchema, {
        id: 'cond_memory_pressure_host_1',
        type: 'condition',
        severity: 'critical',
        ruleId: 'memory_pressure',
        message: 'Memory pressure is critical: swap-in 47.0 MiB/s.',
        openedAt: timestampFromDate(new Date('2026-10-07T11:50:00Z')),
        escalated: true,
        anomalyId: 'anomaly_memory_pressure_1',
      })],
      rules: [
        create(RuleStateSchema, { ruleId: 'cpu_usage', status: 'supported' }),
        create(RuleStateSchema, { ruleId: 'orphaned_agents', status: 'unsupported', reason: 'control plane does not classify orphans yet' }),
      ],
    });

    const { alerts, feed } = toAlertFeed(response);

    expect(alerts).toEqual([{
      id: 'cond_memory_pressure_host_1',
      timestamp: '2026-10-07T11:50:00.000Z',
      severity: 'critical',
      category: 'memory_pressure',
      message: 'Memory pressure is critical: swap-in 47.0 MiB/s.',
      resolved: false,
      details: 'Escalated to investigation (anomaly_memory_pressure_1)',
    }]);
    expect(feed.status).toBe('ready');
    expect(feed.ruleIssues).toEqual([{ ruleId: 'orphaned_agents', status: 'unsupported', reason: 'control plane does not classify orphans yet' }]);
  });

  it('keeps the warning severity the evaluator emits', () => {
    const response = create(ListAlertsResponseSchema, {
      monitoringActive: true,
      evaluatedAt,
      alerts: [create(AlertSchema, { id: 'a', severity: 'warning', ruleId: 'cpu_usage', message: 'CPU usage is 90.0%' })],
    });
    expect(toAlertFeed(response).alerts[0]?.severity).toBe('warning');
  });

  it('distinguishes a paused monitor and an evaluator that has not run from a healthy host', () => {
    expect(toAlertFeed(create(ListAlertsResponseSchema, { monitoringActive: false, evaluatedAt })).feed.status).toBe('paused');
    expect(toAlertFeed(create(ListAlertsResponseSchema, { monitoringActive: true })).feed.status).toBe('unevaluated');
  });

  it('maps an unrecognised severity, a missing open time and a ruleless alert without inventing data', () => {
    const response = create(ListAlertsResponseSchema, {
      monitoringActive: true,
      evaluatedAt,
      alerts: [create(AlertSchema, {
        id: 'legacy',
        type: 'threshold_violation',
        severity: 'severe',
        message: 'Disk usage spiked',
        resolvedAt: timestampFromDate(new Date('2026-10-07T12:05:00Z')),
      })],
    });

    expect(toAlertFeed(response).alerts[0]).toEqual({
      id: 'legacy',
      timestamp: '',
      severity: 'medium',
      category: 'threshold_violation',
      message: 'Disk usage spiked',
      resolved: true,
      details: undefined,
    });
  });
});
