import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { describe, expect, it } from 'vitest';
import { InvestigationStatus } from '../../types';
import { protoToAgentState, statusEnumToString } from './proto-converters';

describe('proto-converters', () => {
  it('maps every known investigation status and an unknown value', () => {
    expect(statusEnumToString(InvestigationStatus.QUEUED)).toBe('queued');
    expect(statusEnumToString(InvestigationStatus.IN_PROGRESS)).toBe('in_progress');
    expect(statusEnumToString(InvestigationStatus.COMPLETED)).toBe('completed');
    expect(statusEnumToString(InvestigationStatus.FAILED)).toBe('failed');
    expect(statusEnumToString(InvestigationStatus.STOPPED)).toBe('stopped');
    expect(statusEnumToString(InvestigationStatus.CANCELLED)).toBe('cancelled');
    expect(statusEnumToString(999 as InvestigationStatus)).toBe('investigating');
  });

  it('returns null without an id and maps optional metadata and timestamps', () => {
    expect(protoToAgentState({ status: InvestigationStatus.QUEUED })).toBeNull();

    const timestamp = timestampFromDate(new Date('2026-01-02T03:04:05.000Z'));
    const state = protoToAgentState({
      id: 'agent-1',
      status: InvestigationStatus.COMPLETED,
      startTime: timestamp,
      endTime: timestamp,
      progress: 42,
      triggerReason: 'operator request',
      anomalyId: 'anomaly-1',
      details: {
        auto_fix: true,
        operation_mode: 'safe',
        agent_model: 'model-a',
        agent_resource: 'resource-a',
        risk_level: 'low',
        label: 'test run',
        error: 'none',
      },
    });

    expect(state).toMatchObject({
      id: 'agent-1', status: 'completed', startTime: '2026-01-02T03:04:05.000Z',
      completedAt: '2026-01-02T03:04:05.000Z', autoFix: true, operationMode: 'safe',
      model: 'model-a', resource: 'resource-a', progress: 42, riskLevel: 'low',
      note: 'operator request', label: 'test run', anomalyId: 'anomaly-1', error: 'none',
    });
  });

  it('uses details progress and note fallbacks when primary fields are absent', () => {
    const state = protoToAgentState({
      id: 'agent-2',
      status: InvestigationStatus.QUEUED,
      details: { progress: 12, user_note: 'saved note', auto_fix: false },
    });
    expect(state).toMatchObject({ id: 'agent-2', progress: 12, note: 'saved note', autoFix: false });
    expect(state?.startTime).toEqual(expect.any(String));
  });
});
