import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ScenarioBadge } from "./scenario-badge";

function renderBadge(acceptanceAllow?: string[]) {
  return render(
    <MemoryRouter>
      <ScenarioBadge acceptanceAllow={acceptanceAllow} />
    </MemoryRouter>,
  );
}

describe("ScenarioBadge", () => {
  it("returns null when acceptanceAllow is undefined", () => {
    const { container } = renderBadge(undefined);
    expect(container.innerHTML).toBe("");
  });

  it("returns null when acceptanceAllow is empty", () => {
    const { container } = renderBadge([]);
    expect(container.innerHTML).toBe("");
  });

  it("returns null when globs have no scenario prefix", () => {
    const { container } = renderBadge(["packages/shared/**"]);
    expect(container.innerHTML).toBe("");
  });

  it("renders scenario name for single scenario", () => {
    renderBadge(["scenarios/web-console/api/**"]);
    expect(screen.getByText("web-console")).toBeInTheDocument();
  });

  it("shows +N suffix for multiple scenarios", () => {
    renderBadge([
      "scenarios/web-console/**",
      "scenarios/shared-ui/**",
      "scenarios/auth/**",
    ]);
    expect(screen.getByText("web-console +2")).toBeInTheDocument();
  });

  it("truncates long scenario names", () => {
    renderBadge(["scenarios/very-long-scenario-name-here/api/**"]);
    expect(screen.getByText("very-long-scenario\u2026")).toBeInTheDocument();
  });

  it("links to scenario detail page", () => {
    renderBadge(["scenarios/web-console/api/**"]);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/scenarios/web-console");
  });

  it("shows tooltip with all scenario names", () => {
    renderBadge(["scenarios/web-console/**", "scenarios/shared-ui/**"]);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("title", "Scenarios: web-console, shared-ui");
  });

  it("shows singular tooltip for single scenario", () => {
    renderBadge(["scenarios/web-console/api/**"]);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("title", "Scenario: web-console");
  });
});
