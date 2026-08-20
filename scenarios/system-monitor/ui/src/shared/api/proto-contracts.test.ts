import { describe, expect, it } from 'vitest';
import * as contracts from './proto-contracts';

describe('protobuf JSON contracts', () => {
  it('parses every public response and domain contract with unknown fields ignored', () => {
    const parserNames = [
      'parseMetricsResponse', 'parseDetailedMetrics', 'parseProcessMonitorData', 'parseInfrastructureMonitorData',
      'parseMetricsTimelineResponse', 'parseDiskDetailResponse', 'parseInvestigation', 'parseTriggerConfig',
      'parseCooldownStatus', 'parseSystemSettings', 'parseEnhancedSystemReport', 'parseListReportsResponse',
      'parseGenerateReportResponse', 'parseInvestigationScript', 'parseScriptExecution', 'parseListScriptsResponse',
      'parseGetScriptResponse', 'parseExecuteScriptResponse', 'parseGetSettingsResponse', 'parseUpdateSettingsResponse',
      'parseResetSettingsResponse', 'parseGetMaintenanceStateResponse', 'parseSetMaintenanceStateResponse',
      'parseGetCapacityOverviewResponse', 'parseListCapacityClaimsResponse', 'parseReconcileCapacityResponse',
      'parseGetCapacityPolicyResponse', 'parseSetCapacityPolicyResponse', 'parseGetTriggersResponse',
      'parseTriggerInvestigationResponse', 'parseGetCooldownStatusResponse', 'parseListInvestigationsResponse',
    ] as const;
    for (const name of parserNames) {
      const parser = contracts[name];
      expect(parser({ unknown_field: 'ignored' })).toBeDefined();
    }
    expect(contracts.parseInvestigations({})).toEqual([]);
    expect(contracts.parseInvestigations([])).toEqual([]);
  });
});
