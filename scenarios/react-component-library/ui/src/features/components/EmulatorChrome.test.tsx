import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { DEVICE_EMULATION_STORAGE_KEY, useDeviceEmulation } from "../../hooks/useDeviceEmulation";
import { useDeviceFilters } from "../../hooks/useDeviceFilters";
import { EmulatorChrome } from "./EmulatorChrome";

function Harness() {
  const emulator = useDeviceEmulation();
  return <EmulatorChrome emulator={emulator}><div data-testid="emulator-child">child</div></EmulatorChrome>;
}

function FiltersHarness() {
  const emulator = useDeviceEmulation();
  const filters = useDeviceFilters();
  return <EmulatorChrome emulator={emulator} filters={filters}><div data-testid="emulator-child">child</div></EmulatorChrome>;
}

async function openViewport(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId(selectors.components.emulator.viewportToggle));
  return screen.getByTestId(selectors.components.emulator.viewportPanel);
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

  it("uses a compact Viewport trigger and keeps dimensions hidden until Responsive", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    const trigger = screen.getByTestId(selectors.components.emulator.viewportToggle);
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId(selectors.components.emulator.dimensions)).toBeNull();
    await openViewport(user);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
  });

  it("allows responsive dimensions only after explicitly selecting Responsive", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await openViewport(user);
    const responsive = screen.getAllByTestId(selectors.components.emulator.presetOption)
      .find((button) => button.getAttribute("data-preset") === "responsive");
    expect(responsive).toBeDefined();
    await user.click(responsive!);
    const inputs = screen.getByTestId(selectors.components.emulator.dimensions).querySelectorAll("input");
    fireEvent.change(inputs[0]!, { target: { value: "369" } });
    fireEvent.change(inputs[1]!, { target: { value: "652" } });
    expect(inputs[0]?.value).toBe("369");
    expect(inputs[1]?.value).toBe("652");
  });

  it("persists viewport selections and applies zoom as a visual transform", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await openViewport(user);
    const zoom = screen.getByTestId<HTMLSelectElement>(selectors.components.emulator.zoomValue);
    await user.selectOptions(zoom, "0.9");
    expect(window.localStorage.getItem(DEVICE_EMULATION_STORAGE_KEY)).toContain("0.9");
    expect(screen.getByTestId(selectors.components.emulator.viewport).style.transform).toContain("scale(0.9)");
  });

  it("returns focus to Viewport after Escape", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    const trigger = screen.getByTestId(selectors.components.emulator.viewportToggle);
    await openViewport(user);
    await user.keyboard("{Escape}");
    expect(screen.queryByTestId(selectors.components.emulator.viewportPanel)).toBeNull();
    expect(trigger).toHaveFocus();
  });

  it("keeps visual filter application independent of the viewport menu", () => {
    renderWithProviders(<FiltersHarness />);
    expect(screen.getByTestId(selectors.components.emulator.filterDefs)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.components.emulator.visionFilterSelect)).toBeNull();
    expect(screen.queryByTestId(selectors.components.emulator.blurSlider)).toBeNull();
  });
});
