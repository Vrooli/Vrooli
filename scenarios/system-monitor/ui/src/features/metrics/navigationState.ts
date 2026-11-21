import type { Alert, AlertSeverity, CardType } from '../../types';
import type { ChartInspection } from './components/MetricSparkline';
import { isRecord } from '../../shared/utils/typeGuards';

export interface ConditionNavigationState {
  alert: Alert;
  revalidated: boolean;
  subject: string;
  target: CardType;
  windowStart?: string;
  windowEnd?: string;
  evaluatedAt?: string;
}

export interface SampleNavigationState {
  selected: NonNullable<ChartInspection['selected']>;
  zoom?: NonNullable<ChartInspection['zoom']>;
  subject: string;
  target: CardType;
}

export interface MetricNavigationState {
  overviewSection?: string;
  condition?: ConditionNavigationState;
  sample?: SampleNavigationState;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return isRecord(value) ? value : undefined;
}

function isCardType(value: unknown): value is CardType {
  return value === 'cpu' || value === 'memory' || value === 'network' || value === 'disk' || value === 'gpu';
}

function isAlertSeverity(value: unknown): value is AlertSeverity {
  return value === 'low' || value === 'medium' || value === 'warning' || value === 'high' || value === 'critical';
}

function readAlert(value: unknown): Alert | undefined {
  const alert = asRecord(value);
  if (!alert
    || typeof alert['id'] !== 'string'
    || typeof alert['timestamp'] !== 'string'
    || !isAlertSeverity(alert['severity'])
    || typeof alert['category'] !== 'string'
    || typeof alert['message'] !== 'string'
    || typeof alert['resolved'] !== 'boolean'
    || (alert['details'] !== undefined && typeof alert['details'] !== 'string')) {
    return undefined;
  }
  return {
    id: alert['id'],
    timestamp: alert['timestamp'],
    severity: alert['severity'],
    category: alert['category'],
    message: alert['message'],
    resolved: alert['resolved'],
    ...(typeof alert['details'] === 'string' ? { details: alert['details'] } : {}),
  };
}

function readCondition(value: unknown): ConditionNavigationState | undefined {
  const condition = asRecord(value);
  const alert = condition ? readAlert(condition['alert']) : undefined;
  if (!condition || !alert || typeof condition['revalidated'] !== 'boolean'
    || typeof condition['subject'] !== 'string' || !isCardType(condition['target'])) {
    return undefined;
  }
  const optionalFields = ['windowStart', 'windowEnd', 'evaluatedAt'] as const;
  if (optionalFields.some(field => condition[field] !== undefined && typeof condition[field] !== 'string')) return undefined;
  return {
    alert,
    revalidated: condition['revalidated'],
    subject: condition['subject'],
    target: condition['target'],
    ...(typeof condition['windowStart'] === 'string' ? { windowStart: condition['windowStart'] } : {}),
    ...(typeof condition['windowEnd'] === 'string' ? { windowEnd: condition['windowEnd'] } : {}),
    ...(typeof condition['evaluatedAt'] === 'string' ? { evaluatedAt: condition['evaluatedAt'] } : {}),
  };
}

function readSample(value: unknown): SampleNavigationState | undefined {
  const sample = asRecord(value);
  const selected = sample ? asRecord(sample['selected']) : undefined;
  const raw = selected?.['raw'];
  if (!sample || !selected || typeof selected['timestamp'] !== 'string'
    || typeof raw !== 'number' || !Number.isFinite(raw)
    || typeof sample['subject'] !== 'string' || !isCardType(sample['target'])) {
    return undefined;
  }

  const zoomValue = sample['zoom'];
  let zoom: NonNullable<ChartInspection['zoom']> | undefined;
  if (zoomValue !== undefined && zoomValue !== null) {
    const candidate = asRecord(zoomValue);
    if (!candidate || typeof candidate['start'] !== 'string' || typeof candidate['end'] !== 'string') return undefined;
    zoom = { start: candidate['start'], end: candidate['end'] };
  }

  return {
    selected: { timestamp: selected['timestamp'], raw },
    ...(zoom ? { zoom } : {}),
    subject: sample['subject'],
    target: sample['target'],
  };
}

/** Parse untrusted React Router state into the dashboard's known navigation contract. */
export function readMetricNavigationState(value: unknown): MetricNavigationState {
  const state = asRecord(value);
  if (!state) return {};
  const condition = readCondition(state['condition']);
  const sample = readSample(state['sample']);
  return {
    ...(typeof state['overviewSection'] === 'string' ? { overviewSection: state['overviewSection'] } : {}),
    ...(condition ? { condition } : {}),
    ...(sample ? { sample } : {}),
  };
}
