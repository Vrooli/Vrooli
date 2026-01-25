import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SearchPanel, type SearchPanelProps } from "./SearchPanel";

const createProps = (overrides: Partial<SearchPanelProps> = {}): SearchPanelProps => ({
  query: "",
  onQueryChange: vi.fn(),
  onSubmit: vi.fn(),
  onClear: vi.fn(),
  onSampleClick: vi.fn(),
  sampleQueries: ["sample one", "sample two"],
  isLoading: false,
  hasError: false,
  errorMessage: "",
  hasData: false,
  hasResults: false,
  isSubmitDisabled: true,
  isClearDisabled: true,
  displayQuery: "your query",
  totalResults: 0,
  tookMsLabel: "?ms",
  results: [],
  ...overrides,
});

describe("SearchPanel", () => {
  it("renders the idle empty state", () => {
    render(<SearchPanel {...createProps()} />);

    expect(screen.getByText(/Enter a query to search the knowledge base/i)).toBeDefined();
    expect(screen.getByText(/Try:/i)).toBeDefined();
  });

  it("renders results and metadata when available", () => {
    render(
      <SearchPanel
        {...createProps({
          hasData: true,
          hasResults: true,
          totalResults: 1,
          tookMsLabel: "12ms",
          results: [
            {
              id: "r-1",
              content: "Result content",
              metadata: { source: "alpha" },
              scoreLabel: "90.0%",
              hasMetadata: true,
            },
          ],
        })}
      />
    );

    expect(screen.getByText(/Found 1 results/i)).toBeDefined();
    expect(screen.getByText(/Score: 90.0%/i)).toBeDefined();
    expect(screen.getByText(/Metadata/i)).toBeDefined();
  });
});
