import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { CommandCenterShell } from "@vrooli/react-component-library/CommandCenterShell/1";

// Fixture copy the shell is asked to display, named once so the queries match
// the fixture instead of restating its copy.
const metricDetail = "Across all environments";
const regionContent = "Recent activity";

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
        metrics={[{ label: "Active runs", value: "12", detail: metricDetail }]}
        regionState="ready"
      >
        <p>{regionContent}</p>
      </CommandCenterShell>,
    );

    expect(screen.getByRole("button", { name: "Refresh" })).toBeVisible();
    expect(screen.getByText(metricDetail)).toBeVisible();
    expect(screen.getByText(regionContent)).toBeVisible();
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("stable non-empty identifier"));
  });
});
