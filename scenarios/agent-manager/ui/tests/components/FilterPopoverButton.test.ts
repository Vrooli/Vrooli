import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { FilterPopoverButton } from "../../src/components/patterns/SearchToolbar/FilterPopoverButton.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("FilterPopoverButton signals active criteria, resets every control, and closes through keyboard/outside controls", () => {
  const status = vi.fn(); const sort = vi.fn();
  renderWithProviders(createElement(FilterPopoverButton, {
    filters: [{ id: "status", label: "Status", value: "failed", defaultValue: "all", onChange: status, options: [{ value: "failed", label: "Failed" }] }],
    sortOptions: [{ value: "newest", label: "Newest" }, { value: "oldest", label: "Oldest" }], currentSort: "oldest", defaultSort: "newest", onSortChange: sort,
  }));
  const trigger = screen.getByRole("button", { name: "Filter and sort options" });
  assert.match(trigger.className, /relative/);
  fireEvent.click(trigger); assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.ok(screen.getByText("Reset filters")); assert.ok(screen.getByText("Sort by"));
  fireEvent.click(screen.getByText("Reset filters"));
  assert.deepEqual(status.mock.calls, [["all"]]); assert.deepEqual(sort.mock.calls, [["newest"]]);
  fireEvent.keyDown(document, { key: "Escape" }); assert.equal(screen.queryByText("Reset filters"), null);
  fireEvent.click(trigger); fireEvent.mouseDown(document.body); assert.equal(screen.queryByText("Reset filters"), null);
});
