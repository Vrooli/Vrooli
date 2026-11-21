import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import { MetricSparkline, type ChartInspection } from './MetricSparkline';

const point = (timestamp: string, value: number) => ({ timestamp, value });

describe('MetricSparkline', () => {
  it('shows the collection state when no samples exist', () => {
    render(<MetricSparkline ariaLabel="CPU history" />);
    expect(screen.getByText('No observations in this window')).toBeInTheDocument();
  });

  it('renders one and two point histories with labels and thresholds', () => {
    const one = render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 12)]} unit="%" windowLabel="1h" ariaLabel="CPU history" />);
    expect(screen.getByRole('img', { name: 'CPU history' })).toBeInTheDocument();
    expect(screen.getByText(/1h · observed/)).toBeInTheDocument();
    one.unmount();

    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)]} threshold={15} valueDomain={[0, 100]} unit="%" />);
    expect(screen.getByRole('img')).toBeInTheDocument();
    expect(screen.getByRole('img').querySelector('line')).toBeInTheDocument();
  });

  it('exposes the nearest hover sample', () => {
    render(<MetricSparkline data={[
      point('2026-01-01T00:00:00Z', 10),
      point('2026-01-01T00:01:00Z', 30),
      point('2026-01-01T00:02:00Z', 20),
    ]} unit="%" />);
    const svg = screen.getByRole('img');
    Object.defineProperty(svg, 'getBoundingClientRect', { configurable: true, value: () => ({ left: 0, width: 100 }) });
    fireEvent.mouseMove(svg, { clientX: 50 });
    expect(screen.getByText('30.0%')).toBeInTheDocument();
    fireEvent.mouseLeave(svg);
    expect(screen.queryByText('30.0%')).not.toBeInTheDocument();
  });

  it('states observed coverage and gaps even when the requested window is unavailable', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 41), { ...point('2026-01-01T00:00:10Z', 0), gapBefore: true }]} unit="%" />);
    expect(screen.getByText(/Observed .*1 gap/)).toBeInTheDocument();
    expect(document.querySelector('[data-testid="sparkline-line"]')?.getAttribute('d')?.match(/M/g)).toHaveLength(2);
  });

  it('reveals the latest observation on focus without fixing the view', () => {
    const data = [point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 30)];
    const { rerender } = render(<MetricSparkline data={data} unit="%" ariaLabel="CPU history" />);
    const plot = screen.getByRole('img', { name: 'CPU history' });
    fireEvent.focus(plot);
    expect(document.querySelector('.readout-trace__summary')).toHaveTextContent('30.0%');
    const summary = document.querySelector('.readout-trace__summary');
    if (!summary) throw new Error('Expected the summary readout');
    expect(summary.compareDocumentPosition(plot) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(document.querySelector('.readout-trace__tooltip')).not.toBeInTheDocument();
    expect(plot).toHaveAccessibleDescription(/use left and right arrows/);
    expect(screen.getByTestId('sparkline-latest-announcement')).toHaveTextContent('Latest observation: 30.0%');
    expect(screen.queryByRole('button', { name: 'Return to latest' })).not.toBeInTheDocument();
    rerender(<MetricSparkline data={[...data, point('2026-01-01T00:02:00Z', 40)]} unit="%" ariaLabel="CPU history" />);
    expect(document.querySelector('.readout-trace__summary')).toHaveTextContent('40.0%');
    expect(screen.getByTestId('sparkline-latest-announcement')).toHaveTextContent('Latest observation: 40.0%');
    fireEvent.keyDown(plot, { key: 'ArrowLeft' });
    expect(screen.getByTestId('sparkline-latest-announcement')).toBeEmptyDOMElement();
    expect(screen.getByRole('status')).toHaveTextContent('Fixed selection: 30.0%');
    fireEvent.keyDown(plot, { key: 'Escape' });
    expect(screen.getByTestId('sparkline-latest-announcement')).toHaveTextContent('Latest observation: 40.0%');
  });

  it('places samples by time, breaks missing intervals, and supports keyboard inspection', () => {
    render(<MetricSparkline data={[
      point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20),
      point('2026-01-01T00:04:00Z', 30), point('2026-01-01T00:05:00Z', 40),
    ]} sampleIntervalSeconds={60} unit="%" />);
    const line = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    expect(line.match(/M/g)).toHaveLength(2);
    expect(line).toContain('L20.00,');
    expect(line).toContain('M80.00,');
    const chart = screen.getByRole('img');
    fireEvent.keyDown(chart, { key: 'ArrowLeft' });
    expect(screen.getByText('30.0%')).toBeInTheDocument();
    fireEvent.keyDown(chart, { key: 'PageUp' });
    expect(screen.getByRole('status')).toHaveTextContent('20.0%');
    expect(screen.getByRole('button', { name: 'Return to latest' })).toBeInTheDocument();
    fireEvent.keyDown(chart, { key: 'Escape' });
    expect(screen.queryByRole('button', { name: 'Return to latest' })).not.toBeInTheDocument();
  });

  it('places partial coverage within the exact shared query window', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:43:00Z', 10), point('2026-01-01T00:50:00Z', 20)]} windowStart="2026-01-01T00:00:00Z" windowEnd="2026-01-01T01:00:00Z" />);
    const line = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    expect(line).toMatch(/^M71\.67,/);
    expect(line).toContain('L83.33,');
    expect(screen.getByLabelText(/^Window /)).toBeInTheDocument();
  });

  it('holds an inspected view fixed across refresh until returning to latest', () => {
    const initial = [point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)];
    const { rerender } = render(<MetricSparkline data={initial} unit="%" />);
    fireEvent.keyDown(screen.getByRole('img'), { key: 'ArrowLeft' });
    rerender(<MetricSparkline data={[...initial, point('2026-01-01T00:02:00Z', 80)]} unit="%" />);
    expect(screen.getByRole('status')).toHaveTextContent('Fixed selection: 10.0%');
    expect(screen.getByRole('img').querySelector('title')).toHaveTextContent('Latest 20.0%');
    fireEvent.click(screen.getByRole('button', { name: 'Return to latest' }));
    expect(screen.getByRole('img').querySelector('title')).toHaveTextContent('Latest 80.0%');
  });

  it('zooms around an inspected observation and resets to the shared window', () => {
    const data = Array.from({ length: 61 }, (_, i) => point(new Date(Date.UTC(2026, 0, 1, 0, i)).toISOString(), i));
    render(<MetricSparkline data={data} unit="%" windowStart="2026-01-01T00:00:00Z" windowEnd="2026-01-01T01:00:00Z" ariaLabel="CPU history" />);
    fireEvent.keyDown(screen.getByRole('img', { name: 'CPU history' }), { key: 'ArrowLeft' });
    fireEvent.click(screen.getByRole('button', { name: 'Zoom around selected sample' }));
    expect(screen.getByRole('img', { name: 'CPU history' })).toHaveFocus();
    expect(screen.getByText(/Chart zoom:/)).toBeInTheDocument();
    const zoomed = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    expect((zoomed.match(/[ML]/g) ?? []).length).toBeLessThan(61);
    fireEvent.click(screen.getByRole('button', { name: 'Reset chart zoom' }));
    expect(screen.getByRole('img', { name: 'CPU history' })).toHaveFocus();
    expect(screen.queryByText(/Chart zoom:/)).not.toBeInTheDocument();
    expect((screen.getByTestId('sparkline-line').getAttribute('d')?.match(/[ML]/g) ?? []).length).toBe(61);
    expect(screen.getByRole('status')).toHaveTextContent('Fixed selection: 59.0%');
  });

  it('restores a zoomed inspection when the chart remounts', () => {
    const data = Array.from({ length: 61 }, (_, i) => point(new Date(Date.UTC(2026, 0, 1, 0, i)).toISOString(), i));
    const store = new Map<string, ChartInspection>();
    const props = { data, unit: '%', windowStart: '2026-01-01T00:00:00Z', windowEnd: '2026-01-01T01:00:00Z', ariaLabel: 'CPU history', inspectionStore: store, inspectionKey: 'cpu' };
    const first = render(<MetricSparkline {...props} />);
    fireEvent.keyDown(screen.getByRole('img', { name: 'CPU history' }), { key: 'ArrowLeft' });
    fireEvent.click(screen.getByRole('button', { name: 'Zoom around selected sample' }));
    const bounds = screen.getByText(/Chart zoom:/).textContent;
    first.unmount();
    render(<MetricSparkline {...props} />);
    expect(screen.getByText(/Chart zoom:/)).toHaveTextContent(bounds ?? '');
    expect(screen.getByRole('status')).toHaveTextContent('Fixed selection: 59.0%');
    fireEvent.click(screen.getByRole('button', { name: 'Return to latest' }));
    expect(store.has('cpu')).toBe(false);
  });

  it('selects an observed sample on a touch-style click without a prior hover', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)]} unit="%" />);
    const chart = screen.getByRole('img');
    Object.defineProperty(chart, 'getBoundingClientRect', { configurable: true, value: () => ({ left: 0, width: 100 }) });
    fireEvent.click(chart, { clientX: 0 });
    expect(screen.getByRole('button', { name: 'Return to latest' })).toBeInTheDocument();
    expect(document.querySelector('.readout-trace__summary')).toHaveTextContent('10.0%');
  });

  it('provides an accessible table of observed samples', async () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 4), point('2026-01-01T00:01:00Z', 8)]} unit="%" ariaLabel="CPU history" />);
    fireEvent.click(screen.getByText('View observed samples'));
    expect(await screen.findByRole('table', { name: 'CPU history — observed samples' })).toBeInTheDocument();
    expect(screen.getByRole('table')).toHaveTextContent('4.0%');
    expect(screen.getByRole('table')).toHaveTextContent('8.0%');
  });

  it('draws a graticule with a stronger baseline and mid-division', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)]} valueDomain={[0, 100]} />);

    const graticule = screen.getByTestId('sparkline-graticule');
    const lines = Array.from(graticule.querySelectorAll('line'));
    expect(lines).toHaveLength(10);

    const strong = lines.filter(line => line.getAttribute('stroke') === 'var(--chart-grid-strong)');
    const faint = lines.filter(line => line.getAttribute('stroke') === 'var(--chart-grid)');
    expect(strong).toHaveLength(2);
    expect(faint).toHaveLength(8);
    for (const line of lines) {
      expect(line.getAttribute('vector-effect')).toBe('non-scaling-stroke');
    }
    expect(lines.filter(line => line.getAttribute('y1') === line.getAttribute('y2'))).toHaveLength(5);
    expect(lines.filter(line => line.getAttribute('x1') === line.getAttribute('x2'))).toHaveLength(5);
  });

  it('labels the vertical extent on a rail outside the plot', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)]} valueDomain={[0, 100]} unit="%" />);

    const rail = screen.getByTestId('sparkline-rail');
    expect(rail).toHaveTextContent('100%');
    expect(rail).toHaveTextContent('0%');
    // The rail annotates the chart; it must not be inside it.
    expect(rail.closest('svg')).toBeNull();
  });

  it('falls back to the series extent when no domain is supplied', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 4), point('2026-01-01T00:01:00Z', 37)]} />);

    const rail = screen.getByTestId('sparkline-rail');
    expect(rail).toHaveTextContent('37');
    expect(rail).toHaveTextContent('4');
  });

  it('keeps the stroke weight uniform regardless of the horizontal stretch', () => {
    render(<MetricSparkline data={[
      point('2026-01-01T00:00:00Z', 10),
      point('2026-01-01T00:01:00Z', 30),
      point('2026-01-01T00:02:00Z', 20),
    ]} threshold={25} />);

    const svg = screen.getByRole('img');
    const strokedPaths = Array.from(svg.querySelectorAll('path')).filter(
      path => path.getAttribute('fill') === 'none',
    );
    expect(strokedPaths.length).toBeGreaterThan(0);
    for (const path of strokedPaths) {
      expect(path.getAttribute('vector-effect')).toBe('non-scaling-stroke');
    }
  });

  it('haloes the newest sample and moves the dot to the hovered sample', () => {
    render(<MetricSparkline data={[
      point('2026-01-01T00:00:00Z', 10),
      point('2026-01-01T00:01:00Z', 30),
      point('2026-01-01T00:02:00Z', 20),
    ]} />);

    expect(screen.getByTestId('sparkline-endpoint-halo')).toBeInTheDocument();
    const dot = screen.getByTestId('sparkline-endpoint');
    expect(dot.style.left).toBe('100%');

    const svg = screen.getByRole('img');
    Object.defineProperty(svg, 'getBoundingClientRect', { configurable: true, value: () => ({ left: 0, width: 100 }) });
    fireEvent.mouseMove(svg, { clientX: 0 });
    // While the reader is inspecting a sample, the live halo stands down.
    expect(screen.queryByTestId('sparkline-endpoint-halo')).not.toBeInTheDocument();
    expect(screen.getByTestId('sparkline-endpoint').style.left).toBe('0%');
  });

  // A 60-minute window at the poll interval is several hundred samples across
  // a few hundred pixels; drawn one vertex per sample the stroke fills the
  // band and the chart reads as a solid slab.
  const dense = (count: number, valueAt: (idx: number) => number) =>
    Array.from({ length: count }, (_, idx) => point(
      new Date(Date.UTC(2026, 0, 1, 0, 0, idx)).toISOString(),
      valueAt(idx),
    ));

  it('plots every observed sample in a dense series', () => {
    const data = dense(700, idx => (idx % 2 === 0 ? 40 : 70));
    render(<MetricSparkline data={data} valueDomain={[0, 100]} />);

    const svg = screen.getByRole('img');
    expect(svg).toHaveAttribute('data-render-mode', 'line');

    const d = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    const vertices = (d.match(/L/g) ?? []).length + 1;
    expect(vertices).toBe(700);
    // No band: the line alone carries the series.
    expect(screen.queryByTestId('sparkline-envelope')).not.toBeInTheDocument();
  });

  it('keeps a sparse series on straight observed segments', () => {
    render(<MetricSparkline data={dense(20, idx => 40 + idx)} valueDomain={[0, 100]} />);

    expect(screen.getByRole('img')).toHaveAttribute('data-render-mode', 'line');
    expect(screen.getByRole('img').querySelector('path[fill^="url("]')).toBeInTheDocument();
  });

  it('starts and ends a dense line on the true first and last samples', () => {
    // Distinct endpoints so their positions are unambiguous.
    const data = dense(700, idx => (idx === 0 ? 10 : idx === 699 ? 90 : 50));
    render(<MetricSparkline data={data} valueDomain={[0, 100]} />);

    const d = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    // height 48, inset 4 => y = 44 - value/100 * 40.
    const firstY = Number(/^M(?:-?[\d.]+),(-?[\d.]+)/.exec(d)?.[1]);
    const lastY = Number(/(?:-?[\d.]+),(-?[\d.]+)$/.exec(d)?.[1]);
    expect(firstY).toBeCloseTo(40, 5);
    expect(lastY).toBeCloseTo(8, 5);
  });

  it('retains a peak that averaging would have flattened away', () => {
    // One spike to the top of the domain, buried among 700 mid-band samples.
    render(<MetricSparkline data={dense(700, idx => (idx === 337 ? 100 : 50))} valueDomain={[0, 100]} />);

    const d = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    const ys = [...d.matchAll(/(?:-?[\d.]+),(-?[\d.]+)/g)].map(match => Number(match[1]));
    // The spike sample is plotted, so the line reaches the plot ceiling.
    expect(Math.min(...ys)).toBeCloseTo(4, 5);
  });

  it('pins the endpoint marker to the true latest sample, not to an aggregate', () => {
    render(<MetricSparkline data={dense(700, idx => (idx === 699 ? 90 : 20))} valueDomain={[0, 100]} />);

    const dot = screen.getByTestId('sparkline-endpoint');
    expect(dot.style.left).toBe('100%');
    // 90% of a [0,100] domain => y = 44 - 0.9 * 40 = 8.
    expect(dot.style.top).toBe('8px');
    expect(screen.getByTestId('sparkline-endpoint-halo').style.top).toBe('8px');
  });

  it('reports a hovered sample as a single real reading with no aggregate label', () => {
    render(<MetricSparkline data={dense(700, idx => (idx % 2 === 0 ? 40 : 70))} valueDomain={[0, 100]} unit="%" />);

    const svg = screen.getByRole('img');
    Object.defineProperty(svg, 'getBoundingClientRect', { configurable: true, value: () => ({ left: 0, width: 100 }) });
    fireEvent.mouseMove(svg, { clientX: 0 });

    // Every rendered vertex is a real sample, so the read-out is one of them.
    expect(screen.getByText(/^(40\.0|70\.0)%$/)).toBeInTheDocument();
    expect(screen.queryByText(/mean/i)).not.toBeInTheDocument();
    expect(document.querySelector('.readout-trace__summary')).not.toHaveTextContent('–');
  });

  it('gives a near-constant series a readable domain instead of a degenerate axis', () => {
    // 0.6 MB/s of wobble around 23.5 previously scaled to the full plot height
    // and printed a "24 / 23" rail as though it framed a real span.
    render(<MetricSparkline data={dense(40, idx => 23.2 + (idx % 3) * 0.3)} />);

    const rail = screen.getByTestId('sparkline-rail');
    const bounds = Array.from(rail.querySelectorAll('span')).map(span => Number(span.textContent));
    const [high, low] = bounds as [number, number];
    // The rail states a span the reader can act on, not two rounded neighbours.
    expect(high - low).toBeGreaterThanOrEqual(2);
    expect(rail.textContent).not.toBe('2423');

    // And the line stays in the middle of the plot rather than sweeping it.
    const d = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    const ys = [...d.matchAll(/(?:-?[\d.]+),(-?[\d.]+)/g)].map(match => Number(match[1]));
    expect(Math.min(...ys)).toBeGreaterThan(10);
    expect(Math.max(...ys)).toBeLessThan(38);
  });

  it('never widens a domain the caller stated explicitly', () => {
    render(<MetricSparkline data={dense(40, () => 23.5)} valueDomain={[0, 100]} unit="%" />);

    const rail = screen.getByTestId('sparkline-rail');
    expect(rail).toHaveTextContent('100%');
    expect(rail).toHaveTextContent('0%');
  });

  it('rounds read-outs to the requested precision', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 499.4), point('2026-01-01T00:01:00Z', 512.6)]} unit=" connections" precision={0} />);

    const svg = screen.getByRole('img');
    Object.defineProperty(svg, 'getBoundingClientRect', { configurable: true, value: () => ({ left: 0, width: 100 }) });
    fireEvent.mouseMove(svg, { clientX: 100 });
    expect(screen.getByText('513 connections')).toBeInTheDocument();
  });

  it('renders a composed empty state rather than an empty chart', () => {
    render(<MetricSparkline ariaLabel="CPU history" />);

    const empty = screen.getByTestId('sparkline-empty');
    expect(empty).toHaveTextContent('No observations in this window');
    expect(empty.querySelector('svg')).toBeNull();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
    expect(screen.queryByTestId('sparkline-rail')).not.toBeInTheDocument();
  });

  it('normalizes non-finite and flat domains and supports custom accessibility', () => {
    const { rerender } = render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', Number.NaN)]} valueDomain={[5, 5]} threshold={99} ariaLabel="custom chart" />);
    expect(screen.getByTestId('sparkline-empty')).toBeInTheDocument();
    rerender(<MetricSparkline data={[point('2026-01-01T00:00:00Z', Number.POSITIVE_INFINITY), point('2026-01-01T00:01:00Z', 2)]} threshold={-1} />);
    expect(screen.getByRole('img')).toBeInTheDocument();
  });
});
