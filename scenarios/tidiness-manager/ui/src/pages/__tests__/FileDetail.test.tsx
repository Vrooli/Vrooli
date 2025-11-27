// [REQ:TM-UI-003] Test file detail view
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import FileDetail from "../FileDetail";
import * as api from "../../lib/api";

// Mock the API module
vi.mock("../../lib/api", () => ({
  fetchFileIssues: vi.fn(),
}));

const mockFetchFileIssues = vi.mocked(api.fetchFileIssues);

function renderFileDetail(scenarioName: string, filePath: string) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/scenario/${scenarioName}/file/${filePath}`]}>
        <Routes>
          <Route path="/scenario/:scenarioName/file/*" element={<FileDetail />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("FileDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // [REQ:TM-UI-003] Test loading state
  it("displays loading spinner while fetching file issues", () => {
    mockFetchFileIssues.mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    renderFileDetail("test-scenario", "src/main.go");

    // Check for spinner with status role
    const spinner = screen.getByRole("status");
    expect(spinner).toBeInTheDocument();
    expect(spinner).toHaveAttribute("data-testid", "spinner");
  });

  // [REQ:TM-UI-003] Test error state
  it("displays error message when file issues fail to load", async () => {
    const errorMessage = "Failed to fetch file issues";
    mockFetchFileIssues.mockRejectedValue(new Error(errorMessage));

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
      expect(screen.getByText("Failed to load file issues")).toBeInTheDocument();
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });

    // Verify troubleshooting tips are shown
    expect(screen.getByText(/Verify the file path is correct/)).toBeInTheDocument();
    expect(screen.getByText(/Check if the scenario exists/)).toBeInTheDocument();
  });

  // [REQ:TM-UI-003] Test successful file detail display with issues
  it("displays file detail with issue statistics", async () => {
    const mockIssues = [
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 10,
        column: 5,
        severity: "error" as const,
        category: "lint",
        message: "Undefined variable 'x'",
        rule: "no-undef",
        tool: "eslint",
        status: "open" as const,
      },
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 20,
        column: 8,
        severity: "warning" as const,
        category: "style",
        message: "Missing semicolon",
        rule: "semi",
        tool: "eslint",
        status: "open" as const,
      },
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 30,
        column: 1,
        severity: "error" as const,
        category: "type",
        message: "Type mismatch",
        rule: null,
        tool: "typescript",
        status: "resolved" as const,
      },
    ];

    mockFetchFileIssues.mockResolvedValue(mockIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByText("src/main.go")).toBeInTheDocument();
    });

    // Verify summary cards
    expect(screen.getByTestId("total-issues-card")).toBeInTheDocument();
    expect(screen.getByTestId("open-issues-card")).toBeInTheDocument();
    expect(screen.getByTestId("resolved-issues-card")).toBeInTheDocument();

    // Verify totals
    expect(screen.getByLabelText("3 total issues")).toBeInTheDocument();
    expect(screen.getByLabelText("2 open issues")).toBeInTheDocument();
    expect(screen.getByLabelText("1 resolved issues")).toBeInTheDocument();

    // Verify issues are displayed
    expect(screen.getByText("Undefined variable 'x'")).toBeInTheDocument();
    expect(screen.getByText("Missing semicolon")).toBeInTheDocument();
    expect(screen.getByText("Type mismatch")).toBeInTheDocument();

    // Verify issue metadata
    expect(screen.getByText("Line 10:5")).toBeInTheDocument();
    expect(screen.getByText("Line 20:8")).toBeInTheDocument();
    expect(screen.getByText("Line 30:1")).toBeInTheDocument();
    expect(screen.getByText("Rule: no-undef")).toBeInTheDocument();
    expect(screen.getByText("Rule: semi")).toBeInTheDocument();
  });

  // [REQ:TM-UI-003] Test empty state (no issues)
  it("displays empty state when no issues found", async () => {
    mockFetchFileIssues.mockResolvedValue([]);

    renderFileDetail("clean-scenario", "src/perfect.go");

    await waitFor(() => {
      expect(screen.getByRole("status")).toBeInTheDocument();
      expect(screen.getByText("No issues found")).toBeInTheDocument();
    });

    expect(screen.getByText(/This file passed all linting, type checking, and AI analysis/)).toBeInTheDocument();
    expect(screen.getByText(/Run a new scan to check for changes/)).toBeInTheDocument();

    // Verify zero counts in cards
    expect(screen.getByLabelText("0 total issues")).toBeInTheDocument();
    expect(screen.getByLabelText("0 open issues")).toBeInTheDocument();
    expect(screen.getByLabelText("0 resolved issues")).toBeInTheDocument();
  });

  // [REQ:TM-UI-003] Test back navigation button
  it("displays back button to scenario detail", async () => {
    mockFetchFileIssues.mockResolvedValue([]);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByRole("status")).toBeInTheDocument();
    });

    const backButton = screen.getByLabelText("Go back to previous page");
    expect(backButton).toBeInTheDocument();
  });

  // [REQ:TM-UI-003] Test severity badge rendering
  it("renders correct severity badges for issues", async () => {
    const mockIssues = [
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 1,
        column: 1,
        severity: "error" as const,
        category: "lint",
        message: "Critical error",
        rule: null,
        tool: "eslint",
        status: "open" as const,
      },
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 2,
        column: 1,
        severity: "warning" as const,
        category: "style",
        message: "Minor warning",
        rule: null,
        tool: "eslint",
        status: "open" as const,
      },
    ];

    mockFetchFileIssues.mockResolvedValue(mockIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      const errorBadges = screen.getAllByText("error");
      const warningBadges = screen.getAllByText("warning");
      expect(errorBadges.length).toBeGreaterThan(0);
      expect(warningBadges.length).toBeGreaterThan(0);
    });
  });

  // [REQ:TM-UI-003] Test all issues are open
  it("handles file with all open issues", async () => {
    const mockIssues = [
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 1,
        column: 1,
        severity: "error" as const,
        category: "lint",
        message: "Issue 1",
        rule: null,
        tool: "eslint",
        status: "open" as const,
      },
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 2,
        column: 1,
        severity: "error" as const,
        category: "lint",
        message: "Issue 2",
        rule: null,
        tool: "eslint",
        status: "open" as const,
      },
    ];

    mockFetchFileIssues.mockResolvedValue(mockIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByLabelText("2 total issues")).toBeInTheDocument();
      expect(screen.getByLabelText("2 open issues")).toBeInTheDocument();
      expect(screen.getByLabelText("0 resolved issues")).toBeInTheDocument();
    });
  });

  // [REQ:TM-UI-003] Test all issues are resolved
  it("handles file with all resolved issues", async () => {
    const mockIssues = [
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 1,
        column: 1,
        severity: "error" as const,
        category: "lint",
        message: "Fixed issue 1",
        rule: null,
        tool: "eslint",
        status: "resolved" as const,
      },
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 2,
        column: 1,
        severity: "error" as const,
        category: "lint",
        message: "Fixed issue 2",
        rule: null,
        tool: "eslint",
        status: "resolved" as const,
      },
    ];

    mockFetchFileIssues.mockResolvedValue(mockIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByLabelText("2 total issues")).toBeInTheDocument();
      expect(screen.getByLabelText("0 open issues")).toBeInTheDocument();
      expect(screen.getByLabelText("2 resolved issues")).toBeInTheDocument();
    });
  });

  // [REQ:TM-UI-003] Test action buttons presence
  it("displays mark resolved and ignore buttons for each issue", async () => {
    const mockIssues = [
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 10,
        column: 5,
        severity: "error" as const,
        category: "lint",
        message: "Test issue",
        rule: null,
        tool: "eslint",
        status: "open" as const,
      },
    ];

    mockFetchFileIssues.mockResolvedValue(mockIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByLabelText("Mark issue at line 10:5 as resolved")).toBeInTheDocument();
      expect(screen.getByLabelText("Ignore issue at line 10:5")).toBeInTheDocument();
    });
  });

  // [REQ:TM-UI-003] Test CLI command hint display
  it("displays CLI command hint for file issues", async () => {
    mockFetchFileIssues.mockResolvedValue([]);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByText(/CLI: tidiness-manager issues test-scenario --file="src\/main.go"/)).toBeInTheDocument();
    });
  });

  // [REQ:TM-UI-003] Test file with very long path
  it("handles file with very long path", async () => {
    const longPath = "src/very/deeply/nested/directory/structure/that/goes/on/and/on/file.go";
    mockFetchFileIssues.mockResolvedValue([]);

    renderFileDetail("test-scenario", longPath);

    await waitFor(() => {
      expect(screen.getByText(longPath)).toBeInTheDocument();
    });
  });

  // [REQ:TM-UI-003] Test issue with null rule
  it("handles issue without rule gracefully", async () => {
    const mockIssues = [
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 1,
        column: 1,
        severity: "error" as const,
        category: "custom",
        message: "Custom issue without rule",
        rule: null,
        tool: "custom-tool",
        status: "open" as const,
      },
    ];

    mockFetchFileIssues.mockResolvedValue(mockIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByText("Custom issue without rule")).toBeInTheDocument();
      expect(screen.getByText("Tool: custom-tool")).toBeInTheDocument();
    });

    // Should not display "Rule:" when rule is null
    expect(screen.queryByText(/^Rule:/)).not.toBeInTheDocument();
  });

  // [REQ:TM-UI-003] Test large number of issues
  it("renders many issues efficiently", async () => {
    const manyIssues = Array.from({ length: 50 }, (_, i) => ({
      scenario: "test-scenario",
      file_path: "src/main.go",
      line: i + 1,
      column: 1,
      severity: i % 2 === 0 ? ("error" as const) : ("warning" as const),
      category: "lint",
      message: `Issue ${i + 1}`,
      rule: `rule-${i}`,
      tool: "eslint",
      status: "open" as const,
    }));

    mockFetchFileIssues.mockResolvedValue(manyIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      expect(screen.getByLabelText("50 total issues")).toBeInTheDocument();
    });

    // Verify first and last issues are rendered
    expect(screen.getByText("Issue 1")).toBeInTheDocument();
    expect(screen.getByText("Issue 50")).toBeInTheDocument();
  });

  // [REQ:TM-UI-003] Test accessibility attributes
  it("includes proper ARIA labels and roles", async () => {
    const mockIssues = [
      {
        scenario: "test-scenario",
        file_path: "src/main.go",
        line: 10,
        column: 5,
        severity: "error" as const,
        category: "lint",
        message: "Accessibility test issue",
        rule: null,
        tool: "eslint",
        status: "open" as const,
      },
    ];

    mockFetchFileIssues.mockResolvedValue(mockIssues);

    renderFileDetail("test-scenario", "src/main.go");

    await waitFor(() => {
      const issue = screen.getByRole("article");
      expect(issue).toBeInTheDocument();
      expect(issue).toHaveAttribute("aria-labelledby");
      expect(issue).toHaveAttribute("aria-describedby");
    });

    // Verify severity icon has label
    expect(screen.getByLabelText("error severity")).toBeInTheDocument();
  });
});
