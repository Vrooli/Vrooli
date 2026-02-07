import { describe, it, expect, vi } from "vitest";
import type { ReactElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { HealthPanel, type HealthPanelProps } from "./HealthPanel";

const renderWithQueryClient = (ui: ReactElement) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
};

const createProps = (overrides: Partial<HealthPanelProps> = {}): HealthPanelProps => ({
  scenarioName: "alpha",
  healthViewModel: {
    healthScoreLabel: "95%",
    healthTone: "good",
    totalDocsLabel: "10 docs",
    missingDocs: [],
    extraDocs: [],
    misplacedDocs: [],
    warningCount: 0,
    hasIssues: false,
    canAutoFix: false,
    fixCategory: "none",
  },
  isLoading: false,
  hasError: false,
  errorMessage: "",
  onRefresh: vi.fn(),
  ...overrides,
});

describe("HealthPanel", () => {
  it("renders no-issues state", () => {
    renderWithQueryClient(<HealthPanel {...createProps()} />);

    expect(screen.getByText(/No documentation issues detected/i)).toBeDefined();
    expect(screen.getByText(/10 docs/i)).toBeDefined();
  });

  it("renders missing docs list", () => {
    renderWithQueryClient(
      <HealthPanel
        {...createProps({
          healthViewModel: {
            healthScoreLabel: "75%",
            healthTone: "medium",
            totalDocsLabel: "4 docs",
            missingDocs: ["readme"],
            extraDocs: [],
            misplacedDocs: [],
            warningCount: 1,
            hasIssues: true,
            canAutoFix: false,
            fixCategory: "none",
          },
        })}
      />
    );

    expect(screen.getByText(/Missing Docs/i)).toBeDefined();
    expect(screen.getByText("readme")).toBeDefined();
  });
});
