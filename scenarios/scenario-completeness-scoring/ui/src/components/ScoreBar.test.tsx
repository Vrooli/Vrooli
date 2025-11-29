import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ScoreBar } from "./ScoreBar";

// [REQ:SCS-UI-005] Unit tests for ScoreBar component

describe("ScoreBar", () => {
  it("renders label and value", () => {
    render(<ScoreBar label="Quality" value={40} max={50} />);

    expect(screen.getByText("Quality")).toBeInTheDocument();
    expect(screen.getByText("40.0/50 (80%)")).toBeInTheDocument();
  });

  it("calculates percentage correctly", () => {
    render(<ScoreBar label="Coverage" value={8} max={15} />);

    // 8/15 = 53.33%
    expect(screen.getByText("8.0/15 (53%)")).toBeInTheDocument();
  });

  it("handles zero max value", () => {
    render(<ScoreBar label="Test" value={0} max={0} />);

    expect(screen.getByText("0.0/0 (0%)")).toBeInTheDocument();
  });

  it("renders with data-testid", () => {
    render(<ScoreBar label="UI Score" value={20} max={25} />);

    expect(screen.getByTestId("score-bar-ui-score")).toBeInTheDocument();
  });

  it("caps percentage at 100", () => {
    render(<ScoreBar label="Overflow" value={60} max={50} />);

    // Value exceeds max, percentage should cap at 100%
    expect(screen.getByText("60.0/50 (100%)")).toBeInTheDocument();
  });
});
