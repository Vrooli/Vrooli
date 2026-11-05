import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../../test-utils/renderWithProviders";
import { ActionableBadge, StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders with data-testid when executionStatus is provided", () => {
    renderWithCanonicalProviders(<StatusBadge executionStatus="running" />);
    expect(screen.getByTestId("status-badge")).toBeInTheDocument();
  });

  it("returns null when executionStatus is undefined", () => {
    const { container } = renderWithCanonicalProviders(<StatusBadge executionStatus={undefined} />);
    expect(container.firstChild).toBeNull();
  });

  it("shows human-readable title", () => {
    renderWithCanonicalProviders(<StatusBadge executionStatus="needs_review" />);
    expect(screen.getByTestId("status-badge")).toHaveAttribute("title", "Execution: needs review");
  });
});

describe("ActionableBadge", () => {
  it("renders with data-testid", () => {
    renderWithCanonicalProviders(<ActionableBadge status="backlog" />);
    expect(screen.getByTestId("actionable-badge")).toBeInTheDocument();
  });

  it("shows human-readable title", () => {
    renderWithCanonicalProviders(<ActionableBadge status="in_progress" />);
    expect(screen.getByTestId("actionable-badge")).toHaveAttribute("title", "Actionable: in progress");
  });

  it.each(["backlog", "researching", "ready", "queued", "in_progress", "failed"])(
    "renders for actionable status %s",
    (status) => {
      const { container } = renderWithCanonicalProviders(<ActionableBadge status={status} />);
      expect(container.firstChild).not.toBeNull();
    },
  );
});
