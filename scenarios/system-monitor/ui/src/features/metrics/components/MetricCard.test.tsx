import { create } from '@bufbuild/protobuf';
import { cleanup, fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { MetricValueSchema, type MetricValue } from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';
import { MetricCard } from './MetricCard';

const renderCard = (metric?: MetricValue) =>
  // provider-free-exception: MetricCard is a pure presentational component;
  // the shared provider package currently ships a conflicting React major.
  render(
    <MetricCard
      type="cpu"
      label="CPU"
      unit="%"
      metric={metric}
      isExpanded={false}
      onToggle={() => undefined}
      details={{} as never}
      alertCount={0}
    />,
  );

describe('MetricCard metric states', () => {
  it('does not fabricate a numeric reading when no metric is present', () => {
    renderCard();

    expect(screen.getByText('—')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'CPU: not measured');
  });

  it('renders an explicit unsupported reason without a zero fallback', () => {
    renderCard(
      create(MetricValueSchema, {
        state: { case: 'unsupportedReason', value: 'GPU is not available on this host' },
      }),
    );

    expect(screen.getByText('—')).toBeInTheDocument();
    expect(screen.getByText('GPU is not available on this host')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'CPU: GPU is not available on this host');
  });

  it('keeps a measured zero distinct from an unavailable metric', () => {
    renderCard(create(MetricValueSchema, { state: { case: 'measured', value: 0 } }));

    expect(screen.getByText('0.0')).toBeInTheDocument();
    expect(screen.queryByText('—')).not.toBeInTheDocument();
  });

  it('renders failed evidence as an alert state with its reason', () => {
    renderCard(
      create(MetricValueSchema, {
        state: { case: 'failedError', value: 'permission denied' },
      }),
    );

    expect(screen.getByText('⚠')).toBeInTheDocument();
    expect(screen.getByText('permission denied')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'CPU: permission denied');
  });

  it('uses the legacy value only when no typed metric state is supplied', () => {
    render(
      <MetricCard
        type="memory"
        label="Memory"
        unit="%"
        value={42.5}
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={2}
      />,
    );

    expect(screen.getByText('42.5')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByRole('meter')).toHaveAttribute('aria-valuenow', '42.5');
  });

  it('uses the measured state over a stale legacy value and exposes details action', () => {
    const onOpenDetails = vi.fn();
    const onToggle = vi.fn();
    render(
      <MetricCard
        type="disk"
        label="Disk"
        unit="%"
        value={1}
        metric={create(MetricValueSchema, { state: { case: 'measured', value: 91.25 } })}
        isExpanded={false}
        onToggle={onToggle}
        onOpenDetails={onOpenDetails}
        detailButtonLabel="OPEN DISK DETAILS"
        alertCount={0}
      />,
    );

    expect(screen.getByText('91.3')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'OPEN DISK DETAILS' }));
    expect(onOpenDetails).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText('Disk'));
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it('renders a bounded history sparkline instead of the progress bar', () => {
    render(
      <MetricCard
        type="network"
        label="Network"
        unit="connections"
        metric={create(MetricValueSchema, { state: { case: 'measured', value: 12 } })}
        history={[{ timestamp: new Date().toISOString(), value: 12 }]}
        historyWindowSeconds={3600}
        historyUnit="connections/s"
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={0}
      />,
    );

    expect(screen.queryByText('12.0')).toBeInTheDocument();
    expect(screen.queryByRole('meter')).toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
  });

  it('covers the metric-family color and unit policies', () => {
    const families = [
      ['cpu', '%'],
      ['memory', 'MB'],
      ['network', 'connections'],
      ['disk', 'GB'],
      ['gpu', '%'],
    ] as const;

    for (const [type, unit] of families) {
      render(
        <MetricCard
          type={type}
          label={type.toUpperCase()}
          unit={unit}
          value={10}
          isExpanded={false}
          onToggle={() => undefined}
          alertCount={0}
        />,
      );
      expect(screen.getByText('10.0')).toBeInTheDocument();
      cleanup();
    }
  });
});
