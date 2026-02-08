import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAppInsightsStore } from './appInsightsStore';

const {
  getCompleteDiagnosticsMock,
  getLighthouseHistoryMock,
  getAppCompletenessMock,
  getAppProxyMetadataMock,
  getAppLocalhostReportMock,
} = vi.hoisted(() => ({
  getCompleteDiagnosticsMock: vi.fn(),
  getLighthouseHistoryMock: vi.fn(),
  getAppCompletenessMock: vi.fn(),
  getAppProxyMetadataMock: vi.fn(),
  getAppLocalhostReportMock: vi.fn(),
}));

vi.mock('@/services/api', () => ({
  appService: {
    getCompleteDiagnostics: getCompleteDiagnosticsMock,
    getLighthouseHistory: getLighthouseHistoryMock,
    getAppCompleteness: getAppCompletenessMock,
    getAppProxyMetadata: getAppProxyMetadataMock,
    getAppLocalhostReport: getAppLocalhostReportMock,
  },
}));

describe('appInsightsStore', () => {
  beforeEach(() => {
    useAppInsightsStore.getState().reset();
    getCompleteDiagnosticsMock.mockReset();
    getLighthouseHistoryMock.mockReset();
    getAppCompletenessMock.mockReset();
    getAppProxyMetadataMock.mockReset();
    getAppLocalhostReportMock.mockReset();

    getCompleteDiagnosticsMock.mockResolvedValue({ scenario: 'scenario-1', warnings: [], severity: 'ok' });
    getLighthouseHistoryMock.mockResolvedValue({ scenario: 'scenario-1', reports: [], trend: { performance: [], accessibility: [], best_practices: [], seo: [] } });
    getAppCompletenessMock.mockResolvedValue({ scenario: 'scenario-1', details: [] });
    getAppProxyMetadataMock.mockResolvedValue({ appId: 'scenario-1', generatedAt: Date.now(), hosts: [], primary: { port: 3000, path: '/' }, ports: [] });
    getAppLocalhostReportMock.mockResolvedValue({ scenario: 'scenario-1', checked_at: new Date().toISOString(), files_scanned: 0, findings: [] });
  });

  it('prefetches all insights for an app', async () => {
    await useAppInsightsStore.getState().prefetch('scenario-1');

    const entry = useAppInsightsStore.getState().byAppId['scenario-1'];
    expect(entry?.diagnostics.data?.scenario).toBe('scenario-1');
    expect(entry?.lighthouse.data?.scenario).toBe('scenario-1');
    expect(entry?.completeness.data?.scenario).toBe('scenario-1');
    expect(entry?.proxy.data?.proxyMetadata?.appId).toBe('scenario-1');
    expect(entry?.proxy.data?.localhostReport?.scenario).toBe('scenario-1');
  });

  it('reuses fresh cache unless force is true', async () => {
    await useAppInsightsStore.getState().prefetch('scenario-1');
    await useAppInsightsStore.getState().prefetch('scenario-1');

    expect(getCompleteDiagnosticsMock).toHaveBeenCalledTimes(1);
    expect(getLighthouseHistoryMock).toHaveBeenCalledTimes(1);
    expect(getAppCompletenessMock).toHaveBeenCalledTimes(1);
    expect(getAppProxyMetadataMock).toHaveBeenCalledTimes(1);
    expect(getAppLocalhostReportMock).toHaveBeenCalledTimes(1);

    await useAppInsightsStore.getState().prefetch('scenario-1', { force: true });

    expect(getCompleteDiagnosticsMock).toHaveBeenCalledTimes(2);
    expect(getLighthouseHistoryMock).toHaveBeenCalledTimes(2);
    expect(getAppCompletenessMock).toHaveBeenCalledTimes(2);
    expect(getAppProxyMetadataMock).toHaveBeenCalledTimes(2);
    expect(getAppLocalhostReportMock).toHaveBeenCalledTimes(2);
  });
});
