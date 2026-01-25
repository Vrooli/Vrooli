import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MetricsPanel, type MetricsPanelProps } from "./MetricsPanel";
import type { MetricsViewModel } from "../controllers/knowledgeController";

const baseViewModel: MetricsViewModel = {
  metricCards: [
    {
      label: "Coherence",
      description: "Topical consistency",
      percentageLabel: "70.0%",
      tone: "good",
    },
  ],
  collections: [
    {
      name: "alpha",
      sizeLabel: "12 vectors",
      metrics: [{ label: "Coherence", percentageLabel: "70%" }],
    },
  ],
  overallHealth: "steady",
  lastUpdated: "10:00 AM",
  totalEntriesLabel: "42",
  hasMetrics: true,
};

const createProps = (overrides: Partial<MetricsPanelProps> = {}): MetricsPanelProps => ({
  isLoading: false,
  hasError: false,
  errorMessage: "",
  hasData: true,
  viewModel: baseViewModel,
  onRetry: vi.fn(),
  ...overrides,
});

describe("MetricsPanel", () => {
  it("renders a loading state", () => {
    render(<MetricsPanel {...createProps({ isLoading: true })} />);

    expect(screen.getByText(/Loading metrics/i)).toBeDefined();
  });

  it("renders metrics and collections when data is available", () => {
    render(<MetricsPanel {...createProps()} />);

    expect(screen.getByText(/Overall Health/i)).toBeDefined();
    expect(screen.getByText(/Coherence/i)).toBeDefined();
    expect(screen.getByText(/alpha/i)).toBeDefined();
    expect(screen.getByText(/12 vectors/i)).toBeDefined();
  });

  it("falls back safely when the view model is missing", () => {
    const unsafeProps = {
      ...createProps(),
      viewModel: undefined as unknown as MetricsViewModel,
    };

    expect(() => render(<MetricsPanel {...unsafeProps} />)).not.toThrow();
    expect(screen.getByText(/unknown condition/i)).toBeDefined();
  });
});
