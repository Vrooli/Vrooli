import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test } from "vitest";
import { useResizablePanel } from "../../src/hooks/useResizablePanel.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

function PanelHarness() {
  const panel = useResizablePanel({ storageKey: "test", defaultSize: 300, minSize: 100, minOtherSize: 200 });
  return createElement("div", { ref: panel.containerRef, "data-testid": "panel" },
    createElement("output", { "data-testid": "size" }, String(panel.size)),
    createElement("button", { onMouseDown: panel.handleResizeStart }, "Resize"),
  );
}

function VerticalPanelHarness() {
  const panel = useResizablePanel({
    storageKey: "vertical-test",
    persistKey: "legacy.vertical.panel",
    axis: "vertical",
    defaultSize: 250,
    minSize: 100,
    minOtherSize: 150,
  });
  return createElement("div", { ref: panel.containerRef, "data-testid": "vertical-panel" },
    createElement("output", { "data-testid": "vertical-size" }, String(panel.width)),
    createElement("button", { onMouseDown: panel.handleResizeStart }, "Resize vertically"),
  );
}

afterEach(() => localStorage.clear());

test("useResizablePanel restores persisted size, clamps a drag, and persists only its committed result", async () => {
  localStorage.setItem("agm.panel.test.width", "300");
  const originalClientWidth = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientWidth");
  Object.defineProperty(HTMLElement.prototype, "clientWidth", { configurable: true, get: () => 700 });
  renderWithProviders(createElement(PanelHarness));
  const panel = screen.getByTestId("panel");
  assert.equal(screen.getByTestId("size").textContent, "300");
  fireEvent.mouseDown(screen.getByRole("button", { name: "Resize" }), { clientX: 100 });
  assert.equal(document.body.style.cursor, "col-resize");
  fireEvent.mouseMove(window, { clientX: 650 });
  fireEvent.mouseUp(window);
  await waitFor(() => assert.equal(screen.getByTestId("size").textContent, "500"));
  assert.equal(localStorage.getItem("agm.panel.test.width"), "500");
  assert.equal(document.body.style.cursor, "");
  if (originalClientWidth) Object.defineProperty(HTMLElement.prototype, "clientWidth", originalClientWidth);
});

test("useResizablePanel supports vertical legacy persistence and clamps the drag to remaining height", async () => {
  localStorage.setItem("legacy.vertical.panel", "not-a-number");
  const originalClientHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientHeight");
  Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get: () => 600 });
  renderWithProviders(createElement(VerticalPanelHarness));
  assert.equal(screen.getByTestId("vertical-size").textContent, "250");
  fireEvent.mouseDown(screen.getByRole("button", { name: "Resize vertically" }), { clientY: 100 });
  assert.equal(document.body.style.cursor, "row-resize");
  fireEvent.mouseMove(window, { clientY: -500 });
  fireEvent.mouseUp(window);
  await waitFor(() => assert.equal(screen.getByTestId("vertical-size").textContent, "100"));
  assert.equal(localStorage.getItem("legacy.vertical.panel"), "100");
  if (originalClientHeight) Object.defineProperty(HTMLElement.prototype, "clientHeight", originalClientHeight);
});
