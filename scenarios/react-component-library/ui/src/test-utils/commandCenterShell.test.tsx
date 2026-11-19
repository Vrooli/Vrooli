import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { CommandCenterShell } from "@vrooli/react-component-library/CommandCenterShell/1";

describe("CommandCenterShell", () => {
  it("keeps operational context visible while its primary region reports a partial lifecycle", () => {
    // The authored usage contract now requires `metrics[].id`, while the
    // published 1.0.5 type still declares label/value/detail only. The runtime
    // warning is the mismatch; assert it and keep the published type valid.
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
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
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("stable non-empty identifier"));
  });

  it("renders optional controls, metric details, and primary content", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
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
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("stable non-empty identifier"));
  });
});
