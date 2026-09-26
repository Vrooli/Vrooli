import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { CommandCenterShell } from "@vrooli/react-component-library/CommandCenterShell/1";

describe("CommandCenterShell", () => {
  it("keeps operational context visible while its primary region reports a partial lifecycle", () => {
    renderWithProviders(
      <CommandCenterShell
        title="Operations"
        navigation={<a href="#runs">Runs</a>}
        metrics={[{ label: "Active runs", value: "12" }]}
        regionId="operations-runs"
        regionState="partial"
      />,
    );

    expect(screen.getByRole("navigation", { name: "Operations navigation" })).toHaveTextContent(
      "Runs",
    );
    expect(screen.getByText("12")).toBeVisible();
    expect(screen.getByRole("status")).toHaveAccessibleName("Some information is unavailable.");
    expect(document.querySelector('[data-experience-surface="operations-runs"]')).toHaveAttribute(
      "data-experience-state",
      "partial",
    );
  });

  it("renders optional controls, metric details, and primary content", () => {
    renderWithProviders(
      <CommandCenterShell
        title="Operations"
        navigation={<a href="#runs">Runs</a>}
        controls={<button type="button">Refresh</button>}
        metrics={[{ label: "Active runs", value: "12", detail: "Across all environments" }]}
        regionState="ready"
      >
        <p>Recent activity</p>
      </CommandCenterShell>,
    );

    expect(screen.getByRole("button", { name: "Refresh" })).toBeVisible();
    expect(screen.getByText("Across all environments")).toBeVisible();
    expect(screen.getByText("Recent activity")).toBeVisible();
  });
});
