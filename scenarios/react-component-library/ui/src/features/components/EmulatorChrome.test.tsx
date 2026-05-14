import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import {
  DEVICE_EMULATION_STORAGE_KEY,
  useDeviceEmulation,
} from "../../hooks/useDeviceEmulation";
import { useDeviceFilters } from "../../hooks/useDeviceFilters";
import { EmulatorChrome } from "./EmulatorChrome";

function Harness() {
  const emulator = useDeviceEmulation();
  return (
    <EmulatorChrome emulator={emulator}>
      <div data-testid="emulator-child">child</div>
    </EmulatorChrome>
  );
}

function FiltersHarness() {
  const emulator = useDeviceEmulation();
  const filters = useDeviceFilters();
  return (
    <EmulatorChrome emulator={emulator} filters={filters}>
      <div data-testid="emulator-child">child</div>
    </EmulatorChrome>
  );
}

describe("EmulatorChrome", () => {
  beforeEach(async () => {
    window.localStorage.clear();
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
  });

  it("renders the toolbar, dimensions, and child", () => {
    renderWithProviders(<Harness />);
    expect(screen.getByTestId(selectors.components.emulator.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.components.emulator.toolbar)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.components.emulator.dimensions).textContent).toContain(
      "1280",
    );
    expect(screen.getByTestId("emulator-child")).toBeInTheDocument();
  });

  it("switching preset updates dimensions display", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    const select = screen.getByTestId(selectors.components.emulator.presetSelect) as HTMLSelectElement;
    await user.selectOptions(select, "iphone-se");
    expect(screen.getByTestId(selectors.components.emulator.dimensions).textContent).toContain(
      "375",
    );
    expect(screen.getByTestId(selectors.components.emulator.dimensions).textContent).toContain(
      "667",
    );
  });

  it("zoom in/out updates the displayed percent and persists", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    expect(screen.getByTestId(selectors.components.emulator.zoomValue).textContent).toBe("100%");
    await user.click(screen.getByTestId(selectors.components.emulator.zoomOut));
    expect(screen.getByTestId(selectors.components.emulator.zoomValue).textContent).toBe("90%");
    const persisted = window.localStorage.getItem(DEVICE_EMULATION_STORAGE_KEY);
    expect(persisted).toContain("0.9");
  });

  it("rotate flips display dimensions", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await user.selectOptions(
      screen.getByTestId(selectors.components.emulator.presetSelect),
      "iphone-14",
    );
    expect(screen.getByTestId(selectors.components.emulator.dimensions).textContent).toContain(
      "390",
    );
    await user.click(screen.getByTestId(selectors.components.emulator.rotate));
    const dims = screen.getByTestId(selectors.components.emulator.dimensions).textContent ?? "";
    expect(dims).toContain("844");
    expect(dims.indexOf("844")).toBeLessThan(dims.indexOf("390"));
  });

  it("reset returns to defaults", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await user.selectOptions(
      screen.getByTestId(selectors.components.emulator.presetSelect),
      "ipad-pro",
    );
    await user.click(screen.getByTestId(selectors.components.emulator.zoomIn));
    await user.click(screen.getByTestId(selectors.components.emulator.rotate));
    await user.click(screen.getByTestId(selectors.components.emulator.reset));
    expect(
      (screen.getByTestId(selectors.components.emulator.presetSelect) as HTMLSelectElement).value,
    ).toBe("desktop-1280");
    expect(screen.getByTestId(selectors.components.emulator.zoomValue).textContent).toBe("100%");
  });

  it("filter controls absent when no filters prop is supplied", () => {
    renderWithProviders(<Harness />);
    expect(
      screen.queryByTestId(selectors.components.emulator.visionFilterSelect),
    ).toBeNull();
    expect(
      screen.queryByTestId(selectors.components.emulator.filterDefs),
    ).toBeNull();
  });

  it("changing vision filter updates the viewport CSS filter chain", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FiltersHarness />);
    const viewport = screen.getByTestId(selectors.components.emulator.viewport);
    expect(viewport.style.filter || "").toBe("");
    const select = screen.getByTestId(
      selectors.components.emulator.visionFilterSelect,
    ) as HTMLSelectElement;
    await user.selectOptions(select, "protanopia");
    expect(viewport.style.filter).toContain("url(#rcl-vision-protanopia)");
  });

  it("blur slider composes blur(Npx) into the filter chain", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FiltersHarness />);
    await user.selectOptions(
      screen.getByTestId(selectors.components.emulator.visionFilterSelect),
      "deuteranopia",
    );
    const slider = screen.getByTestId(
      selectors.components.emulator.blurSlider,
    ) as HTMLInputElement;
    fireEvent.change(slider, { target: { value: "4" } });
    const viewport = screen.getByTestId(selectors.components.emulator.viewport);
    expect(viewport.style.filter).toContain("url(#rcl-vision-deuteranopia)");
    expect(viewport.style.filter).toContain("blur(4px)");
    expect(
      screen.getByTestId(selectors.components.emulator.blurValue).textContent,
    ).toContain("4px");
  });

  it("color-scheme select reflects the chosen option", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FiltersHarness />);
    const select = screen.getByTestId(
      selectors.components.emulator.colorSchemeSelect,
    ) as HTMLSelectElement;
    expect(select.value).toBe("system");
    await user.selectOptions(select, "dark");
    expect(select.value).toBe("dark");
  });

  it("SVG filter defs are emitted when filters are wired", () => {
    renderWithProviders(<FiltersHarness />);
    const defs = screen.getByTestId(selectors.components.emulator.filterDefs);
    expect(defs).toBeInTheDocument();
    expect(defs.querySelector("filter#rcl-vision-grayscale")).not.toBeNull();
    expect(defs.querySelector("filter#rcl-vision-protanopia")).not.toBeNull();
    expect(defs.querySelector("filter#rcl-vision-deuteranopia")).not.toBeNull();
    expect(defs.querySelector("filter#rcl-vision-tritanopia")).not.toBeNull();
  });

  it("applies CSS transform scale on the viewport", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await user.click(screen.getByTestId(selectors.components.emulator.zoomOut));
    const viewport = screen.getByTestId(selectors.components.emulator.viewport);
    expect(viewport.style.transform).toContain("scale(0.9)");
  });
});
