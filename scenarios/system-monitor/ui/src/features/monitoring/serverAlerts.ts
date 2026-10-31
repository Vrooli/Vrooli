// DOC: docs/concepts/ARCHITECTURE.md#alerting
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { ListAlertsResponse } from '@vrooli/proto-types/system-monitor/v1/alerts/alerts_pb';
import { protoFetch } from '../../shared/api/apiFetch';
import { parseListAlertsResponse } from '../../shared/api/proto-contracts';
import type { Alert, AlertSeverity } from '../../types';

/**
 * Alerts are evaluated, deduplicated, and resolved by the API. The dashboard
 * only reads them; it never derives an alert from a metric value.
 */

/** Feed state, so an unreachable or idle evaluator is never shown as "no alerts". */
export type AlertFeedStatus = 'loading' | 'ready' | 'unevaluated' | 'paused' | 'error';

export interface RuleIssue {
  ruleId: string;
  status: string;
  reason: string;
}

export interface AlertFeed {
  status: AlertFeedStatus;
  error?: string;
  evaluatedAt?: string;
  sustainSeconds?: number;
  ruleIssues: RuleIssue[];
}

const knownSeverities: readonly AlertSeverity[] = ['low', 'medium', 'warning', 'high', 'critical'];

const toSeverity = (value: string): AlertSeverity =>
  knownSeverities.find(severity => severity === value) ?? 'medium';

/** Maps the server response onto the panel's alert model and feed state. */
export const toAlertFeed = (response: ListAlertsResponse): { alerts: Alert[]; feed: AlertFeed } => {
  const alerts: Alert[] = response.alerts.map(alert => ({
    id: alert.id,
    timestamp: alert.openedAt ? timestampDate(alert.openedAt).toISOString() : '',
    severity: toSeverity(alert.severity),
    category: alert.ruleId || alert.type,
    message: alert.message,
    resolved: Boolean(alert.resolvedAt),
    details: alert.escalated ? `Escalated to investigation (${alert.anomalyId})` : undefined,
  }));
  const ruleIssues = response.rules
    .filter(rule => rule.status !== 'supported')
    .map(rule => ({ ruleId: rule.ruleId, status: rule.status, reason: rule.reason }));
  let status: AlertFeedStatus = 'ready';
  if (!response.monitoringActive) {
    status = 'paused';
  } else if (!response.evaluatedAt) {
    status = 'unevaluated';
  }
  return {
    alerts,
    feed: {
      status,
      evaluatedAt: response.evaluatedAt ? timestampDate(response.evaluatedAt).toISOString() : undefined,
      sustainSeconds: response.sustainSeconds,
      ruleIssues,
    },
  };
};

/** Reads the active condition alerts from the API. */
export const fetchServerAlerts = (signal?: AbortSignal) =>
  protoFetch('/api/v1/alerts?status=active', parseListAlertsResponse, { signal });
