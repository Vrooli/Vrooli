import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { IsolationBadge } from "./IsolationBadge";
import type { ScenarioIsolation } from "../hooks/useScenarioIsolation";

function makeIsolation(overrides: Partial<ScenarioIsolation>): ScenarioIsolation {
  return {
    status: "loading",
    reasons: [],
    violations: [],
    refetch: () => {},
    ...overrides,
  };
}

describe("IsolationBadge", () => {
  it("renders the routed (green) headline when isolation is confirmed", () => {
    render(<IsolationBadge isolation={makeIsolation({ status: "routed" })} />);
    expect(screen.getByText(/Data isolation confirmed/i)).toBeInTheDocument();
  });

  it("renders the amber 'not routed' headline and shows violation details on expand", () => {
    const isolation = makeIsolation({
      status: "not_routed",
      reasons: ["raw sql.Open"],
      violations: [{ rule_id: "routed_database_drivers", severity: "high", file: "db.go", line: 42 }],
    });
    render(<IsolationBadge isolation={isolation} />);
    expect(screen.getByText(/Data isolation not available/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /show details/i }));
    expect(screen.getByText(/raw sql.Open/)).toBeInTheDocument();
    expect(screen.getByText(/routed_database_drivers/)).toBeInTheDocument();
    expect(screen.getByText(/db.go:42/)).toBeInTheDocument();
  });

  it("renders the unknown (grey) state when test-genie is unreachable", () => {
    render(<IsolationBadge isolation={makeIsolation({ status: "unknown" })} />);
    expect(screen.getByText(/Isolation status unavailable/i)).toBeInTheDocument();
  });

  it("renders the loading skeleton headline while the check is in-flight", () => {
    render(<IsolationBadge isolation={makeIsolation({ status: "loading" })} />);
    expect(screen.getByText(/Checking isolation status/i)).toBeInTheDocument();
  });

  it("does not render a details toggle when there are no reasons or violations", () => {
    render(<IsolationBadge isolation={makeIsolation({ status: "not_routed" })} />);
    expect(screen.queryByRole("button", { name: /show details/i })).not.toBeInTheDocument();
  });
});
