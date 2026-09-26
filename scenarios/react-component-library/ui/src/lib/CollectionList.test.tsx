import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { CollectionList } from "@vrooli/react-component-library/CollectionList/1.3.0";
afterEach(cleanup);
const items = [{ id: "support", title: "Support" }, { id: "triage", title: "Triage" }];
it("keeps the open detail distinct from keyboard navigation and bulk selection", () => {
  const onOpen = vi.fn();
  const { rerender } = render(<CollectionList items={items} label="Conversations" currentKey="triage" onOpen={onOpen} />);
  const rows = screen.getAllByRole("listitem");
  expect(rows[0]!).not.toHaveAttribute("aria-current");
  expect(rows[1]!).toHaveAttribute("aria-current", "true");
  fireEvent.focus(rows[0]!);
  fireEvent.keyDown(rows[0]!, { key: "Enter" });
  expect(onOpen).toHaveBeenCalledWith(items[0]);
  expect(rows[1]!).toHaveAttribute("aria-current", "true");
  expect(rows[1]!).not.toHaveAttribute("aria-selected");
  rerender(<CollectionList items={items} label="Conversations" currentKey="support" onOpen={onOpen} />);
  const updatedRows = screen.getAllByRole("listitem");
  expect(updatedRows[0]!).toHaveAttribute("aria-current", "true");
  expect(updatedRows[1]!).not.toHaveAttribute("aria-current");
});
it("does not substitute another current item when the open record is filtered out", () => {
  render(<CollectionList items={items.slice(0, 1)} label="Conversations" currentKey="triage" />);
  expect(screen.getByRole("listitem")).not.toHaveAttribute("aria-current");
});
