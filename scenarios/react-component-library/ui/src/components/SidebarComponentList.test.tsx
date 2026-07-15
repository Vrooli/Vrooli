import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { makeComponent, makeListComponentsResponse } from "../features/components/mocks/factories";
import { makeComponentsMocks } from "../features/components/mocks/components";

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return { ...actual, ...makeComponentsMocks() };
});

import { SidebarComponentList } from "./SidebarComponentList";

const mockCatalog = () =>
  makeListComponentsResponse({
    components: [
      makeComponent({ id: "button", displayName: "Button", slot: "ui-primitive", category: "actions" }),
      makeComponent({ id: "icon-button", displayName: "Icon Button", slot: "ui-primitive", category: "actions" }),
      makeComponent({ id: "input", displayName: "Input", slot: "ui-primitive", category: "forms" }),
      makeComponent({ id: "dialog", displayName: "Dialog", slot: "ui-pattern", category: "overlays" }),
    ],
  });

const renderCatalog = async () => {
  const { componentsClient } = await import("../api/components");
  vi.mocked(componentsClient.listComponents).mockResolvedValueOnce(mockCatalog());
  const user = userEvent.setup();
  renderWithProviders(<SidebarComponentList />);
  await screen.findByTestId("sidebar-component-tree");
  return user;
};

describe("SidebarComponentList", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => cleanup());

  it("renders a two-level slot → category hierarchy with per-level counts", async () => {
    await renderCatalog();

    // Slot parents (level 1) are expandable toggles carrying the aggregate count.
    const primitiveSlot = screen.getByTestId("sidebar-component-slot-ui-primitive");
    expect(primitiveSlot.tagName).toBe("BUTTON");
    expect(primitiveSlot).toHaveAttribute("aria-expanded", "true");
    expect(primitiveSlot).toHaveTextContent("ui primitive");
    expect(primitiveSlot).toHaveTextContent("3");
    const patternSlot = screen.getByTestId("sidebar-component-slot-ui-pattern");
    expect(patternSlot).toHaveTextContent("ui pattern");
    expect(patternSlot).toHaveTextContent("1");

    // Category children (level 2) nested beneath their slot with their own count.
    const actionsCategory = screen.getByTestId("sidebar-component-category-ui-primitive-actions");
    expect(actionsCategory.tagName).toBe("BUTTON");
    expect(actionsCategory).toHaveAttribute("aria-expanded", "true");
    expect(actionsCategory).toHaveTextContent("actions");
    expect(actionsCategory).toHaveTextContent("2");
    expect(screen.getByTestId("sidebar-component-category-ui-primitive-forms")).toBeInTheDocument();

    // Component leaves (level 3) are links into the editor.
    const buttonLeaf = screen.getByTestId("sidebar-component-button");
    expect(buttonLeaf.tagName).toBe("A");
    expect(buttonLeaf).toHaveAttribute("href", "/components/button");
  });

  it("nests items deeper than their section headers (visual hierarchy)", async () => {
    await renderCatalog();

    // Left padding increases with depth: slot < category < component.
    expect(screen.getByTestId("sidebar-component-slot-ui-primitive").className).toContain("pl-2");
    expect(
      screen.getByTestId("sidebar-component-category-ui-primitive-actions").className,
    ).toContain("pl-6");
    expect(screen.getByTestId("sidebar-component-button").className).toContain("pl-10");
  });

  it("collapses a slot and hides its whole subtree, then re-expands", async () => {
    const user = await renderCatalog();

    const primitiveSlot = screen.getByTestId("sidebar-component-slot-ui-primitive");
    expect(primitiveSlot).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("sidebar-component-category-ui-primitive-actions")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-component-button")).toBeInTheDocument();

    await user.click(primitiveSlot);

    await waitFor(() => expect(primitiveSlot).toHaveAttribute("aria-expanded", "false"));
    // Entire subtree (categories + components) is removed from the DOM.
    expect(screen.queryByTestId("sidebar-component-category-ui-primitive-actions")).toBeNull();
    expect(screen.queryByTestId("sidebar-component-button")).toBeNull();
    // A sibling slot is unaffected.
    expect(screen.getByTestId("sidebar-component-slot-ui-pattern")).toBeInTheDocument();

    await user.click(primitiveSlot);
    await waitFor(() => expect(primitiveSlot).toHaveAttribute("aria-expanded", "true"));
    expect(screen.getByTestId("sidebar-component-button")).toBeInTheDocument();
  });

  it("collapses an individual category without hiding sibling categories", async () => {
    const user = await renderCatalog();

    const actions = screen.getByTestId("sidebar-component-category-ui-primitive-actions");
    await user.click(actions);

    await waitFor(() => expect(actions).toHaveAttribute("aria-expanded", "false"));
    expect(screen.queryByTestId("sidebar-component-button")).toBeNull();
    // Sibling category and its component remain visible.
    expect(screen.getByTestId("sidebar-component-category-ui-primitive-forms")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-component-input")).toBeInTheDocument();
  });

  it("supports roving-tabindex keyboard navigation with arrow keys", async () => {
    const user = await renderCatalog();

    const primitiveSlot = screen.getByTestId("sidebar-component-slot-ui-primitive");
    // Only the first row (ui-pattern sorts before ui-primitive) is in the tab
    // order; the rest are roving (-1).
    expect(screen.getByTestId("sidebar-component-slot-ui-pattern")).toHaveAttribute("tabindex", "0");
    expect(primitiveSlot).toHaveAttribute("tabindex", "-1");

    act(() => primitiveSlot.focus());

    // Down moves to the first category child.
    await user.keyboard("{ArrowDown}");
    const actions = screen.getByTestId("sidebar-component-category-ui-primitive-actions");
    await waitFor(() => expect(actions).toHaveFocus());
    expect(actions).toHaveAttribute("tabindex", "0");

    // Right on an expanded row moves to its first child (a component leaf).
    await user.keyboard("{ArrowRight}");
    await waitFor(() => expect(screen.getByTestId("sidebar-component-button")).toHaveFocus());

    // Left on a leaf returns focus to its parent category.
    await user.keyboard("{ArrowLeft}");
    await waitFor(() => expect(actions).toHaveFocus());

    // Left on an expanded parent collapses it (focus stays put).
    await user.keyboard("{ArrowLeft}");
    await waitFor(() => expect(actions).toHaveAttribute("aria-expanded", "false"));
    expect(actions).toHaveFocus();

    // Right re-expands the collapsed parent.
    await user.keyboard("{ArrowRight}");
    await waitFor(() => expect(actions).toHaveAttribute("aria-expanded", "true"));

    // Home jumps to the first row (ui-pattern slot); End to the last visible row
    // (the input leaf under ui-primitive → forms).
    await user.keyboard("{Home}");
    await waitFor(() =>
      expect(screen.getByTestId("sidebar-component-slot-ui-pattern")).toHaveFocus(),
    );
    await user.keyboard("{End}");
    await waitFor(() => expect(screen.getByTestId("sidebar-component-input")).toHaveFocus());
  });

  it("toggles a group with Enter and Space via native button activation", async () => {
    const user = await renderCatalog();

    const primitiveSlot = screen.getByTestId("sidebar-component-slot-ui-primitive");
    act(() => primitiveSlot.focus());

    await user.keyboard("{Enter}");
    await waitFor(() => expect(primitiveSlot).toHaveAttribute("aria-expanded", "false"));

    await user.keyboard(" ");
    await waitFor(() => expect(primitiveSlot).toHaveAttribute("aria-expanded", "true"));
  });

  it("groups components with no slot or category under the fallback labels", async () => {
    const { componentsClient } = await import("../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValueOnce(
      makeListComponentsResponse({
        components: [
          // No slot, no category, no displayName → falls back to "other" /
          // "uncategorized" grouping and the libraryId/id name.
          makeComponent({ id: "loose", displayName: "", libraryId: "lib:Loose", slot: "", category: "" }),
        ],
      }),
    );
    renderWithProviders(<SidebarComponentList />);

    expect(await screen.findByTestId("sidebar-component-slot-other")).toHaveTextContent("other");
    expect(
      screen.getByTestId("sidebar-component-category-other-uncategorized"),
    ).toHaveTextContent("uncategorized");
    // The leaf falls back to its libraryId when no displayName is present.
    expect(screen.getByTestId("sidebar-component-loose")).toHaveTextContent("lib:Loose");
  });

  it("names a component by its id when it has no displayName or libraryId", async () => {
    const { componentsClient } = await import("../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValueOnce(
      makeListComponentsResponse({
        components: [
          makeComponent({ id: "raw-leaf", displayName: "", libraryId: "", slot: "ui-primitive", category: "actions" }),
        ],
      }),
    );
    renderWithProviders(<SidebarComponentList />);
    // Falls all the way through displayName → libraryId → id.
    const leaf = await screen.findByTestId("sidebar-component-raw-leaf");
    expect(leaf).toHaveTextContent("raw-leaf");
  });

  it("navigates leaf and top-level rows with arrow keys as no-ops where appropriate", async () => {
    const user = await renderCatalog();

    // ArrowLeft on a top-level slot (no parent) collapses it; a second
    // ArrowLeft on the now-collapsed leafless slot is a no-op.
    const patternSlot = screen.getByTestId("sidebar-component-slot-ui-pattern");
    act(() => patternSlot.focus());
    await user.keyboard("{ArrowRight}"); // expanded → move to first child
    const overlays = screen.getByTestId("sidebar-component-category-ui-pattern-overlays");
    await waitFor(() => expect(overlays).toHaveFocus());

    // ArrowRight on a leaf (no children) is a no-op — focus stays on the leaf.
    await user.keyboard("{ArrowRight}");
    const dialogLeaf = screen.getByTestId("sidebar-component-dialog");
    await waitFor(() => expect(dialogLeaf).toHaveFocus());
    await user.keyboard("{ArrowRight}");
    await waitFor(() => expect(dialogLeaf).toHaveFocus());
  });

  it("treats arrow navigation past the first and last rows as a no-op", async () => {
    const user = await renderCatalog();

    const firstRow = screen.getByTestId("sidebar-component-slot-ui-pattern");
    act(() => firstRow.focus());
    // ArrowUp on the first row has no previous sibling — focus stays put.
    await user.keyboard("{ArrowUp}");
    await waitFor(() => expect(firstRow).toHaveFocus());

    // Jump to the last visible row and press ArrowDown — no next sibling.
    await user.keyboard("{End}");
    const lastRow = screen.getByTestId("sidebar-component-input");
    await waitFor(() => expect(lastRow).toHaveFocus());
    await user.keyboard("{ArrowDown}");
    await waitFor(() => expect(lastRow).toHaveFocus());
  });

  it("shows the empty state when no components are indexed", async () => {
    const { componentsClient } = await import("../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValueOnce(
      makeListComponentsResponse({ components: [] }),
    );
    renderWithProviders(<SidebarComponentList />);

    expect(await screen.findByTestId("sidebar-component-list-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-component-tree")).toBeNull();
  });
});
