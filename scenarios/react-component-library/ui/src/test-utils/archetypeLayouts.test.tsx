import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AssetDetailShell } from "../components/AssetDetailShell/versions/1.0.0/AssetDetailShell";
import { InspectorLayout } from "../components/InspectorLayout/versions/1.0.0/InspectorLayout";

describe("archetype lifecycle layouts", () => {
  it("keeps asset preview and metadata usable when supporting activity is partial", () => {
    render(
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
    render(
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
