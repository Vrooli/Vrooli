import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { Tabs } from "./tabs";

afterEach(cleanup);

const items = [
  { value: "a", label: "Tab A" },
  { value: "b", label: "Tab B" },
  { value: "c", label: "Tab C" },
];

function renderTabs(props: Partial<Parameters<typeof Tabs>[0]> = {}) {
  return render(
    <Tabs items={items} ariaLabel="Test tabs" {...props}>
      {(active) => <div data-testid="panel-content">{active}</div>}
    </Tabs>,
  );
}

describe("Tabs", () => {
  describe("default/uncontrolled", () => {
    it("renders all tab buttons", () => {
      renderTabs();
      expect(screen.getAllByRole("tab")).toHaveLength(3);
    });

    it("first tab is selected by default", () => {
      renderTabs();
      const tabs = screen.getAllByRole("tab");
      expect(tabs[0]).toHaveAttribute("aria-selected", "true");
      expect(tabs[1]).toHaveAttribute("aria-selected", "false");
    });

    it("respects defaultValue", () => {
      renderTabs({ defaultValue: "b" });
      const tabs = screen.getAllByRole("tab");
      expect(tabs[0]).toHaveAttribute("aria-selected", "false");
      expect(tabs[1]).toHaveAttribute("aria-selected", "true");
    });

    it("clicking a tab selects it", async () => {
      const user = userEvent.setup();
      renderTabs();
      await user.click(screen.getByRole("tab", { name: "Tab B" }));
      expect(screen.getByRole("tab", { name: "Tab B" })).toHaveAttribute("aria-selected", "true");
    });

    it("children receive the active tab value", async () => {
      const user = userEvent.setup();
      renderTabs();
      await user.click(screen.getByRole("tab", { name: "Tab C" }));
      expect(screen.getByTestId("panel-content")).toHaveTextContent("c");
    });

    it("calls onValueChange when tab is clicked", async () => {
      const user = userEvent.setup();
      const onValueChange = vi.fn();
      renderTabs({ onValueChange });
      await user.click(screen.getByRole("tab", { name: "Tab B" }));
      expect(onValueChange).toHaveBeenCalledWith("b");
    });
  });

  describe("controlled mode", () => {
    it("respects externally provided value", () => {
      renderTabs({ value: "c" });
      expect(screen.getByRole("tab", { name: "Tab C" })).toHaveAttribute("aria-selected", "true");
    });

    it("does not update internal state in controlled mode", async () => {
      const user = userEvent.setup();
      const onValueChange = vi.fn();
      renderTabs({ value: "a", onValueChange });
      await user.click(screen.getByRole("tab", { name: "Tab B" }));
      // controlled: value stays "a", onValueChange fires but internal state unchanged
      expect(screen.getByRole("tab", { name: "Tab A" })).toHaveAttribute("aria-selected", "true");
      expect(onValueChange).toHaveBeenCalledWith("b");
    });
  });

  describe("keyboard navigation", () => {
    it("ArrowRight moves to next tab", async () => {
      const user = userEvent.setup();
      renderTabs();
      const tabA = screen.getByRole("tab", { name: "Tab A" });
      tabA.focus();
      await user.keyboard("{ArrowRight}");
      expect(screen.getByRole("tab", { name: "Tab B" })).toHaveAttribute("aria-selected", "true");
    });

    it("ArrowLeft moves to previous tab", async () => {
      const user = userEvent.setup();
      renderTabs({ defaultValue: "b" });
      const tabB = screen.getByRole("tab", { name: "Tab B" });
      tabB.focus();
      await user.keyboard("{ArrowLeft}");
      expect(screen.getByRole("tab", { name: "Tab A" })).toHaveAttribute("aria-selected", "true");
    });

    it("ArrowRight wraps around from last to first", async () => {
      const user = userEvent.setup();
      renderTabs({ defaultValue: "c" });
      const tabC = screen.getByRole("tab", { name: "Tab C" });
      tabC.focus();
      await user.keyboard("{ArrowRight}");
      expect(screen.getByRole("tab", { name: "Tab A" })).toHaveAttribute("aria-selected", "true");
    });

    it("ArrowLeft wraps around from first to last", async () => {
      const user = userEvent.setup();
      renderTabs();
      const tabA = screen.getByRole("tab", { name: "Tab A" });
      tabA.focus();
      await user.keyboard("{ArrowLeft}");
      expect(screen.getByRole("tab", { name: "Tab C" })).toHaveAttribute("aria-selected", "true");
    });

    it("ArrowRight skips disabled tabs", async () => {
      const user = userEvent.setup();
      const itemsWithDisabled = [
        { value: "a", label: "Tab A" },
        { value: "b", label: "Tab B", disabled: true },
        { value: "c", label: "Tab C" },
      ];
      render(
        <Tabs items={itemsWithDisabled} ariaLabel="Test tabs">
          {(active) => <div>{active}</div>}
        </Tabs>,
      );
      const tabA = screen.getByRole("tab", { name: "Tab A" });
      tabA.focus();
      await user.keyboard("{ArrowRight}");
      expect(screen.getByRole("tab", { name: "Tab C" })).toHaveAttribute("aria-selected", "true");
    });

    it("ArrowLeft skips disabled tabs", async () => {
      const user = userEvent.setup();
      const itemsWithDisabled = [
        { value: "a", label: "Tab A" },
        { value: "b", label: "Tab B", disabled: true },
        { value: "c", label: "Tab C" },
      ];
      render(
        <Tabs items={itemsWithDisabled} defaultValue="c" ariaLabel="Test tabs">
          {(active) => <div>{active}</div>}
        </Tabs>,
      );
      const tabC = screen.getByRole("tab", { name: "Tab C" });
      tabC.focus();
      await user.keyboard("{ArrowLeft}");
      expect(screen.getByRole("tab", { name: "Tab A" })).toHaveAttribute("aria-selected", "true");
    });
  });

  describe("accessibility", () => {
    it("tablist has the provided aria-label", () => {
      renderTabs({ ariaLabel: "My tabs" });
      expect(screen.getByRole("tablist")).toHaveAttribute("aria-label", "My tabs");
    });

    it("tabpanel is labelled by the active tab", () => {
      renderTabs();
      const panel = screen.getByRole("tabpanel");
      expect(panel).toHaveAttribute("aria-labelledby", "tab-a");
    });

    it("disabled tab has disabled attribute", () => {
      const itemsWithDisabled = [
        { value: "a", label: "Tab A", disabled: true },
        { value: "b", label: "Tab B" },
      ];
      render(
        <Tabs items={itemsWithDisabled} ariaLabel="Test">
          {() => <div />}
        </Tabs>,
      );
      expect(screen.getByRole("tab", { name: "Tab A" })).toBeDisabled();
    });

    it("accepts optional className on the wrapper", () => {
      const { container } = renderTabs({ className: "custom-class" });
      expect(container.firstChild).toHaveClass("custom-class");
    });
  });
});

describe("Tabs edge cases", () => {
  it("defaults internal value to empty string when items array is empty", () => {
    // covers the `?? ""` fallback branch at defaultValue ?? items[0]?.value ?? ""
    render(
      <Tabs items={[]} ariaLabel="Empty">
        {(active) => <div data-testid="content">{active}</div>}
      </Tabs>,
    );
    expect(screen.getByTestId("content")).toHaveTextContent("");
    cleanup();
  });

  it("keyboard navigation does nothing when active value is not in items (idx < 0)", async () => {
    // controlled with a value that's not in items — exercises idx < 0 early-return
    const user = userEvent.setup();
    const onValueChange = vi.fn();
    render(
      <Tabs items={[{ value: "a", label: "A" }, { value: "b", label: "B" }]} value="z" onValueChange={onValueChange} ariaLabel="Test">
        {() => <div />}
      </Tabs>,
    );
    // Focus the first rendered tab button and press ArrowRight
    const tabBtns = screen.getAllByRole("tab");
    tabBtns[0]?.focus();
    await user.keyboard("{ArrowRight}");
    // onValueChange should not have been called (no valid active idx)
    expect(onValueChange).not.toHaveBeenCalled();
    cleanup();
  });
});
