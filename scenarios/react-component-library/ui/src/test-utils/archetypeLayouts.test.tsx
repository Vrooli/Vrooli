import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { AssetDetailShell } from "@vrooli/react-component-library/AssetDetailShell/1";
import { InspectorLayout } from "@vrooli/react-component-library/InspectorLayout/1";

describe("archetype lifecycle layouts", () => {
  it("keeps asset preview and metadata usable when supporting activity is partial", () => {
    renderWithProviders(
      <AssetDetailShell
        title="Release artifact"
        preview={<p>Artifact preview</p>}
        metadata={<p>Version 1.2.0</p>}
        activityState="partial"
        activitySurfaceId="asset-activity"
      />,
    );

    expect(screen.getByRole("region", { name: "Release artifact preview" })).toHaveTextContent(
      "Artifact preview",
    );
    expect(
      screen.getByRole("complementary", { name: "Release artifact metadata" }),
    ).toHaveTextContent("Version 1.2.0");
    expect(screen.getByRole("status")).toHaveAccessibleName("Some information is unavailable.");
    expect(document.querySelector('[data-experience-surface="asset-activity"]')).toHaveAttribute(
      "data-experience-state",
      "partial",
    );
  });

  it("keeps the primary canvas available when the inspector is empty", () => {
    renderWithProviders(
      <InspectorLayout
        title="Workflow editor"
        canvas={<p>Workflow canvas</p>}
        inspectorState="empty"
        inspector={<p>Nothing selected</p>}
      />,
    );

    expect(screen.getByRole("region", { name: "Workflow editor canvas" })).toHaveTextContent(
      "Workflow canvas",
    );
    expect(screen.getByText("Nothing to show yet.")).toBeVisible();
    expect(document.querySelector('[data-experience-surface="inspector"]')).toHaveAttribute(
      "data-experience-state",
      "empty",
    );
  });
});
