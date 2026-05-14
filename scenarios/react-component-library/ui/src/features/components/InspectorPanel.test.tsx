import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { InspectorPanel } from "./InspectorPanel";
import type { UseComponentInspectorReturn } from "../../hooks/useComponentInspector";

function makeInspector(overrides: Partial<UseComponentInspectorReturn> = {}): UseComponentInspectorReturn {
  return {
    active: false,
    hover: null,
    result: null,
    lastReason: null,
    selected: null,
    startInspect: vi.fn(() => true),
    stopInspect: vi.fn(() => true),
    ...overrides,
  };
}

describe("InspectorPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
  });

  it("shows empty state and idle status by default", () => {
    renderWithProviders(<InspectorPanel inspector={makeInspector()} />);
    expect(screen.getByTestId(selectors.components.inspector.empty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.components.inspector.statusBadge).textContent).toContain("Idle");
  });

  it("toggle button calls startInspect when idle, stopInspect when active", async () => {
    const start = vi.fn(() => true);
    const stop = vi.fn(() => true);
    const user = userEvent.setup();
    const { rerender } = renderWithProviders(
      <InspectorPanel inspector={makeInspector({ startInspect: start, stopInspect: stop })} />,
    );
    await user.click(screen.getByTestId(selectors.components.inspector.toggleButton));
    expect(start).toHaveBeenCalled();
    expect(stop).not.toHaveBeenCalled();

    rerender(
      <InspectorPanel
        inspector={makeInspector({ active: true, startInspect: start, stopInspect: stop })}
      />,
    );
    await user.click(screen.getByTestId(selectors.components.inspector.toggleButton));
    expect(stop).toHaveBeenCalled();
  });

  it("renders selected element details + ancestor breadcrumb", () => {
    const selected = {
      meta: {
        tag: "button",
        id: "submit",
        classes: ["primary"],
        selector: "#submit",
        label: "",
        ariaLabel: "",
        ariaDescription: "",
        title: "",
        role: "",
        text: "Save changes",
      },
      rect: { x: 10, y: 20, width: 80, height: 32 },
      documentRect: { x: 10, y: 20, width: 80, height: 32 },
      ancestors: [
        { depth: 0, tag: "button", selector: "#submit", id: "submit", classes: ["primary"], rect: null, documentRect: null },
        { depth: 1, tag: "form", selector: "form.login", id: "", classes: ["login"], rect: null, documentRect: null },
        { depth: 2, tag: "div", selector: "div.shell", id: "", classes: ["shell"], rect: null, documentRect: null },
      ],
      selectedAncestorIndex: 0,
    };
    renderWithProviders(
      <InspectorPanel
        inspector={makeInspector({ result: selected, selected, lastReason: "complete" })}
      />,
    );
    expect(screen.getByTestId(selectors.components.inspector.selectedTag).textContent).toContain("button");
    expect(screen.getByTestId(selectors.components.inspector.selectedSelector).textContent).toContain("#submit");
    expect(screen.getByTestId(selectors.components.inspector.selectedRect).textContent).toContain("80×32");
    expect(screen.getByTestId(selectors.components.inspector.selectedText).textContent).toContain("Save changes");
    const crumbs = screen.getAllByTestId(selectors.components.inspector.breadcrumbItem);
    expect(crumbs).toHaveLength(3);
    expect(crumbs[0]?.textContent).toContain("div.shell");
    expect(crumbs[2]?.textContent).toContain("#submit");
    expect(screen.getByTestId(selectors.components.inspector.statusBadge).textContent).toContain("selected");
  });

  it("shows active status when inspector is active", () => {
    renderWithProviders(<InspectorPanel inspector={makeInspector({ active: true })} />);
    expect(screen.getByTestId(selectors.components.inspector.statusBadge).textContent).toContain("Click");
  });
});
