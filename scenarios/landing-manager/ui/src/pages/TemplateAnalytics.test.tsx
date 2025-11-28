// [REQ:TMPL-GENERATION-ANALYTICS] Focused UI test for factory-level analytics requirement
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { AnalyticsPanel } from '../components/AnalyticsPanel';
import * as api from '../lib/api';

vi.mock('../lib/api');

const mockEmptyAnalytics: api.AnalyticsSummary = {
  total_generations: 0,
  successful_count: 0,
  failed_count: 0,
  dry_run_count: 0,
  success_rate: 0,
  by_template: {},
  average_duration_ms: 0,
  recent_events: [],
};

const mockAnalyticsSummary: api.AnalyticsSummary = {
  total_generations: 15,
  successful_count: 12,
  failed_count: 2,
  dry_run_count: 1,
  success_rate: 80,
  by_template: {
    'saas-landing-page': 10,
    'lead-magnet': 5,
  },
  average_duration_ms: 2500,
  recent_events: [
    {
      event_id: '20251128-120000.001',
      template_id: 'saas-landing-page',
      scenario_id: 'my-landing',
      is_dry_run: false,
      success: true,
      timestamp: '2025-11-28T12:00:00Z',
      duration_ms: 2000,
    },
    {
      event_id: '20251128-110000.001',
      template_id: 'lead-magnet',
      scenario_id: 'test-magnet',
      is_dry_run: true,
      success: true,
      timestamp: '2025-11-28T11:00:00Z',
      duration_ms: 500,
    },
    {
      event_id: '20251128-100000.001',
      template_id: 'saas-landing-page',
      scenario_id: 'failed-landing',
      is_dry_run: false,
      success: false,
      error_reason: 'template not found',
      timestamp: '2025-11-28T10:00:00Z',
      duration_ms: 100,
    },
  ],
  first_generation: '2025-11-25T08:00:00Z',
  last_generation: '2025-11-28T12:00:00Z',
};

describe('[REQ:TMPL-GENERATION-ANALYTICS] Factory Analytics - UI Layer', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('should render analytics panel with loading state', async () => {
    vi.mocked(api.getAnalyticsSummary).mockImplementation(
      () => new Promise(() => {}) // Never resolves to show loading
    );

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const panel = screen.getByTestId('analytics-panel');
      expect(panel).toBeInTheDocument();
    });
  });

  it('should display empty state when no generations exist', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockEmptyAnalytics);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const emptyState = screen.getByTestId('analytics-empty-state');
      expect(emptyState).toBeInTheDocument();
      expect(emptyState).toHaveTextContent(/No generation events yet/i);
    });
  });

  it('should display total generations count', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const totalElement = screen.getByTestId('analytics-total');
      expect(totalElement).toBeInTheDocument();
      expect(totalElement).toHaveTextContent('15');
    });
  });

  it('should display success rate as percentage', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const successRateElement = screen.getByTestId('analytics-success-rate');
      expect(successRateElement).toBeInTheDocument();
      expect(successRateElement).toHaveTextContent('80%');
    });
  });

  it('should display successful count with emerald color', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const successfulElement = screen.getByTestId('analytics-successful');
      expect(successfulElement).toBeInTheDocument();
      expect(successfulElement).toHaveTextContent('12');
      expect(successfulElement).toHaveClass('text-emerald-400');
    });
  });

  it('should display average duration formatted correctly', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const durationElement = screen.getByTestId('analytics-avg-duration');
      expect(durationElement).toBeInTheDocument();
      expect(durationElement).toHaveTextContent('2.5s');
    });
  });

  it('should display template breakdown with counts', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const saasTemplate = screen.getByTestId('analytics-template-saas-landing-page');
      expect(saasTemplate).toBeInTheDocument();
      expect(saasTemplate).toHaveTextContent('saas-landing-page');
      expect(saasTemplate).toHaveTextContent('10');

      const leadTemplate = screen.getByTestId('analytics-template-lead-magnet');
      expect(leadTemplate).toBeInTheDocument();
      expect(leadTemplate).toHaveTextContent('lead-magnet');
      expect(leadTemplate).toHaveTextContent('5');
    });
  });

  it('should expand recent events when clicked', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      expect(screen.getByText(/Recent Events/i)).toBeInTheDocument();
    });

    // Click to expand
    const expandButton = screen.getByText(/Recent Events/i);
    fireEvent.click(expandButton);

    await waitFor(() => {
      const recentEvents = screen.getByTestId('analytics-recent-events');
      expect(recentEvents).toBeInTheDocument();
    });
  });

  it('should show dry-run badge for dry-run events', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      expect(screen.getByText(/Recent Events/i)).toBeInTheDocument();
    });

    // Expand events
    fireEvent.click(screen.getByText(/Recent Events/i));

    await waitFor(() => {
      const dryRunBadge = screen.getByText('dry-run');
      expect(dryRunBadge).toBeInTheDocument();
    });
  });

  it('should handle refresh button click', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-total')).toBeInTheDocument();
    });

    // Click refresh button
    const refreshButton = screen.getByRole('button', { name: /Refresh analytics/i });
    fireEvent.click(refreshButton);

    // Should call API again
    await waitFor(() => {
      expect(api.getAnalyticsSummary).toHaveBeenCalledTimes(2);
    });
  });

  it('should display error message when API fails', async () => {
    vi.mocked(api.getAnalyticsSummary).mockRejectedValue(new Error('Failed to fetch analytics'));

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const errorMessage = screen.getByRole('alert');
      expect(errorMessage).toBeInTheDocument();
      expect(errorMessage).toHaveTextContent(/Failed to fetch analytics/i);
    });
  });

  it('should show Factory Analytics heading', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      expect(screen.getByText('Factory Analytics')).toBeInTheDocument();
    });
  });

  it('should have accessible role and aria-label', async () => {
    vi.mocked(api.getAnalyticsSummary).mockResolvedValue(mockAnalyticsSummary);

    render(<AnalyticsPanel />);

    await waitFor(() => {
      const panel = screen.getByTestId('analytics-panel');
      expect(panel).toHaveAttribute('role', 'region');
      expect(panel).toHaveAttribute('aria-label', 'Generation analytics');
    });
  });
});
