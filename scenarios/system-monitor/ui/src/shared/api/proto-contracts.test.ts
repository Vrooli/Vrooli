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

  it('preserves measured paging-flow states in timeline samples', () => {
    const parsed = contracts.parseMetricsTimelineResponse({
      window_seconds: 30,
      sample_interval_seconds: 5,
      samples: [{
        timestamp: '2026-08-22T15:53:11.374Z',
        major_faults: { measured: 180.8 },
        swap_traffic: { measured: 244.9 }
      }]
    });
    expect(parsed.samples[0].majorFaults?.state.case).toBe('measured');
    expect(parsed.samples[0].majorFaults?.state.value).toBe(180.8);
    expect(parsed.samples[0].swapTraffic?.state.case).toBe('measured');
  });
});
