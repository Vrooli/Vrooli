/**
 * Proto-to-UI conversion helpers for investigations.
 *
 * Extracted from useInvestigationAgents to share across the codebase.
 */

import type { MessageShape } from '@bufbuild/protobuf';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { InvestigationSchema } from './proto-contracts';
import { InvestigationStatus } from '../../types/api';
import type { InvestigationAgentState } from '../../types';
import { str, num, bool } from '../utils/typeGuards';

type ProtoInvestigation = MessageShape<typeof InvestigationSchema>;

/** Convert an InvestigationStatus enum value to a lowercase string. */
export const statusEnumToString = (status: InvestigationStatus): string => {
  switch (status) {
    case InvestigationStatus.QUEUED: return 'queued';
    case InvestigationStatus.IN_PROGRESS: return 'in_progress';
    case InvestigationStatus.COMPLETED: return 'completed';
    case InvestigationStatus.FAILED: return 'failed';
    case InvestigationStatus.STOPPED: return 'stopped';
    case InvestigationStatus.CANCELLED: return 'cancelled';
    default: return 'investigating';
  }
};

/**
 * Map a proto Investigation to InvestigationAgentState.
 *
 * Uses camelCase fields from the protobuf-generated type.
 * Agent-specific metadata (operation_mode, agent_model, etc.) is extracted
 * from the `details` Struct.
 */
export const protoToAgentState = (inv: ProtoInvestigation): InvestigationAgentState | null => {
  if (!inv.id) return null;

  const details = inv.details as Record<string, unknown> | undefined;

  return {
    id: inv.id,
    status: statusEnumToString(inv.status),
    startTime: inv.startTime ? timestampDate(inv.startTime).toISOString() : new Date().toISOString(),
    autoFix: (details ? bool(details.auto_fix) : undefined) ?? false,
    operationMode: details ? str(details.operation_mode) : undefined,
    model: details ? str(details.agent_model) : undefined,
    resource: details ? str(details.agent_resource) : undefined,
    progress: typeof inv.progress === 'number' ? inv.progress : (details ? num(details.progress) : undefined),
    riskLevel: details ? str(details.risk_level) : undefined,
    note: str(inv.triggerReason) ?? (details ? str(details.note ?? details.user_note) : undefined),
    label: details ? str(details.label) : undefined,
    anomalyId: inv.anomalyId || undefined,
    details,
    lastUpdated: undefined,
    completedAt: inv.endTime ? timestampDate(inv.endTime).toISOString() : undefined,
    error: details ? str(details.error) : undefined,
  };
};
