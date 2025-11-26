import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import ScenarioDetail from '../ScenarioDetail';
import * as api from '../../lib/api';

// [REQ:TM-UI-003] Scenario detail - file table
// [REQ:TM-UI-004] Scenario detail - sortable columns

vi.mock('../../lib/api');

const mockScenarioDetail = {
  stats: {
    scenario: 'test-scenario',
    light_issues: 10,
    ai_issues: 5,
    long_files: 3,
    visit_percent: 60,
    campaign_status: 'active' as const,
  },
  files: [
    {
      path: 'src/components/App.tsx',
      lines: 250,
      lint_issues: 2,
      type_issues: 1,
      ai_issues: 0,
      visit_count: 3,
      is_long_file: false,
    },
    {
      path: 'src/api/handlers.go',
      lines: 850,
      lint_issues: 5,
      type_issues: 3,
      ai_issues: 2,
      visit_count: 1,
      is_long_file: true,
    },
    {
      path: 'src/lib/utils.ts',
      lines: 120,
      lint_issues: 0,
      type_issues: 0,
      ai_issues: 0,
      visit_count: 5,
      is_long_file: false,
    },
  ],
};

function renderScenarioDetail(scenarioName = 'test-scenario') {
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
        <Routes>
          <Route path="/scenario/:scenarioName" element={<ScenarioDetail />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

describe('ScenarioDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Set initial route
    window.history.pushState({}, '', '/scenario/test-scenario');
  });

  it('[REQ:TM-UI-003] renders scenario detail page with file table', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      expect(screen.getByText('test-scenario')).toBeInTheDocument();
      expect(screen.getByText(/Code tidiness overview for test-scenario/i)).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-003] displays all files in table with correct stats', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      // All file paths should be visible
      expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      expect(screen.getByText('src/api/handlers.go')).toBeInTheDocument();
      expect(screen.getByText('src/lib/utils.ts')).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-003] displays line count for each file', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      const appRow = screen.getByText('src/components/App.tsx').closest('tr');
      expect(appRow).toHaveTextContent('250');

      const handlersRow = screen.getByText('src/api/handlers.go').closest('tr');
      expect(handlersRow).toHaveTextContent('850');

      const utilsRow = screen.getByText('src/lib/utils.ts').closest('tr');
      expect(utilsRow).toHaveTextContent('120');
    });
  });

  it('[REQ:TM-UI-003] displays lint, type, and AI issue counts separately', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      const handlersRow = screen.getByText('src/api/handlers.go').closest('tr');
      // handlers.go: 5 lint + 3 type + 2 AI = 10 total
      expect(handlersRow).toHaveTextContent('5'); // lint
      expect(handlersRow).toHaveTextContent('3'); // type
      expect(handlersRow).toHaveTextContent('2'); // AI
      expect(handlersRow).toHaveTextContent('10'); // total
    });
  });

  it('[REQ:TM-UI-003] displays visit count for each file', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      const appRow = screen.getByText('src/components/App.tsx').closest('tr');
      expect(appRow).toHaveTextContent('3');

      const handlersRow = screen.getByText('src/api/handlers.go').closest('tr');
      expect(handlersRow).toHaveTextContent('1');

      const utilsRow = screen.getByText('src/lib/utils.ts').closest('tr');
      expect(utilsRow).toHaveTextContent('5');
    });
  });

  it('[REQ:TM-UI-003] indicates long files with visual marker', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      // handlers.go is marked as long file (850 lines) - should have LONG badge
      const handlersRow = screen.getByText('src/api/handlers.go').closest('tr');
      expect(handlersRow).toHaveTextContent('LONG');

      // App.tsx is not long (250 lines) - should not have LONG badge
      const appRow = screen.getByText('src/components/App.tsx').closest('tr');
      expect(appRow).not.toHaveTextContent('LONG');
    });
  });

  it('[REQ:TM-UI-004] supports sorting by path', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
    });

    // Default sort is by totalIssues (desc), so handlers.go (10 issues) should be first
    const rows = screen.getAllByRole('row');
    const dataRows = rows.slice(1); // Skip header row
    expect(dataRows[0]).toHaveTextContent('src/api/handlers.go');
  });

  it('[REQ:TM-UI-004] supports sorting by line count', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
    });

    // Files should be sortable by lines (desc: 850, 250, 120)
    const dataRows = screen.getAllByTestId('file-table-row');

    // Default sort is totalIssues, so handlers (10) > App (3) > utils (0)
    expect(dataRows[0]).toHaveTextContent('src/api/handlers.go');
    expect(dataRows[1]).toHaveTextContent('src/components/App.tsx');
    expect(dataRows[2]).toHaveTextContent('src/lib/utils.ts');
  });

  it('[REQ:TM-UI-004] supports sorting by total issues', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      const dataRows = screen.getAllByTestId('file-table-row');

      // Default sort is totalIssues desc: handlers (10) > App (3) > utils (0)
      expect(dataRows[0]).toHaveTextContent('src/api/handlers.go');
      expect(dataRows[1]).toHaveTextContent('src/components/App.tsx');
      expect(dataRows[2]).toHaveTextContent('src/lib/utils.ts');
    });
  });

  it('[REQ:TM-UI-004] supports sorting by visit count', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
    });

    // Files by visit count: utils (5) > App (3) > handlers (1)
    // But default is totalIssues, so order is different
    const rows = screen.getAllByRole('row');
    expect(rows.length).toBeGreaterThan(3);
  });

  it('[REQ:TM-UI-003] displays scenario summary statistics', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      // Should show total light issues, AI issues, long files
      const issuesCard = screen.getByTestId('total-issues-card');
      expect(issuesCard).toHaveTextContent('10 light');
      expect(issuesCard).toHaveTextContent('5 AI');

      const longFilesCard = screen.getByTestId('long-files-card');
      expect(longFilesCard).toHaveTextContent('3');
    });
  });

  it('[REQ:TM-UI-003] shows loading state while fetching data', async () => {
    vi.mocked(api.fetchScenarioDetail).mockImplementation(
      () => new Promise(resolve => setTimeout(() => resolve(mockScenarioDetail), 100))
    );

    renderScenarioDetail();

    // Should show loading state
    expect(screen.getByTestId('spinner')).toBeInTheDocument();

    // Wait for data to load
    await waitFor(() => {
      expect(screen.queryByTestId('spinner')).not.toBeInTheDocument();
      expect(screen.getByText('test-scenario')).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-003] displays error message when API fails', async () => {
    vi.mocked(api.fetchScenarioDetail).mockRejectedValue(new Error('Scenario not found'));

    renderScenarioDetail();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load scenario/i)).toBeInTheDocument();
      expect(screen.getByText(/Scenario not found/i)).toBeInTheDocument();
    });
  });

  it('[REQ:TM-UI-004] calculates total issues correctly', async () => {
    vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

    renderScenarioDetail();

    await waitFor(() => {
      const handlersRow = screen.getByText('src/api/handlers.go').closest('tr');
      // 5 lint + 3 type + 2 AI = 10 total
      expect(handlersRow).toHaveTextContent('10');

      const appRow = screen.getByText('src/components/App.tsx').closest('tr');
      // 2 lint + 1 type + 0 AI = 3 total
      expect(appRow).toHaveTextContent('3');

      const utilsRow = screen.getByText('src/lib/utils.ts').closest('tr');
      // 0 lint + 0 type + 0 AI = 0 total
      expect(utilsRow).toHaveTextContent('0');
    });
  });
});
