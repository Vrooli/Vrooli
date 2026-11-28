import { screen } from '@testing-library/react';
import { fromJson } from '@bufbuild/protobuf';
import { GraphSchema } from '@vrooli/proto-types/system-monitor/v1/devicegraph/devicegraph_pb';
import { DeviceGraphPanel } from './DeviceGraphPanel';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';

describe('DeviceGraphPanel', () => {
  it('renders platform, measured hardware and graded gaps', () => {
    const graph = fromJson(GraphSchema, {
      collectedAt: '2026-08-26T22:00:00Z',
      platform: 'darwin',
      available: true,
      devices: [{
        id: 'block:disk0',
        class: 'storage',
        model: 'APPLE SSD',
        readings: { capacity_bytes: 251000193024 },
        rungs: [
          { rung: 'RUNG_IDENTITY', grade: 'RUNG_GRADE_MEASURED' },
          { rung: 'RUNG_TELEMETRY', grade: 'RUNG_GRADE_UNAVAILABLE', reason: 'SMART is not installed' },
        ],
      }],
      subsystems: [],
    });

    render(<DeviceGraphPanel graph={graph} />);

    expect(screen.getByRole('heading', { name: 'Device graph' })).toBeInTheDocument();
    expect(screen.getByText(/darwin/)).toBeInTheDocument();
    expect(screen.getByText('APPLE SSD')).toBeInTheDocument();
    expect(screen.getByText('251000193024')).toBeInTheDocument();
    expect(screen.getByText('unavailable')).toBeInTheDocument();
    expect(screen.getByText('SMART is not installed')).toBeInTheDocument();
  });

  it('does not turn an unavailable graph into an empty hardware claim', () => {
    const graph = fromJson(GraphSchema, {
      platform: 'darwin',
      available: false,
      unavailableReason: 'device walk failed',
      devices: [],
      subsystems: [],
    });

    render(<DeviceGraphPanel graph={graph} />);

    expect(screen.getByRole('status')).toHaveTextContent('Device graph unavailable: device walk failed.');
    expect(screen.queryByText('No graded devices were observed.')).not.toBeInTheDocument();
  });
});
