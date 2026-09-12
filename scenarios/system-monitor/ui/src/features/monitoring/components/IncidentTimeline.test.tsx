import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { describe, expect, it, vi } from 'vitest';
import { IncidentTimeline } from './IncidentTimeline';
import type { Investigation, MetricHistory } from '../../../types';

describe('IncidentTimeline', () => {
  it('correlates metric crossings and investigations with source actions', () => {
    const onSource = vi.fn();
    const onInvestigate = vi.fn();
    const history = { cpu: [{ timestamp: '2026-01-01T00:00:00Z', value: 96 }], memory: [], network: [], diskUsage: [] } as unknown as MetricHistory;
    const investigation = { id: 'inv-1', status: 1, startTime: timestampFromDate(new Date('2026-01-01T00:01:00Z')), findings: 'root cause found' } as unknown as Investigation;
    render(<IncidentTimeline history={history} investigations={[investigation]} onOpenSource={onSource} onInvestigate={onInvestigate} />);
    expect(screen.getByText('CPU crossed the attention threshold')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /CPU crossed/ }));
    expect(screen.getByRole('dialog', { name: 'Correlated incident view' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Open logs' }));
    fireEvent.click(screen.getByRole('button', { name: 'Open forensics' }));
    fireEvent.click(screen.getByRole('button', { name: 'Investigate this window' }));
    expect(onSource).toHaveBeenNthCalledWith(1, 'logs');
    expect(onSource).toHaveBeenNthCalledWith(2, 'forensics');
    expect(onInvestigate).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('names the investigation status in words, never as the raw enum ordinal', () => {
    // `status` is a numeric proto enum. Interpolating it produced literal
    // titles like "Investigation 3" in the shipped UI.
    const investigation = { id: 'inv-1', status: 3, startTime: timestampFromDate(new Date('2026-01-01T00:01:00Z')), findings: 'done' } as unknown as Investigation;
    const history = { cpu: [], memory: [], network: [], diskUsage: [] } as unknown as MetricHistory;
    render(<IncidentTimeline history={history} investigations={[investigation]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);

    expect(screen.getByText(/Investigation completed/i)).toBeInTheDocument();
    expect(screen.queryByText(/Investigation 3\b/)).not.toBeInTheDocument();
  });

  it('never grades a connection count against a percentage threshold', () => {
    // Network values are a COUNT. Comparing them against the 80/95 percentage
    // bars made any host with more than 95 open connections report a permanent
    // CRITICAL reading "558.0%" — a unit error, not a plant condition.
    const history = {
      cpu: [],
      memory: [],
      network: [{ timestamp: '2026-01-01T00:00:00Z', value: 558 }],
      diskUsage: [],
    } as unknown as MetricHistory;
    render(<IncidentTimeline history={history} investigations={[]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);

    expect(screen.queryByText(/Network crossed the attention threshold/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/558\.0%/)).not.toBeInTheDocument();
    // With nothing else in the window, the panel must say so rather than
    // inventing a crossing to fill itself.
    expect(screen.getByText(/No threshold crossings or investigations/i)).toBeInTheDocument();
  });

  it('still reports the percentage series with their unit', () => {
    const history = {
      cpu: [{ timestamp: '2026-01-01T00:00:00Z', value: 96 }],
      memory: [], network: [], diskUsage: [],
    } as unknown as MetricHistory;
    render(<IncidentTimeline history={history} investigations={[]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);
    expect(screen.getByText(/96\.0% measured in the shared observation window/)).toBeInTheDocument();
  });

  it('shows a truthful empty state', () => {
    render(<IncidentTimeline history={null} investigations={[]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);
    expect(screen.getByText(/No threshold crossings or investigations/)).toBeInTheDocument();
  });

  it('classifies warning metrics and investigation fallback details', () => {
    const history = { cpu: [], memory: [{ timestamp: '2026-01-01T00:00:00Z', value: 80 }], network: [{ timestamp: '2026-01-01T00:00:00Z', value: 79 }], diskUsage: [{ timestamp: '2026-01-01T00:00:00Z', value: 95 }] } as unknown as MetricHistory;
    const investigations = [
      { id: 'failed', status: 3, startTime: undefined, findings: '', details: undefined },
      { id: 'details', status: 1, startTime: timestampFromDate(new Date('2026-01-01T00:00:00Z')), findings: '', details: { attached: true } },
    ] as unknown as Investigation[];
    render(<IncidentTimeline history={history} investigations={investigations} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);
    expect(screen.getByText('Memory crossed the attention threshold')).toBeInTheDocument();
    expect(screen.getByText(/Investigation details attached/)).toBeInTheDocument();
    expect(screen.getByText(/Investigation activity recorded/)).toBeInTheDocument();
  });
});
