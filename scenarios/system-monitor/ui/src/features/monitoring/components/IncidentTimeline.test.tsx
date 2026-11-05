import { fireEvent, screen } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { describe, expect, it, vi } from 'vitest';
import { IncidentTimeline } from './IncidentTimeline';
import type { Investigation, MetricHistory } from '../../../types';
import { InvestigationSchema, InvestigationStatus } from '@vrooli/proto-types/system-monitor/v1/investigations/investigations_pb';

function metricHistory(overrides: Partial<MetricHistory> = {}): MetricHistory {
  return {
    windowSeconds: 3600,
    sampleIntervalSeconds: 60,
    cpu: [],
    memory: [],
    swap: [],
    network: [],
    ...overrides,
  };
}

function investigation(id: string, status: InvestigationStatus, details?: Investigation['details']): Investigation {
  return create(InvestigationSchema, {
    id,
    status,
    details,
    findings: '',
  });
}

describe('IncidentTimeline', () => {
  it('correlates metric crossings and investigations with source actions', () => {
    const onSource = vi.fn();
    const onInvestigate = vi.fn();
    const history = metricHistory({ cpu: [{ timestamp: '2026-01-01T00:00:00Z', value: 96 }] });
    const activeInvestigation = create(InvestigationSchema, {
      ...investigation('inv-1', InvestigationStatus.QUEUED),
      startTime: timestampFromDate(new Date('2026-01-01T00:01:00Z')),
      findings: 'root cause found',
    });
    render(<IncidentTimeline history={history} investigations={[activeInvestigation]} onOpenSource={onSource} onInvestigate={onInvestigate} />);
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
    const completedInvestigation = create(InvestigationSchema, {
      ...investigation('inv-1', InvestigationStatus.COMPLETED),
      startTime: timestampFromDate(new Date('2026-01-01T00:01:00Z')),
      findings: 'done',
    });
    const history = metricHistory();
    render(<IncidentTimeline history={history} investigations={[completedInvestigation]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);

    expect(screen.getByText(/Investigation completed/i)).toBeInTheDocument();
    expect(screen.queryByText(/Investigation 3\b/)).not.toBeInTheDocument();
  });

  it('never grades a connection count against a percentage threshold', () => {
    // Network values are a COUNT. Comparing them against the 80/95 percentage
    // bars made any host with more than 95 open connections report a permanent
    // CRITICAL reading "558.0%" — a unit error, not a plant condition.
    const history = metricHistory({
      network: [{ timestamp: '2026-01-01T00:00:00Z', value: 558 }],
    });
    render(<IncidentTimeline history={history} investigations={[]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);

    expect(screen.queryByText(/Network crossed the attention threshold/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/558\.0%/)).not.toBeInTheDocument();
    // With nothing else in the window, the panel must say so rather than
    // inventing a crossing to fill itself.
    expect(screen.getByText(/No threshold crossings or investigations/i)).toBeInTheDocument();
  });

  it('still reports the percentage series with their unit', () => {
    const history = metricHistory({
      cpu: [{ timestamp: '2026-01-01T00:00:00Z', value: 96 }],
    });
    render(<IncidentTimeline history={history} investigations={[]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);
    expect(screen.getByText(/96\.0% measured in the shared observation window/)).toBeInTheDocument();
  });

  it('shows a truthful empty state', () => {
    render(<IncidentTimeline history={null} investigations={[]} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);
    expect(screen.getByText(/No threshold crossings or investigations/)).toBeInTheDocument();
  });

  it('classifies warning metrics and investigation fallback details', () => {
    const history = metricHistory({
      memory: [{ timestamp: '2026-01-01T00:00:00Z', value: 80 }],
      network: [{ timestamp: '2026-01-01T00:00:00Z', value: 79 }],
      diskUsage: [{ timestamp: '2026-01-01T00:00:00Z', value: 95 }],
    });
    const investigations = [
      investigation('failed', InvestigationStatus.COMPLETED),
      create(InvestigationSchema, {
        ...investigation('details', InvestigationStatus.QUEUED, { attached: true }),
        startTime: timestampFromDate(new Date('2026-01-01T00:00:00Z')),
      }),
    ];
    render(<IncidentTimeline history={history} investigations={investigations} onOpenSource={vi.fn()} onInvestigate={vi.fn()} />);
    expect(screen.getByText('Memory crossed the attention threshold')).toBeInTheDocument();
    expect(screen.getByText(/Investigation details attached/)).toBeInTheDocument();
    expect(screen.getByText(/Investigation activity recorded/)).toBeInTheDocument();
  });
});
