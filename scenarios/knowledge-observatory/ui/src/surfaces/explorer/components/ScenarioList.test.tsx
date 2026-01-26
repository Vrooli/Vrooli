import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ScenarioList, type ScenarioListProps } from "./ScenarioList";

const createProps = (overrides: Partial<ScenarioListProps> = {}): ScenarioListProps => ({
  scenarios: [
    {
      name: "alpha",
      path: "scenarios/alpha",
      docCountLabel: "5 docs",
      healthScoreLabel: "92%",
      healthTone: "good",
      hasManifest: true,
      hasReadme: true,
      lastModifiedLabel: "1/26/2026",
    },
  ],
  filter: "",
  onFilterChange: vi.fn(),
  selectedScenario: "alpha",
  onSelectScenario: vi.fn(),
  isLoading: false,
  hasError: false,
  errorMessage: "",
  onRefresh: vi.fn(),
  ...overrides,
});

describe("ScenarioList", () => {
  it("renders scenarios with health badge", () => {
    render(<ScenarioList {...createProps()} />);

    expect(screen.getByText("alpha")).toBeDefined();
    expect(screen.getByText("5 docs")).toBeDefined();
    expect(screen.getByText("92%")).toBeDefined();
  });

  it("renders empty state when no scenarios", () => {
    render(<ScenarioList {...createProps({ scenarios: [] })} />);

    expect(screen.getByText(/No scenarios found/i)).toBeDefined();
  });
});
