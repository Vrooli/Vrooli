import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { CpuDetailView } from './CpuDetailView';
import { NetworkDetailView } from './NetworkDetailView';
import { GpuDetailView } from './GpuDetailView';

vi.mock('./MetricDetailViews', () => ({
  MetricDetailLayout: ({ children, headline, subhead }: { children: React.ReactNode; headline: string; subhead?: string }) => <div><div>{headline}</div>{subhead && <div>{subhead}</div>}{children}</div>,
  MetricLineChart: () => <div data-testid="chart" />,
}));
vi.mock('./MetricRenderHelpers', () => ({ renderProcessTable: (processes: unknown) => <div>{processes ? 'processes' : 'no processes'}</div> }));
vi.mock('../../monitoring/components/ProcessMonitor', () => ({ ProcessMonitor: () => <div>process monitor</div> }));
vi.mock('./GpuDeviceCard', () => ({ GpuDeviceCard: ({ device }: { device: { index: number } }) => <div>device {device.index}</div> }));

describe('metric detail variants', () => {
  it('renders CPU measured and unavailable branches', () => {
    render(<CpuDetailView
      metrics={{ cpu: { state: { case: 'measured', value: 12 } } } as never}
      detailedMetrics={{ cpuDetails: { usage: 33, loadAverage: [1, 2, 3], contextSwitches: 100, totalGoroutines: 20, topProcesses: [{}] }, timestamp: { seconds: 0n, nanos: 0 } } as never}
      processMonitorData={{} as never} metricHistory={{ cpu: [] } as never} onBack={vi.fn()}
    />);
    expect(screen.getByText('33.0% utilization')).toBeInTheDocument();
    expect(screen.getByTestId('cpu-mode-chart')).toBeInTheDocument();
    expect(screen.getByText('processes')).toBeInTheDocument();

    render(<CpuDetailView metrics={null} detailedMetrics={null} processMonitorData={null} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('Utilization not measured')).toBeInTheDocument();
    expect(screen.getAllByText('not yet sampled').length).toBeGreaterThan(0);
    expect(screen.getAllByText('no processes').length).toBeGreaterThan(0);
  });

  it('renders typed CPU diagnostic states and thermal evidence', () => {
    render(<CpuDetailView
      metrics={null}
      detailedMetrics={{
        cpuDetails: {
          usage: 42,
          usageState: { state: { case: 'measured', value: 42 } },
          loadAverage: [2, 1, 0.5],
          loadAverageState: { state: { case: 'measured', value: 2 } },
          contextSwitchesPerSecond: { state: { case: 'measured', value: 12 } },
          modeBreakdown: { user: { state: { case: 'measured', value: 40 } }, steal: { state: { case: 'measured', value: 6 } } },
          cpuPsiSomeAvg10: { state: { case: 'measured', value: 11 } },
          cpuPsiFullAvg10: { state: { case: 'unsupportedReason', value: 'PSI is Linux-specific' }, provenance: 'cpu PSI' },
          runQueueDepth: { state: { case: 'measured', value: 3 } },
          normalizedLoad1: { state: { case: 'measured', value: 0.5 } },
          normalizedLoad5: { state: { case: 'measured', value: 0.4 } },
          perCoreUtilization: { cpu0: { state: { case: 'measured', value: 90 } } },
          coreImbalanceIndex: { state: { case: 'measured', value: 35 } },
          quotaThrottling: { state: { case: 'unsupportedReason', value: 'no cgroup CPU limit applies' } },
          frequencyDerateRatio: { state: { case: 'measured', value: 0.8 } },
          thermalThrottleEvidence: { state: { case: 'measured', value: 71 } },
          thermalTripPointCelsius: { state: { case: 'measured', value: 85 } },
          topProcesses: [],
        },
        timestamp: { seconds: 1n, nanos: 0 },
      } as never}
      processMonitorData={{} as never} metricHistory={{ cpu: [] } as never} onBack={vi.fn()}
    />);
    expect(screen.getByText('42.0% utilization — hypervisor steal detected')).toBeInTheDocument();
    expect(screen.getByText('85.0')).toBeInTheDocument();
    expect(screen.getAllByText(/unsupportedReason/).length).toBeGreaterThan(0);
  });

  it('renders network states, pools, and unavailable details', () => {
    render(<NetworkDetailView
      metrics={{ connections: { state: { case: 'measured', value: 7 } } } as never}
      detailedMetrics={{ timestamp: { seconds: 0n, nanos: 0 }, networkDetails: {
        tcpStates: { total: 7, established: 4, '$typeName': 'x' },
        networkStats: { bandwidthInMbps: 1.2, bandwidthOutMbps: 2.3, packetLoss: 0.1, dnsSuccessRate: 99, dnsLatencyMs: 4 },
        portUsage: { used: 2, total: 10 },
        connectionPools: [{ name: 'http', active: 1, idle: 2, waiting: 0, maxSize: 10, leakRisk: 'low' }],
      } } as never}
      metricHistory={{ network: [{ timestamp: 'now', value: 7 }] } as never} onBack={vi.fn()}
    />);
    expect(screen.getByText('7 active connections')).toBeInTheDocument();
    expect(screen.getByText('ESTABLISHED')).toBeInTheDocument();
    expect(screen.getByText('http')).toBeInTheDocument();

    render(<NetworkDetailView metrics={null} detailedMetrics={null} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('Connections not measured')).toBeInTheDocument();
    expect(screen.getAllByText('Connection state metrics unavailable.')).toHaveLength(1);
    expect(screen.getByText('Network statistics unavailable.')).toBeInTheDocument();
    render(<NetworkDetailView metrics={null} detailedMetrics={{ networkDetails: { tcpStates: {}, networkStats: {}, portUsage: {} } } as never} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getAllByText(/—/).length).toBeGreaterThan(0);
    render(<NetworkDetailView metrics={null} detailedMetrics={{ networkDetails: { connectionPools: [] } } as never} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('Network statistics unavailable.')).toBeInTheDocument();
  });

  it('renders GPU summary, errors, devices, and empty state', () => {
    render(<GpuDetailView detailedMetrics={{ gpuDetails: {
      summary: { averageUtilizationPercent: 20, deviceCount: 1, usedMemoryMb: 10, totalMemoryMb: 20, averageTemperatureC: 55 },
      driverVersion: '550', primaryModel: 'A10', errors: ['driver warning'], devices: [{ index: 0, uuid: 'u' }],
    } } as never} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('20.0% Avg')).toBeInTheDocument();
    expect(screen.getByText('Driver 550 • A10')).toBeInTheDocument();
    expect(screen.getByText('driver warning')).toBeInTheDocument();
    expect(screen.getByText('device 0')).toBeInTheDocument();

    render(<GpuDetailView detailedMetrics={{ gpuDetails: { devices: [] } } as never} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('No GPU devices detected.')).toBeInTheDocument();
    render(<GpuDetailView detailedMetrics={null} metricHistory={null} onBack={vi.fn()} />);
    expect(screen.getByText('GPU metrics unavailable on this host.')).toBeInTheDocument();
  });
});
