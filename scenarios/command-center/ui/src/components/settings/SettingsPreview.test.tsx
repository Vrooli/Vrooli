import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import type { CatalogEntry, Catalogs } from "../../lib/api";
import { SettingsPreview } from "./SettingsPreview";

const catalogs: Catalogs = {
  signals: [{ id: "active_scenarios", label: "Apps running", shape: "scalar", coverage: "NOW", sample: { value: 12, series: [], basis: "authored" } }] as CatalogEntry[],
  compositions: [{ id: "orbital-field", slots: { running: { shape: "scalar" } } }] as CatalogEntry[],
  themes: [{ id: "cosmos", tokens: { "--color-primary": "#8aa8ff" } }] as CatalogEntry[],
};

describe("SettingsPreview", () => {
  it("renders the stage and captions it with the room composition and theme", () => {
    const room: CatalogEntry = { id: "mission-control", composition: "orbital-field", beats: [{ hero: "active_scenarios" }], bind: { running: "active_scenarios" } };
    const { getByTestId } = renderWithProviders(<SettingsPreview catalogs={catalogs} room={room} theme={catalogs.themes?.[0]} />);
    const stage = getByTestId("settings-preview-stage");
    expect(stage).toHaveAttribute("data-theme", "cosmos");
    expect(stage.style.getPropertyValue("--color-primary")).toBe("#8aa8ff");
    expect(stage.textContent).toContain("orbital-field");
    expect(stage.textContent).toContain("cosmos");
    expect(getByTestId("settings-preview-viewport")).toHaveStyle({ width: "1440px", height: "900px" });
    expect(getByTestId("cycle-rail")).toBeInTheDocument();
    expect(getByTestId("room-composition")).toHaveAttribute("data-room", "mission-control");
    expect(getByTestId("room-hero").textContent).toContain("Apps running");
  });

  it("falls back to the first composition when the room names none", () => {
    const { getByTestId } = renderWithProviders(<SettingsPreview catalogs={catalogs} room={undefined} theme={undefined} />);
    expect(getByTestId("settings-preview-stage").textContent).toContain("orbital-field");
  });
});
