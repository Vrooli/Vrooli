import { describe, it, expect, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { DeepSearchPanel, type DeepSearchPanelProps } from "./DeepSearchPanel";
import { selectors } from "../../../consts/selectors";

const createProps = (overrides: Partial<DeepSearchPanelProps> = {}): DeepSearchPanelProps => ({
  query: "",
  scope: "global",
  scenario: "",
  basePath: "",
  maxResults: 10,
  followRefs: true,
  timeoutSeconds: undefined,
  isSubmitting: false,
  isRunning: false,
  statusLabel: "idle",
  progressLabel: "",
  errorMessage: "",
  hasResults: false,
  results: [],
  onQueryChange: vi.fn(),
  onScopeChange: vi.fn(),
  onScenarioChange: vi.fn(),
  onBasePathChange: vi.fn(),
  onMaxResultsChange: vi.fn(),
  onFollowRefsChange: vi.fn(),
  onTimeoutSecondsChange: vi.fn(),
  onSubmit: vi.fn(),
  onClear: vi.fn(),
  ...overrides,
});

describe("DeepSearchPanel", () => {
  it("renders empty state fields", () => {
    render(<DeepSearchPanel {...createProps()} />);
    expect(screen.getByText(/Deep documentation search/i)).toBeDefined();
    expect(screen.getByLabelText(/Scope/i)).toBeDefined();
  });

  it("renders results when provided", () => {
    render(
      <DeepSearchPanel
        {...createProps({
          hasResults: true,
          results: [
            {
              path: "docs/README.md",
              relevance: 0.88,
              summary: "Readme overview",
              match_reason: "Overview section",
              references: ["docs/guide.md"],
              snippet: "## Overview",
            },
          ],
        })}
      />
    );

    expect(screen.getByText(/Readme overview/i)).toBeDefined();
    expect(screen.getByText(/Relevance: 0.88/i)).toBeDefined();
    const results = screen.getByTestId(selectors.deepSearch.results);
    expect(within(results).getByText(/References/i)).toBeDefined();
  });

  it("shows error message when present", () => {
    render(<DeepSearchPanel {...createProps({ errorMessage: "Failed" })} />);
    expect(screen.getByText(/Deep Search Error/i)).toBeDefined();
  });
});
