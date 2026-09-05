import { render, screen } from "@testing-library/react";
import { BarChart3 } from "lucide-react";
import { describe, expect, it } from "vitest";
import { KeyValueList } from "./key-value-list";
import { MiniBarChart } from "./mini-bar-chart";
import { ProgressBar } from "./progress-bar";
import { SectionLabel } from "./section-label";
import { StatsCard } from "./stats-card";

describe("stats primitives", () => {
  it("renders the shared metric card shell with an icon", () => {
    render(<StatsCard label="Completed" value="12" icon={BarChart3} testId="shared-stat" />);

    expect(screen.getByTestId("shared-stat")).toHaveTextContent("Completed");
    expect(screen.getByTestId("shared-stat")).toHaveTextContent("12");
  });

  it("renders progress and list primitives", () => {
    render(
      <>
        <ProgressBar value={2} max={4} />
        <SectionLabel icon={BarChart3}>Breakdown</SectionLabel>
        <KeyValueList entries={[["meta_orchestration", 3]]} formatKey={(value) => value.replace("_", " ")} />
      </>,
    );

    expect(screen.getByText("Breakdown")).toBeInTheDocument();
    expect(screen.getByText("meta orchestration")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("renders an SVG bar chart with point titles", () => {
    render(
      <MiniBarChart
        points={[
          { key: "2026-03-17", label: "2026-03-17", value: 3 },
          { key: "2026-03-24", label: "2026-03-24", value: 5 },
        ]}
        testId="mini-chart"
      />,
    );

    expect(screen.getByTestId("mini-chart")).toHaveAttribute("role", "img");
    expect(screen.getByText("2026-03-24: 5 completed")).toBeInTheDocument();
  });
});
