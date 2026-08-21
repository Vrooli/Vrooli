import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import { MetricSparkline } from './MetricSparkline';

const point = (timestamp: string, value: number) => ({ timestamp, value });

describe('MetricSparkline', () => {
  it('shows the collection state when no samples exist', () => {
    render(<MetricSparkline ariaLabel="CPU history" />);
    expect(screen.getByText('Collecting data…')).toBeInTheDocument();
  });

  it('renders one and two point histories with labels and thresholds', () => {
    const one = render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 12)]} unit="%" windowLabel="1h" ariaLabel="CPU history" />);
    expect(screen.getByRole('img', { name: 'CPU history' })).toBeInTheDocument();
    expect(screen.getByText('1h')).toBeInTheDocument();
    one.unmount();

    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)]} threshold={15} valueDomain={[0, 100]} unit="%" />);
    expect(screen.getByRole('img')).toBeInTheDocument();
    expect(screen.getByRole('img').querySelector('line')).toBeInTheDocument();
  });

  it('smooths multi-point data and exposes the nearest hover sample', () => {
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

  it('draws a graticule with a stronger baseline and mid-division', () => {
    render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', 10), point('2026-01-01T00:01:00Z', 20)]} valueDomain={[0, 100]} />);

    const graticule = screen.getByTestId('sparkline-graticule');
    const lines = Array.from(graticule.querySelectorAll('line'));
    expect(lines).toHaveLength(5);

    const strong = lines.filter(line => line.getAttribute('stroke') === 'var(--chart-grid-strong)');
    const faint = lines.filter(line => line.getAttribute('stroke') === 'var(--chart-grid)');
    expect(strong).toHaveLength(2);
    expect(faint).toHaveLength(3);
    for (const line of lines) {
      expect(line.getAttribute('vector-effect')).toBe('non-scaling-stroke');
      // The graticule is horizontal: same y at both ends.
      expect(line.getAttribute('y1')).toBe(line.getAttribute('y2'));
    }
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

  it('downsamples a dense series to a bounded number of real vertices', () => {
    const data = dense(700, idx => (idx % 2 === 0 ? 40 : 70));
    render(<MetricSparkline data={data} valueDomain={[0, 100]} />);

    const svg = screen.getByRole('img');
    expect(svg).toHaveAttribute('data-render-mode', 'lttb');

    const d = screen.getByTestId('sparkline-line').getAttribute('d') ?? '';
    const vertices = (d.match(/C/g) ?? []).length + 1;
    expect(vertices).toBeGreaterThan(1);
    expect(vertices).toBeLessThanOrEqual(150);
    expect(vertices).toBeLessThan(700);
    // No band: the line alone carries the series.
    expect(screen.queryByTestId('sparkline-envelope')).not.toBeInTheDocument();
  });

  it('keeps a sparse series on the original smooth line', () => {
    render(<MetricSparkline data={dense(20, idx => 40 + idx)} valueDomain={[0, 100]} />);

    expect(screen.getByRole('img')).toHaveAttribute('data-render-mode', 'line');
    expect(screen.getByRole('img').querySelector('path[fill^="url("]')).toBeInTheDocument();
  });

  it('starts and ends a downsampled line on the true first and last samples', () => {
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
    // The spike sample itself is selected, so the line reaches the plot ceiling.
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
    expect(screen.queryByText(/–/)).not.toBeInTheDocument();
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
    expect(empty).toHaveTextContent('Collecting data…');
    expect(empty.querySelector('svg')).toBeNull();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
    expect(screen.queryByTestId('sparkline-rail')).not.toBeInTheDocument();
  });

  it('normalizes non-finite and flat domains and supports custom accessibility', () => {
    const { rerender } = render(<MetricSparkline data={[point('2026-01-01T00:00:00Z', Number.NaN)]} valueDomain={[5, 5]} threshold={99} ariaLabel="custom chart" />);
    expect(screen.getByRole('img', { name: 'custom chart' })).toBeInTheDocument();
    rerender(<MetricSparkline data={[point('2026-01-01T00:00:00Z', Number.POSITIVE_INFINITY), point('2026-01-01T00:01:00Z', 2)]} threshold={-1} />);
    expect(screen.getByRole('img')).toBeInTheDocument();
  });
});
