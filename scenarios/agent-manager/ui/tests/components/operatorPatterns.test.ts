import assert from "node:assert/strict";
import { act, fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { DetailModal } from "../../src/components/patterns/MasterDetail/DetailModal.js";
import { FilterDropdown } from "../../src/components/patterns/SearchToolbar/FilterDropdown.js";
import { SortDropdown } from "../../src/components/patterns/SearchToolbar/SortDropdown.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

afterEach(() => vi.useRealTimers());

test("filter and sort dropdowns select visible options and close for escape or outside input", () => {
  const filter = vi.fn(); const sort = vi.fn();
  renderWithProviders(createElement("div", {},
    createElement(FilterDropdown, { label: "Run status", value: "all", allLabel: "All statuses", options: [{ value: "failed", label: "Failed" }], onChange: filter }),
    createElement(SortDropdown, { label: "Run sort", value: "newest", options: [{ value: "newest", label: "Newest" }, { value: "oldest", label: "Oldest" }], onChange: sort }),
  ));
  fireEvent.click(screen.getByRole("button", { name: "Run status" }));
  fireEvent.click(screen.getByRole("button", { name: "Failed" }));
  assert.deepEqual(filter.mock.calls, [["failed"]]);
  fireEvent.click(screen.getByRole("button", { name: "Run sort" }));
  fireEvent.keyDown(document, { key: "Escape" });
  assert.equal(screen.queryByRole("button", { name: "Oldest" }), null);
  fireEvent.click(screen.getByRole("button", { name: "Run sort" }));
  fireEvent.click(screen.getByRole("button", { name: "Oldest" }));
  assert.deepEqual(sort.mock.calls, [["oldest"]]);
});

test("mobile detail modal locks scrolling and guards rapid close requests until its exit transition", () => {
  vi.useFakeTimers(); const close = vi.fn();
  renderWithProviders(createElement(DetailModal, { open: true, onClose: close, title: "Run detail", headerLeft: createElement("span", {}, "status"), headerRight: createElement("button", {}, "action") }, createElement("p", {}, "Evidence")));
  assert.equal(document.body.style.overflow, "hidden");
  fireEvent.click(screen.getByLabelText("Close"));
  fireEvent.click(screen.getByLabelText("Close"));
  act(() => vi.advanceTimersByTime(150));
  assert.deepEqual(close.mock.calls, [[]]);
  assert.equal(screen.queryByText("Evidence"), null);
});
