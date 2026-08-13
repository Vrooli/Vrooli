import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { CommandCenterShell } from "../components/CommandCenterShell/versions/1.0.0/CommandCenterShell";

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
});
