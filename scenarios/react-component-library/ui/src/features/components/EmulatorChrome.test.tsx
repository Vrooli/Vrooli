import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import {
  DEVICE_EMULATION_STORAGE_KEY,
  useDeviceEmulation,
} from "../../hooks/useDeviceEmulation";
import { useDeviceFilters } from "../../hooks/useDeviceFilters";
import { EmulatorChrome, TOOLBAR_INLINE_MIN_WIDTH } from "./EmulatorChrome";

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
    const inputs = screen
      .getByTestId(selectors.components.emulator.dimensions)
      .querySelectorAll("input");
    expect(inputs[0]?.value).toBe("1280");
    expect(inputs[1]?.value).toBe("800");
    expect(screen.getByTestId("emulator-child")).toBeInTheDocument();
  });

  it("switching preset updates dimensions display", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    const select = screen.getByTestId<HTMLSelectElement>(selectors.components.emulator.presetSelect);
    await user.selectOptions(select, "iphone-se");
    const inputs = screen
      .getByTestId(selectors.components.emulator.dimensions)
      .querySelectorAll("input");
    expect(inputs[0]?.value).toBe("375");
    expect(inputs[1]?.value).toBe("667");
  });

  it("zoom in/out updates the displayed percent and persists", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    const zoomSelect = screen.getByTestId<HTMLSelectElement>(
      selectors.components.emulator.zoomValue,
    );
    expect(zoomSelect.value).toBe("1");
    await user.selectOptions(zoomSelect, "0.9");
    expect(zoomSelect.value).toBe("0.9");
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
    const inputs = screen
      .getByTestId(selectors.components.emulator.dimensions)
      .querySelectorAll("input");
    expect(inputs[0]?.value).toBe("390");
    await user.click(screen.getByTestId(selectors.components.emulator.rotate));
    expect(inputs[0]?.value).toBe("844");
    expect(inputs[1]?.value).toBe("390");
  });

  it("responsive dimensions can be edited directly", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await user.selectOptions(
      screen.getByTestId(selectors.components.emulator.presetSelect),
      "responsive",
    );
    const inputs = screen
      .getByTestId(selectors.components.emulator.dimensions)
      .querySelectorAll("input");
    fireEvent.change(inputs[0]!, { target: { value: "369" } });
    fireEvent.change(inputs[1]!, { target: { value: "652" } });
    expect(inputs[0]?.value).toBe("369");
    expect(inputs[1]?.value).toBe("652");
  });

  it("reset returns to defaults", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await user.selectOptions(
      screen.getByTestId(selectors.components.emulator.presetSelect),
      "ipad-pro",
    );
    await user.selectOptions(
      screen.getByTestId<HTMLSelectElement>(selectors.components.emulator.zoomValue),
      "1.25",
    );
    await user.click(screen.getByTestId(selectors.components.emulator.rotate));
    await user.click(screen.getByTestId(selectors.components.emulator.reset));
    expect(
      screen.getByTestId<HTMLSelectElement>(selectors.components.emulator.presetSelect).value,
    ).toBe("desktop-1280");
    expect(
      screen.getByTestId<HTMLSelectElement>(selectors.components.emulator.zoomValue).value,
    ).toBe("1");
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
    const select = screen.getByTestId<HTMLSelectElement>(
      selectors.components.emulator.visionFilterSelect,
    );
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
    const slider = screen.getByTestId<HTMLInputElement>(
      selectors.components.emulator.blurSlider,
    );
    fireEvent.change(slider, { target: { value: "4" } });
    const viewport = screen.getByTestId(selectors.components.emulator.viewport);
    expect(viewport.style.filter).toContain("url(#rcl-vision-deuteranopia)");
    expect(viewport.style.filter).toContain("blur(4px)");
    expect(
      screen.getByTestId(selectors.components.emulator.blurValue).textContent,
    ).toContain("4px");
  });

  it("does not own a color-scheme control (moved to the ThemeSwitcher)", () => {
    renderWithProviders(<FiltersHarness />);
    expect(
      screen.queryByTestId("components-emulator-color-scheme"),
    ).toBeNull();
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
    await user.selectOptions(
      screen.getByTestId<HTMLSelectElement>(selectors.components.emulator.zoomValue),
      "0.9",
    );
    const viewport = screen.getByTestId(selectors.components.emulator.viewport);
    expect(viewport.style.transform).toContain("scale(0.9)");
  });

  describe("responsive priority collapse", () => {
    const globalRef = globalThis as { ResizeObserver?: typeof ResizeObserver };
    let restore: (() => void) | null = null;

    // Drive the ResizeObserver-based collapse deterministically: install a
    // ResizeObserver that reports the width under test synchronously on
    // observe(), so the hook's measured width is exactly `width`.
    function withContainerWidth(width: number) {
      const original = globalRef.ResizeObserver;
      class MockResizeObserver {
        private readonly cb: ResizeObserverCallback;
        constructor(cb: ResizeObserverCallback) {
          this.cb = cb;
        }
        observe(): void {
          this.cb([{ contentRect: { width } } as unknown as ResizeObserverEntry], this);
        }
        unobserve(): void {}
        disconnect(): void {}
      }
      globalRef.ResizeObserver = MockResizeObserver;
      restore = () => {
        globalRef.ResizeObserver = original;
      };
    }

    afterEach(() => {
      restore?.();
      restore = null;
    });

    it("keeps secondary controls inline and hides the overflow toggle when wide", () => {
      withContainerWidth(TOOLBAR_INLINE_MIN_WIDTH + 200);
      renderWithProviders(<FiltersHarness />);
      expect(
        screen.getByTestId(selectors.components.emulator.visionFilterSelect),
      ).toBeInTheDocument();
      expect(
        screen.getByTestId(selectors.components.emulator.blurSlider),
      ).toBeInTheDocument();
      expect(
        screen.queryByTestId(selectors.components.emulator.overflowToggle),
      ).toBeNull();
    });

    it("folds vision/blur/rotate into an overflow menu when narrow", async () => {
      const user = userEvent.setup();
      withContainerWidth(TOOLBAR_INLINE_MIN_WIDTH - 240);
      renderWithProviders(<FiltersHarness />);

      // Secondary controls are not inline until the menu is opened.
      expect(
        screen.queryByTestId(selectors.components.emulator.visionFilterSelect),
      ).toBeNull();
      expect(
        screen.queryByTestId(selectors.components.emulator.rotate),
      ).toBeNull();
      // Primary device controls remain inline.
      expect(
        screen.getByTestId(selectors.components.emulator.presetSelect),
      ).toBeInTheDocument();
      expect(
        screen.getByTestId(selectors.components.emulator.zoomValue),
      ).toBeInTheDocument();

      const toggle = screen.getByTestId(selectors.components.emulator.overflowToggle);
      await user.click(toggle);

      const panel = screen.getByTestId(selectors.components.emulator.overflowPanel);
      expect(panel).toBeInTheDocument();
      expect(
        screen.getByTestId(selectors.components.emulator.visionFilterSelect),
      ).toBeInTheDocument();
      expect(
        screen.getByTestId(selectors.components.emulator.blurSlider),
      ).toBeInTheDocument();
      expect(
        screen.getByTestId(selectors.components.emulator.rotate),
      ).toBeInTheDocument();
    });

    it("closes the open overflow menu on an outside click and on Escape", async () => {
      const user = userEvent.setup();
      withContainerWidth(TOOLBAR_INLINE_MIN_WIDTH - 240);
      renderWithProviders(<FiltersHarness />);

      const toggle = screen.getByTestId(selectors.components.emulator.overflowToggle);
      await user.click(toggle);
      expect(
        screen.getByTestId(selectors.components.emulator.overflowPanel),
      ).toBeInTheDocument();

      // A mousedown outside the panel dismisses it.
      fireEvent.mouseDown(document.body);
      await waitFor(() => {
        expect(
          screen.queryByTestId(selectors.components.emulator.overflowPanel),
        ).toBeNull();
      });

      // Reopen, then Escape dismisses it.
      await user.click(toggle);
      expect(
        screen.getByTestId(selectors.components.emulator.overflowPanel),
      ).toBeInTheDocument();
      fireEvent.keyDown(window, { key: "Escape" });
      await waitFor(() => {
        expect(
          screen.queryByTestId(selectors.components.emulator.overflowPanel),
        ).toBeNull();
      });
    });
  });
});
