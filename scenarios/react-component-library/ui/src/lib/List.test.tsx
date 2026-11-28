import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { List, ListItemRow } from "../../../library/components/List/versions/1.2.0/List";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("List surface and plain variants", () => {
  it("preserves the default surfaced semantic section", () => {
    const title = "First item";
    renderWithProviders(<List items={[{ id: "one", title }]} />);
    expect(screen.getByTestId("data-display.list").tagName).toBe("SECTION");
    expect(screen.getByTestId("data-display.list")).toHaveAttribute("data-variant", "surface");
    expect(screen.getByText(title)).toBeInTheDocument();
    expect(document.querySelector("[data-rcl-list-mark]")).toBeInTheDocument();
  });

  it("renders a compact role list without a surface or status dot", () => {
    const label = "Unreadable file evidence";
    renderWithProviders(
      <List variant="plain" label={label} items={[
        { id: "one", title: "No access", description: <a href="/STATE.md">STATE.md</a> },
        { id: "two", title: <a href="/LEDGER.md">LEDGER.md</a> },
      ]} />,
    );
    expect(screen.getByRole("list", { name: label })).toHaveAttribute("data-variant", "plain");
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(screen.getByRole("link", { name: "STATE.md" })).toHaveAttribute("href", "/STATE.md");
    expect(document.querySelector("[data-rcl-list-mark]")).toBeNull();
  });

  it("renders a semantic li with primary, aside, content and footer slots", () => {
    const linkLabel = "Commission title";
    const asideText = "Running";
    const contentText = "Last write 5 minutes ago";
    const actionLabel = "Open packet";
    renderWithProviders(
      <ul>
        <ListItemRow
          primary={<a href="/commissions/c1">{linkLabel}</a>}
          aside={<span>{asideText}</span>}
          content={<p>{contentText}</p>}
          footer={<button type="button">{actionLabel}</button>}
        />
      </ul>,
    );

    const row = screen.getByRole("listitem");
    expect(row.tagName).toBe("LI");
    expect(row).toHaveAttribute("data-rcl-list-item-row");
    expect(screen.getByRole("link", { name: linkLabel })).toHaveAttribute("href", "/commissions/c1");
    expect(screen.getByText(asideText).parentElement).toHaveAttribute("data-rcl-list-item-row-aside");
    expect(screen.getByText(contentText).parentElement).toHaveAttribute("data-rcl-list-item-row-content");
    expect(screen.getByRole("button", { name: actionLabel }).parentElement).toHaveAttribute("data-rcl-list-item-row-footer");
  });
});
