import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { expectNoA11yViolations } from '@vrooli/api-base/testing';
import { NetworkDetailView } from './NetworkDetailView';

describe('NetworkDetailView accessibility', () => {
  it('has no detectable violations for partial network evidence', async () => {
    const { container } = render(
      <NetworkDetailView
        metrics={{ connections: { state: { case: 'measured', value: 4 } } } as never}
        detailedMetrics={{ networkDetails: {
          tcpStates: { established: 4, total: 4 },
          interfaces: [{ name: 'eth0', up: true }],
          ownership: { totalConnections: 4, attributedConnections: 3, attributionCoveragePercent: 75, truncated: true, owners: [] },
          capabilities: {
            tcpStates: { state: { case: 'measured', value: 1 } },
            interfaceCounters: { state: { case: 'measured', value: 1 } },
            transportCounters: { state: { case: 'unsupportedReason', value: 'not exposed' } },
          },
          verdict: { state: 'warning', summary: 'Network evidence is partial', reasons: ['transport counters unsupported'] },
        } } as never}
        metricHistory={null}
        onBack={() => undefined}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
