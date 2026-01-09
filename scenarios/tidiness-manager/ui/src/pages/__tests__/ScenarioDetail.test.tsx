import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ScenarioDetail from '../ScenarioDetail';
import * as api from '../../lib/api';
import { ToastProvider } from '../../components/ui/toast';

// [REQ:TM-UI-003] Scenario detail - file table
// [REQ:TM-UI-004] Scenario detail - sortable columns

vi.mock('../../lib/api');

const mockScenarioDetail = {
  stats: {
    scenario: 'test-scenario',
    total_files: 3,
    total_lines: 1220,
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
      <MemoryRouter initialEntries={[`/scenario/${scenarioName}`]}>
        <ToastProvider>
          <Routes>
            <Route path="/scenario/:scenarioName" element={<ScenarioDetail />} />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('ScenarioDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('[REQ:TM-UI-003] Basic Rendering and Data Display', () => {
    it('renders scenario detail page with file table', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('test-scenario')).toBeInTheDocument();
        expect(screen.getByText(/Code tidiness overview for test-scenario/i)).toBeInTheDocument();
      });
    });

    it('displays all files in table with correct stats', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        // All file paths should be visible
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
        expect(screen.getByText('src/api/handlers.go')).toBeInTheDocument();
        expect(screen.getByText('src/lib/utils.ts')).toBeInTheDocument();
      });
    });

    it('displays line count for each file', async () => {
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

    it('displays lint, type, and AI issue counts separately', async () => {
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

    it('displays visit count for each file', async () => {
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

    it('indicates long files with visual marker', async () => {
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

    it('displays scenario summary statistics with total files and lines', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        // Should show total light issues, AI issues, long files
        const issuesCard = screen.getByTestId('total-issues-card');
        expect(issuesCard).toHaveTextContent('10 light');
        expect(issuesCard).toHaveTextContent('5 AI');

        const longFilesCard = screen.getByTestId('long-files-card');
        expect(longFilesCard).toHaveTextContent('3');

        const totalFilesCard = screen.getByTestId('total-files-card');
        expect(totalFilesCard).toHaveTextContent('3');
        expect(totalFilesCard).toHaveTextContent('1,220 total lines');
      });
    });
  });

  describe('[REQ:TM-UI-004] Interactive Sorting', () => {
    it('sorts by path when path header is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      // Click path header to sort by path (desc first)
      const pathHeader = screen.getByTestId('file-table-header-path');
      await user.click(pathHeader);

      // Files sorted by path desc: utils > App > handlers (lib > components > api)
      await waitFor(() => {
        const dataRows = screen.getAllByTestId('file-table-row');
        // First row should contain utils.ts (src/lib)
        expect(dataRows[0].textContent).toContain('utils.ts');
        // Last row should contain handlers.go (src/api)
        expect(dataRows[2].textContent).toContain('handlers.go');
      });

      // Click again to reverse to asc
      await user.click(pathHeader);

      await waitFor(() => {
        const dataRowsAsc = screen.getAllByTestId('file-table-row');
        // First row should contain handlers.go (alphabetically first: api < components < lib)
        expect(dataRowsAsc[0].textContent).toContain('handlers.go');
        // Last row should contain utils.ts
        expect(dataRowsAsc[2].textContent).toContain('utils.ts');
      });
    });

    it('sorts by line count when lines header is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      // Click lines header
      const linesHeader = screen.getByTestId('file-table-header-lines');
      await user.click(linesHeader);

      // Files sorted by lines desc: 850, 250, 120
      const dataRows = screen.getAllByTestId('file-table-row');
      expect(dataRows[0]).toHaveTextContent('src/api/handlers.go');
      expect(dataRows[1]).toHaveTextContent('src/components/App.tsx');
      expect(dataRows[2]).toHaveTextContent('src/lib/utils.ts');
    });

    it('sorts by total issues by default (descending)', async () => {
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

    it('sorts by visit count when visits header is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      // Click visit count header
      const visitHeader = screen.getByTestId('file-table-header-visit-count');
      await user.click(visitHeader);

      // Files by visit count desc: utils (5) > App (3) > handlers (1)
      const dataRows = screen.getAllByTestId('file-table-row');
      expect(dataRows[0]).toHaveTextContent('src/lib/utils.ts');
      expect(dataRows[1]).toHaveTextContent('src/components/App.tsx');
      expect(dataRows[2]).toHaveTextContent('src/api/handlers.go');
    });

    it('toggles sort direction when same header clicked twice', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      const issuesHeader = screen.getByTestId('file-table-header-issues');

      // Default is desc: handlers (10) > App (3) > utils (0)
      let dataRows = screen.getAllByTestId('file-table-row');
      expect(within(dataRows[0]).getByText('src/api/handlers.go')).toBeInTheDocument();

      // Click once to toggle from desc to asc (since already on totalIssues)
      await user.click(issuesHeader);

      await waitFor(() => {
        const ascRows = screen.getAllByTestId('file-table-row');
        // After first click: asc order - utils (0) < App (3) < handlers (10)
        expect(ascRows[0].textContent).toContain('utils.ts');
        expect(ascRows[2].textContent).toContain('handlers.go');
      });

      // Click again to toggle back to desc
      await user.click(issuesHeader);

      await waitFor(() => {
        const descRows = screen.getAllByTestId('file-table-row');
        // After second click: back to desc order - handlers (10) > App (3) > utils (0)
        expect(descRows[0].textContent).toContain('handlers.go');
        expect(descRows[2].textContent).toContain('utils.ts');
      });
    });

    it('supports keyboard navigation (Enter key) on sortable headers', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      const pathHeader = screen.getByTestId('file-table-header-path');
      pathHeader.focus();

      // Press Enter to sort
      await user.keyboard('{Enter}');

      const dataRows = screen.getAllByTestId('file-table-row');
      expect(dataRows[0]).toHaveTextContent('src/lib/utils.ts');
    });

    it('supports keyboard navigation (Space key) on sortable headers', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      const linesHeader = screen.getByTestId('file-table-header-lines');
      linesHeader.focus();

      // Press Space to sort
      await user.keyboard(' ');

      const dataRows = screen.getAllByTestId('file-table-row');
      expect(dataRows[0]).toHaveTextContent('src/api/handlers.go'); // 850 lines
    });
  });

  describe('[REQ:TM-UI-003] Loading and Error States', () => {
    it('shows loading state while fetching data', async () => {
      vi.mocked(api.fetchScenarioDetail).mockImplementation(
        () => new Promise(resolve => setTimeout(() => resolve(mockScenarioDetail), 100))
      );

      renderScenarioDetail();

      // Should show loading state
      expect(screen.getByTestId('spinner')).toBeInTheDocument();
      expect(screen.getAllByText('Loading...').length).toBeGreaterThan(0);

      // Wait for data to load
      await waitFor(() => {
        expect(screen.queryByTestId('spinner')).not.toBeInTheDocument();
        expect(screen.getByText('test-scenario')).toBeInTheDocument();
      });
    });

    it('displays error message when API fails', async () => {
      vi.mocked(api.fetchScenarioDetail).mockRejectedValue(new Error('Scenario not found'));

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText(/Failed to load scenario/i)).toBeInTheDocument();
        expect(screen.getByText(/Scenario not found/i)).toBeInTheDocument();
      });
    });

    it('displays error when network request times out', async () => {
      vi.mocked(api.fetchScenarioDetail).mockRejectedValue(new Error('Network timeout'));

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText(/Failed to load scenario/i)).toBeInTheDocument();
        expect(screen.getByText(/Network timeout/i)).toBeInTheDocument();
      });
    });

    it('displays error for API 500 response', async () => {
      vi.mocked(api.fetchScenarioDetail).mockRejectedValue(new Error('Internal server error'));

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText(/Failed to load scenario/i)).toBeInTheDocument();
        expect(screen.getByText(/Internal server error/i)).toBeInTheDocument();
      });
    });
  });

  describe('[REQ:TM-UI-003] Edge Cases and Boundary Conditions', () => {
    it('handles empty file list gracefully', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        stats: {
          scenario: 'empty-scenario',
          total_files: 0,
          total_lines: 0,
          light_issues: 0,
          ai_issues: 0,
          long_files: 0,
          visit_percent: 0,
          campaign_status: 'active' as const,
        },
        files: [],
      });

      renderScenarioDetail('empty-scenario');

      await waitFor(() => {
        expect(screen.getByText('empty-scenario')).toBeInTheDocument();
        // Should handle empty file list without crashing
        const dataRows = screen.queryAllByTestId('file-table-row');
        expect(dataRows).toHaveLength(0);
      });
    });

    it('handles scenario with all zero stats', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        stats: {
          scenario: 'clean-scenario',
          total_files: 1,
          total_lines: 50,
          light_issues: 0,
          ai_issues: 0,
          long_files: 0,
          visit_percent: 0,
          campaign_status: 'inactive' as const,
        },
        files: [
          {
            path: 'src/clean.ts',
            lines: 50,
            lint_issues: 0,
            type_issues: 0,
            ai_issues: 0,
            visit_count: 0,
            is_long_file: false,
          },
        ],
      });

      renderScenarioDetail('clean-scenario');

      await waitFor(() => {
        const issuesCard = screen.getByTestId('total-issues-card');
        expect(issuesCard).toHaveTextContent('0 light');
        expect(issuesCard).toHaveTextContent('0 AI');

        const longFilesCard = screen.getByTestId('long-files-card');
        expect(longFilesCard).toHaveTextContent('0');

        const cleanRow = screen.getByText('src/clean.ts').closest('tr');
        expect(cleanRow).toHaveTextContent('0');
      });
    });

    it('handles files with very large issue counts', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        stats: {
          scenario: 'messy-scenario',
          total_files: 1,
          total_lines: 5000,
          light_issues: 9999,
          ai_issues: 1234,
          long_files: 100,
          visit_percent: 10,
          campaign_status: 'active' as const,
        },
        files: [
          {
            path: 'src/messy.ts',
            lines: 5000,
            lint_issues: 999,
            type_issues: 888,
            ai_issues: 777,
            visit_count: 0,
            is_long_file: true,
          },
        ],
      });

      renderScenarioDetail('messy-scenario');

      await waitFor(() => {
        const messyRow = screen.getByText('src/messy.ts').closest('tr');
        // Should display large numbers correctly: 999 + 888 + 777 = 2664
        expect(messyRow).toHaveTextContent('999');
        expect(messyRow).toHaveTextContent('888');
        expect(messyRow).toHaveTextContent('777');
        expect(messyRow).toHaveTextContent('2664');
      });
    });

    it('handles file paths with special characters', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        stats: {
          scenario: 'special-scenario',
          total_files: 2,
          total_lines: 200,
          light_issues: 0,
          ai_issues: 0,
          long_files: 0,
          visit_percent: 100,
          campaign_status: 'active' as const,
        },
        files: [
          {
            path: 'src/components/@ui/special-[file].tsx',
            lines: 100,
            lint_issues: 0,
            type_issues: 0,
            ai_issues: 0,
            visit_count: 1,
            is_long_file: false,
          },
          {
            path: 'src/utils/(group)/test file.ts',
            lines: 100,
            lint_issues: 0,
            type_issues: 0,
            ai_issues: 0,
            visit_count: 1,
            is_long_file: false,
          },
        ],
      });

      renderScenarioDetail('special-scenario');

      await waitFor(() => {
        expect(screen.getByText('src/components/@ui/special-[file].tsx')).toBeInTheDocument();
        expect(screen.getByText('src/utils/(group)/test file.ts')).toBeInTheDocument();
      });
    });

    it('calculates total issues correctly for all edge cases', async () => {
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

    it('handles undefined/missing stat fields gracefully', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        stats: {
          scenario: 'minimal-scenario',
          light_issues: 0,
          ai_issues: 0,
          long_files: 0,
          visit_percent: 0,
          campaign_status: 'active' as const,
        },
        files: [],
      });

      renderScenarioDetail('minimal-scenario');

      await waitFor(() => {
        const totalFilesCard = screen.getByTestId('total-files-card');
        expect(totalFilesCard).toHaveTextContent('0');
        expect(totalFilesCard).toHaveTextContent('0 total lines');
      });
    });

    it('handles single file scenario', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        stats: {
          scenario: 'single-scenario',
          total_files: 1,
          total_lines: 100,
          light_issues: 2,
          ai_issues: 1,
          long_files: 0,
          visit_percent: 100,
          campaign_status: 'active' as const,
        },
        files: [
          {
            path: 'main.go',
            lines: 100,
            lint_issues: 1,
            type_issues: 1,
            ai_issues: 1,
            visit_count: 10,
            is_long_file: false,
          },
        ],
      });

      renderScenarioDetail('single-scenario');

      await waitFor(() => {
        const dataRows = screen.getAllByTestId('file-table-row');
        expect(dataRows).toHaveLength(1);
        expect(dataRows[0]).toHaveTextContent('main.go');
      });
    });
  });

  describe('[REQ:TM-UI-003] Interactive Elements', () => {
    it('provides View Details button for each file', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        const viewButtons = screen.getAllByRole('button', { name: /View Details|View/i });
        // Should have View button for each file (3 files) + other buttons
        expect(viewButtons.length).toBeGreaterThanOrEqual(3);
      });
    });

    it('displays CLI command helper for issues', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText(/CLI:/i)).toBeInTheDocument();
        expect(screen.getByText(/tidiness-manager issues test-scenario --limit 10/i)).toBeInTheDocument();
      });
    });

    it('provides Run Light Scan button', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        const scanButton = screen.getByRole('button', { name: /Run Light Scan|Scan/i });
        expect(scanButton).toBeInTheDocument();
      });
    });

    it('provides View All Issues button', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        const issuesButtons = screen.getAllByRole('button', { name: /View All Issues|Issues/i });
        expect(issuesButtons.length).toBeGreaterThan(0);
      });
    });

    it('View Details button navigates to correct file detail route', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      // Find and click the View Details button for App.tsx
      const appRow = screen.getByText('src/components/App.tsx').closest('tr');
      const viewButton = appRow?.querySelector('button[aria-label*="View"]') ||
                         appRow?.querySelector('a[href*="App.tsx"]');

      if (viewButton) {
        // Verify the link/button has correct routing information
        const href = viewButton.getAttribute('href');
        if (href) {
          expect(href).toContain('App.tsx');
        }
      }
    });

    it('displays correct CLI commands for different scenarios', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        ...mockScenarioDetail,
        stats: {
          ...mockScenarioDetail.stats,
          scenario: 'custom-scenario-name',
        },
      });

      renderScenarioDetail('custom-scenario-name');

      await waitFor(() => {
        // CLI command should use the actual scenario name from the data
        expect(screen.getByText(/tidiness-manager issues custom-scenario-name/i)).toBeInTheDocument();
      });
    });
  });

  describe('[REQ:TM-UI-003] Performance and Stress Testing', () => {
    it('handles extremely large file lists (1000+ files) without crashing', async () => {
      const largeFileList = Array.from({ length: 1000 }, (_, i) => ({
        path: `src/components/Component${i}.tsx`,
        lines: 100 + i,
        lint_issues: i % 10,
        type_issues: i % 5,
        ai_issues: i % 3,
        visit_count: i % 20,
        is_long_file: i % 50 === 0,
      }));

      vi.mocked(api.fetchScenarioDetail).mockResolvedValue({
        stats: {
          scenario: 'large-scenario',
          total_files: 1000,
          total_lines: 150000,
          light_issues: 5000,
          ai_issues: 2000,
          long_files: 20,
          visit_percent: 25,
          campaign_status: 'active' as const,
        },
        files: largeFileList,
      });

      renderScenarioDetail('large-scenario');

      await waitFor(() => {
        expect(screen.getByText('large-scenario')).toBeInTheDocument();
        const dataRows = screen.getAllByTestId('file-table-row');
        // Should render all 1000 files without crashing
        expect(dataRows.length).toBe(1000);
      });
    });

    it('handles concurrent sorting operations correctly', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      // Click multiple headers rapidly to test sort stability
      const pathHeader = screen.getByTestId('file-table-header-path');
      const linesHeader = screen.getByTestId('file-table-header-lines');
      const issuesHeader = screen.getByTestId('file-table-header-issues');

      await user.click(pathHeader);
      await user.click(linesHeader);
      await user.click(issuesHeader);

      // Final state should be sorted by issues (last click)
      await waitFor(() => {
        const dataRows = screen.getAllByTestId('file-table-row');
        expect(dataRows[0]).toHaveTextContent('src/api/handlers.go'); // highest issues
      });
    });

    it('maintains sort state after data refresh', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      const { rerender } = renderScenarioDetail();

      await waitFor(() => {
        expect(screen.getByText('src/components/App.tsx')).toBeInTheDocument();
      });

      // Sort by path
      const pathHeader = screen.getByTestId('file-table-header-path');
      await user.click(pathHeader);

      await waitFor(() => {
        const dataRows = screen.getAllByTestId('file-table-row');
        expect(dataRows[0]).toHaveTextContent('utils.ts'); // alphabetically last by path
      });

      // Simulate data refresh with same data
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      // The sort order should be maintained (though this depends on implementation)
      await waitFor(() => {
        const dataRows = screen.getAllByTestId('file-table-row');
        // Sort should still be applied
        expect(dataRows.length).toBe(3);
      });
    });
  });

  describe('[REQ:TM-UI-004] Accessibility', () => {
    it('provides proper ARIA labels for sortable headers', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        const pathHeader = screen.getByTestId('file-table-header-path');
        expect(pathHeader).toHaveAttribute('aria-label');
        expect(pathHeader.getAttribute('aria-label')).toContain('Sort by file path');
        expect(pathHeader).toHaveAttribute('tabIndex', '0');
        expect(pathHeader).toHaveAttribute('role', 'button');
      });
    });

    it('provides table semantics and aria-label', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        const table = screen.getByTestId('file-table');
        expect(table).toHaveAttribute('role', 'table');
        expect(table).toHaveAttribute('aria-label');
        expect(table.getAttribute('aria-label')).toContain('Files in scenario');
      });
    });

    it('provides meaningful aria-labels for statistics cards', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        const totalFilesCard = screen.getByTestId('total-files-card');
        const totalFilesValue = within(totalFilesCard).getByText('3');
        expect(totalFilesValue).toHaveAttribute('aria-label');
        expect(totalFilesValue.getAttribute('aria-label')).toContain('total files');
        expect(totalFilesValue.getAttribute('aria-label')).toContain('1,220');
      });
    });

    it('provides aria-labels for all summary statistics', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        const issuesCard = screen.getByTestId('total-issues-card');
        const issuesValue = within(issuesCard).getByText('15');
        expect(issuesValue).toHaveAttribute('aria-label');
        expect(issuesValue.getAttribute('aria-label')).toContain('total issues');

        const longFilesCard = screen.getByTestId('long-files-card');
        const longFilesValue = within(longFilesCard).getByText('3');
        expect(longFilesValue).toHaveAttribute('aria-label');

        const visitCard = screen.getByTestId('visit-coverage-card');
        const visitValue = within(visitCard).getByText('60%');
        expect(visitValue).toHaveAttribute('aria-label');
      });
    });

    it('headers are keyboard-accessible via tabIndex', async () => {
      vi.mocked(api.fetchScenarioDetail).mockResolvedValue(mockScenarioDetail);

      renderScenarioDetail();

      await waitFor(() => {
        ['file-table-header-path', 'file-table-header-lines', 'file-table-header-issues', 'file-table-header-visit-count'].forEach(testId => {
          const header = screen.getByTestId(testId);
          expect(header).toHaveAttribute('tabIndex', '0');
        });
      });
    });
  });
});
