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
    const { container } = render(
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

    // A connection count is an integer; see 'renders counts as integers…'.
    // Scoped to the figure: the chart's value rail also carries bare numbers.
    expect(container.querySelector('.readout-figure__value')).toHaveTextContent('12');
    expect(screen.queryByRole('meter')).toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
  });

  it('renders an unread track instead of a fabricated zero-length bar', () => {
    const { container } = renderCard();

    const unread = screen.getByTestId('metric-unread-bar');
    expect(unread).toBeInTheDocument();
    expect(container.querySelector('.metric-fill')).toBeNull();
    expect(container.querySelector('.readout-card')).toHaveAttribute('data-severity', 'unread');
  });

  it('never draws a trace beneath an unavailable collector', () => {
    render(
      <MetricCard
        type="gpu"
        label="GPU"
        unit="%"
        metric={create(MetricValueSchema, {
          state: { case: 'unsupportedReason', value: 'GPU collector unavailable' },
        })}
        history={[
          { timestamp: '2026-01-01T00:00:00Z', value: 40 },
          { timestamp: '2026-01-01T00:01:00Z', value: 55 },
        ]}
        historyWindowSeconds={3600}
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={0}
      />,
    );

    expect(screen.getByText('GPU collector unavailable')).toBeInTheDocument();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
    expect(screen.queryByTestId('sparkline-rail')).not.toBeInTheDocument();
    expect(screen.getByTestId('metric-unread-bar')).toBeInTheDocument();
  });

  it('carries severity on a non-colour channel driven by the same thresholds', () => {
    const severityFor = (value: number) => {
      const { container, unmount } = render(
        <MetricCard
          type="cpu"
          label="CPU"
          unit="%"
          metric={create(MetricValueSchema, { state: { case: 'measured', value } })}
          isExpanded={false}
          onToggle={() => undefined}
          alertCount={0}
        />,
      );
      const severity = container.querySelector('.readout-card')?.getAttribute('data-severity');
      unmount();
      return severity;
    };

    expect(severityFor(12)).toBe('nominal');
    expect(severityFor(70)).toBe('elevated');
    expect(severityFor(90)).toBe('critical');
  });

  it('composes the figure and its unit as one lockup', () => {
    const { container } = render(
      <MetricCard
        type="cpu"
        label="CPU"
        unit="%"
        metric={create(MetricValueSchema, { state: { case: 'measured', value: 56.94 } })}
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={0}
      />,
    );

    const lockup = container.querySelector('.readout-figure');
    expect(lockup).toHaveTextContent('56.9');
    expect(lockup?.querySelector('.readout-figure__unit')?.textContent).toBe('%');
  });

  it('omits the unit when there is no reading to qualify', () => {
    const { container } = renderCard();
    expect(container.querySelector('.readout-figure__unit')).toBeNull();
  });

  it('toggles once from the expand control and not at all from the detail action', () => {
    const onToggle = vi.fn();
    const onOpenDetails = vi.fn();
    render(
      <MetricCard
        type="cpu"
        label="CPU"
        unit="%"
        value={20}
        isExpanded={false}
        onToggle={onToggle}
        onOpenDetails={onOpenDetails}
        alertCount={0}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Expand CPU' }));
    expect(onToggle).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole('button', { name: 'View detail' }));
    expect(onOpenDetails).toHaveBeenCalledOnce();
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it('covers the metric-family color and unit policies', () => {
    // [type, unit, rendered figure]. Counts are integers, everything else
    // carries one decimal.
    const families = [
      ['cpu', '%', '10.0'],
      ['memory', 'MB', '10.0'],
      ['network', 'connections', '10'],
      ['disk', 'GB', '10.0'],
      ['gpu', '%', '10.0'],
    ] as const;

    for (const [type, unit, figure] of families) {
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
      expect(screen.getByText(figure)).toBeInTheDocument();
      cleanup();
    }
  });

  it('draws every family with the same series colour so the stripe is the only state channel', () => {
    // A trace keyed off the card type put an amber "threshold crossed" stripe
    // above a blue line — two channels disagreeing about one reading.
    for (const type of ['cpu', 'memory', 'network', 'disk', 'gpu'] as const) {
      const { container } = render(
        <MetricCard
          type={type}
          label={type.toUpperCase()}
          unit="%"
          value={95}
          history={[
            { timestamp: '2026-01-01T00:00:00Z', value: 91 },
            { timestamp: '2026-01-01T00:01:00Z', value: 95 },
          ]}
          isExpanded={false}
          onToggle={() => undefined}
          alertCount={0}
        />,
      );
      expect(container.querySelector('.readout-card')).toHaveAttribute('data-severity', 'critical');
      expect(container.querySelector('[data-testid="sparkline-line"]')?.getAttribute('stroke'))
        .toBe('var(--chart-line-1)');
      cleanup();
    }
  });

  it('renders counts as integers with a real unit rather than a "#" glyph', () => {
    const { container } = render(
      <MetricCard
        type="network"
        label="NETWORK & CONNECTIONS"
        unit="#"
        metric={create(MetricValueSchema, { state: { case: 'measured', value: 499 } })}
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={0}
      />,
    );

    expect(screen.getByText('499')).toBeInTheDocument();
    expect(screen.queryByText('499.0')).not.toBeInTheDocument();
    expect(container.querySelector('.readout-figure__unit')?.textContent).toBe('connections');
    expect(screen.queryByText('#')).not.toBeInTheDocument();
    expect(screen.getByRole('meter')).toHaveAttribute(
      'aria-label',
      'NETWORK & CONNECTIONS: 499 connections',
    );
  });

  it('does not grade a connection count against percentage thresholds', () => {
    // 551 open connections is not "551 percent". Comparing a count against the
    // 70/90 percentage bars pinned the network card permanently to critical.
    render(
      <MetricCard
        type="network"
        label="Network & connections"
        unit="connections"
        value={551}
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={0}
      />,
    );
    const card = document.querySelector('.readout-card');
    expect(card).toHaveAttribute('data-severity', 'ungraded');
    // And no percentage track, which a count would have filled completely...
    expect(document.querySelector('.metric-fill')).not.toBeInTheDocument();
    // ...nor the unread treatment, because the value WAS read.
    expect(screen.queryByTestId('metric-unread-bar')).not.toBeInTheDocument();
  });

  it('still grades a percentage metric against the percentage bars', () => {
    render(
      <MetricCard
        type="cpu"
        label="CPU usage"
        unit="%"
        value={96}
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={0}
      />,
    );
    expect(document.querySelector('.readout-card')).toHaveAttribute('data-severity', 'critical');
  });
});
