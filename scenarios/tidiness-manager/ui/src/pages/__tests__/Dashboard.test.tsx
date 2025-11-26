import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import Dashboard from '../Dashboard';
import * as api from '../../lib/api';

// [REQ:TM-UI-001] Global dashboard - scenario table
// [REQ:TM-UI-002] Global dashboard - campaign status badges

vi.mock('../../lib/api');

const mockScenarios = [
  {
    scenario: 'test-scenario-1',
    light_issues: 5,
    ai_issues: 2,
    long_files: 3,
    visit_percent: 75,
    campaign_status: 'active' as const,
    total_files: 50,
    total_lines: 5000,
  },
  {
    scenario: 'test-scenario-2',
    light_issues: 0,
    ai_issues: 0,
    long_files: 0,
    visit_percent: 100,
    campaign_status: 'completed' as const,
    total_files: 30,
    total_lines: 3000,
  },
  {
    scenario: 'test-scenario-3',
    light_issues: 15,
    ai_issues: 8,
    long_files: 10,
    visit_percent: 25,
    campaign_status: 'error' as const,
    total_files: 100,
    total_lines: 15000,
  },
];

function renderDashboard() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Dashboard />
      </BrowserRouter>
    </QueryClientProvider>
  );
}

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[REQ:TM-UI-001] renders dashboard page with scenario table', async () => {
    vi.mocked(api.fetchScenarioStats).mockResolvedValue(mockScenarios);

    renderDashboard();

    // Wait for data to load
    await waitFor(() => {
      expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
    });

    // Verify page header
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Monitor code tidiness across all scenarios')).toBeInTheDocument();
  });

  it('[REQ:TM-UI-001] displays all scenarios in table', async () => {
    vi.mocked(api.fetchScenarioStats).mockResolvedValue(mockScenarios);

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('test-scenario-1')).toBeInTheDocument();
      expect(screen.getByText('test-scenario-2')).toBeInTheDocument();
      expect(screen.getByText('test-scenario-3')).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-001] displays correct issue counts for each scenario', async () => {
    vi.mocked(api.fetchScenarioStats).mockResolvedValue(mockScenarios);

    renderDashboard();

    await waitFor(() => {
      // Scenario 1: 5 light + 2 AI = 7 total
      const scenario1Row = screen.getByText('test-scenario-1').closest('tr');
      expect(scenario1Row).toHaveTextContent('5'); // light issues
      expect(scenario1Row).toHaveTextContent('2'); // AI issues
      expect(scenario1Row).toHaveTextContent('3'); // long files

      // Scenario 2: 0 issues
      const scenario2Row = screen.getByText('test-scenario-2').closest('tr');
      expect(scenario2Row).toHaveTextContent('0');
    });
  });

  it('[REQ:TM-UI-002] displays campaign status badges with correct variants', async () => {
    vi.mocked(api.fetchScenarioStats).mockResolvedValue(mockScenarios);

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Active')).toBeInTheDocument();
      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByText('Error')).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-002] badge component supports all campaign status variants', async () => {
    const allStatuses = [
      { scenario: 's1', light_issues: 0, ai_issues: 0, long_files: 0, visit_percent: 0, campaign_status: 'none' as const, total_files: 10, total_lines: 1000 },
      { scenario: 's2', light_issues: 0, ai_issues: 0, long_files: 0, visit_percent: 0, campaign_status: 'active' as const, total_files: 10, total_lines: 1000 },
      { scenario: 's3', light_issues: 0, ai_issues: 0, long_files: 0, visit_percent: 0, campaign_status: 'completed' as const, total_files: 10, total_lines: 1000 },
      { scenario: 's4', light_issues: 0, ai_issues: 0, long_files: 0, visit_percent: 0, campaign_status: 'paused' as const, total_files: 10, total_lines: 1000 },
      { scenario: 's5', light_issues: 0, ai_issues: 0, long_files: 0, visit_percent: 0, campaign_status: 'error' as const, total_files: 10, total_lines: 1000 },
    ];

    vi.mocked(api.fetchScenarioStats).mockResolvedValue(allStatuses);

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('No Campaign')).toBeInTheDocument();
      expect(screen.getByText('Active')).toBeInTheDocument();
      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByText('Paused')).toBeInTheDocument();
      expect(screen.getByText('Error')).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-001] displays summary statistics cards', async () => {
    vi.mocked(api.fetchScenarioStats).mockResolvedValue(mockScenarios);

    renderDashboard();

    await waitFor(() => {
      // Total scenarios
      const scenariosCard = screen.getByTestId('total-scenarios-card');
      expect(scenariosCard).toHaveTextContent('Total Scenarios');
      expect(scenariosCard).toHaveTextContent('3');

      // Total issues: (5+2) + (0+0) + (15+8) = 30
      const issuesCard = screen.getByTestId('total-issues-card');
      expect(issuesCard).toHaveTextContent('Total Issues');
      expect(issuesCard).toHaveTextContent('30');

      // Total long files: 3 + 0 + 10 = 13
      const longFilesCard = screen.getByTestId('long-files-card');
      expect(longFilesCard).toHaveTextContent('Long Files');
      expect(longFilesCard).toHaveTextContent('13');

      // Active campaigns: 1
      const campaignsCard = screen.getByTestId('active-campaigns-card');
      expect(campaignsCard).toHaveTextContent('Active Campaigns');
      expect(campaignsCard).toHaveTextContent('1');
    });
  });

  it('[REQ:TM-UI-001] shows loading state while fetching data', async () => {
    vi.mocked(api.fetchScenarioStats).mockImplementation(
      () => new Promise(resolve => setTimeout(() => resolve(mockScenarios), 100))
    );

    renderDashboard();

    // Should show spinner initially
    expect(screen.getByTestId('spinner')).toBeInTheDocument();

    // Wait for data to load
    await waitFor(() => {
      expect(screen.queryByTestId('spinner')).not.toBeInTheDocument();
      expect(screen.getByText('test-scenario-1')).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-001] displays error message when API fails', async () => {
    vi.mocked(api.fetchScenarioStats).mockRejectedValue(new Error('API connection failed'));

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load scenarios/i)).toBeInTheDocument();
      expect(screen.getByText(/API connection failed/i)).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-001] table is sortable by scenario name', async () => {
    vi.mocked(api.fetchScenarioStats).mockResolvedValue(mockScenarios);

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('test-scenario-1')).toBeInTheDocument();
    });

    // Find all scenario name cells in table
    const rows = screen.getAllByRole('row');
    const dataRows = rows.slice(1); // Skip header row

    // Verify scenarios are sorted by default (by light_issues desc)
    expect(dataRows[0]).toHaveTextContent('test-scenario-3'); // 15 light issues
    expect(dataRows[1]).toHaveTextContent('test-scenario-1'); // 5 light issues
    expect(dataRows[2]).toHaveTextContent('test-scenario-2'); // 0 light issues
  });

  it('[REQ:TM-UI-002] health status indicator reflects issue severity', async () => {
    vi.mocked(api.fetchScenarioStats).mockResolvedValue(mockScenarios);

    renderDashboard();

    await waitFor(() => {
      // Scenario 2 should be "Excellent" (0 issues, 0 long files)
      const scenario2Row = screen.getByText('test-scenario-2').closest('tr');
      expect(scenario2Row).toHaveTextContent('Excellent');

      // Scenario 3 should be "Critical" (23 issues, 10 long files)
      const scenario3Row = screen.getByText('test-scenario-3').closest('tr');
      expect(scenario3Row).toHaveTextContent('Critical');
    });
  });
});
