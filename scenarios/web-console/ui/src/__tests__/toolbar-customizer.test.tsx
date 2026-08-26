import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, fireEvent, within } from "@testing-library/react";
import ToolbarCustomizer from "../components/settings/ToolbarCustomizer";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { i18n } from "../i18n";
import { layoutToolbar, toolbarPrefsFromPreset } from "../lib/toolbarLayout";

function prefs() {
  return useWorkspaceStore.getState().toolbarPrefs;
}

function openPanel() {
  render(<ToolbarCustomizer />);
  fireEvent.click(screen.getByTestId("toolbar-customize-toggle"));
}

describe("ToolbarCustomizer", () => {
  beforeEach(async () => {
    // The readout assertions below read interpolated numbers, so the catalog
    // has to be loaded rather than falling back to raw keys.
    await i18n.changeLanguage("en");
    useWorkspaceStore.setState({ toolbarPrefs: toolbarPrefsFromPreset("balanced") });
  });

  it("keeps the customization panel closed until it is asked for", () => {
    render(<ToolbarCustomizer />);
    expect(screen.queryByTestId("toolbar-customizer-panel")).toBeNull();
    fireEvent.click(screen.getByTestId("toolbar-customize-toggle"));
    expect(screen.getByTestId("toolbar-customizer-panel")).toBeInTheDocument();
  });

  it("applies a preset wholesale", () => {
    render(<ToolbarCustomizer />);
    fireEvent.click(screen.getByTestId("toolbar-preset-dense"));
    expect(prefs()).toEqual(toolbarPrefsFromPreset("dense"));
    expect(screen.queryByTestId("toolbar-preset-custom")).toBeNull();
  });

  it("moves to custom when a single control is changed, and says so", () => {
    openPanel();
    fireEvent.click(screen.getByTestId("toolbar-density-compact"));
    expect(prefs().density).toBe("compact");
    expect(prefs().preset).toBe("custom");
    expect(screen.getByTestId("toolbar-preset-custom")).toBeInTheDocument();
  });

  it("warns about small targets without preventing them", () => {
    openPanel();
    // Only the largest size clears the 44px recommendation; the default
    // standard size is already below it and says so.
    fireEvent.click(screen.getByTestId("toolbar-density-large"));
    expect(screen.queryByTestId("toolbar-density-warning")).toBeNull();

    fireEvent.click(screen.getByTestId("toolbar-density-compact"));
    // Warn, never block: the preference still took effect.
    expect(screen.getByTestId("toolbar-density-warning")).toBeInTheDocument();
    expect(prefs().density).toBe("compact");
  });

  it("hides a control on request and pins the one that holds the hidden ones", () => {
    openPanel();
    const image = screen.getByTestId("toolbar-control-image");
    expect(image).not.toBeDisabled();
    fireEvent.click(image);
    expect(prefs().enabled.image).toBe(false);

    // More cannot be switched off: it is where the hidden controls live.
    const more = screen.getByTestId("toolbar-control-more");
    expect(more).toBeDisabled();
    expect(more).toBeChecked();
  });

  it("switches the arrows off through the same control that styles them", () => {
    openPanel();
    fireEvent.click(screen.getByTestId("toolbar-arrows-off"));
    expect(prefs().enabled.arrows).toBe(false);
    fireEvent.click(screen.getByTestId("toolbar-arrows-inline"));
    expect(prefs().enabled.arrows).toBe(true);
    expect(prefs().arrows).toBe("inline");
  });

  it("reports the row count and height the engine actually produced", () => {
    openPanel();
    const readout = () => screen.getByTestId("toolbar-preview-readout").textContent ?? "";
    fireEvent.click(screen.getByTestId("toolbar-preview-width-430"));

    const expected = layoutToolbar(prefs(), 430);
    expect(readout()).toContain(String(expected.rowCount));
    expect(readout()).toContain(String(expected.keysHeightPx));

    fireEvent.click(screen.getByTestId("toolbar-rows-1"));
    const afterBudget = layoutToolbar(prefs(), 430);
    expect(afterBudget.rowCount).toBe(1);
    expect(readout()).toContain(String(afterBudget.keysHeightPx));
  });

  it("renders the preview from the same engine as the toolbar, and keeps it out of the way", () => {
    openPanel();
    fireEvent.click(screen.getByTestId("toolbar-preview-width-430"));

    const surface = screen.getByTestId("toolbar-preview-surface");
    // Decorative by construction: no focus order, no assistive-tech presence.
    expect(surface).toHaveAttribute("aria-hidden", "true");
    expect(surface).toHaveAttribute("inert");

    const expected = layoutToolbar(prefs(), 430);
    for (let row = 0; row < expected.rowCount; row += 1) {
      expect(within(surface).getByTestId(`toolbar-row-${String(row)}`)).toBeInTheDocument();
    }
    expect(within(surface).queryByTestId(`toolbar-row-${String(expected.rowCount)}`)).toBeNull();
  });
});
