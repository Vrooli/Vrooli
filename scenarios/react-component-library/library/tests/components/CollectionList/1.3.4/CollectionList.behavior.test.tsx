import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CollectionList } from "@vrooli/react-component-library/CollectionList/1.3.4";

type Row = { id: string; name: string };
const rows: Row[] = [
  { id: "a", name: "Alpha" },
  { id: "b", name: "Beta" },
];

function collectionStyles() {
  return [...document.querySelectorAll("style")].map((style) => style.textContent ?? "").join("\n");
}

describe("CollectionList 1.3.4 virtualized sizing", () => {
  // jsdom does no layout, so this pins the contract the browser relies on: a
  // virtualized list is marked, and the marked list and its wrappers fill a
  // sized container. Without that the viewport's percentage height resolved
  // to auto, the viewport grew to the full content, and every row mounted.
  it("marks a virtualized list so it fills its container", () => {
    render(
      <CollectionList
        items={rows}
        getKey={(row) => row.id}
        label="Rows"
        virtualize
        renderItem={(row) => <span>{row.name}</span>}
      />,
    );
    expect(document.querySelector(".rcl-collection-list")).toHaveAttribute("data-rcl-virtualized");
    const styles = collectionStyles();
    expect(styles).toMatch(
      /\.rcl-collection-list\[data-rcl-virtualized\]\s*\{\s*display:\s*flex;\s*flex-direction:\s*column;\s*block-size:\s*100%;\s*\}/,
    );
    expect(styles).toMatch(
      /\.rcl-collection-list\[data-rcl-virtualized\] > \[data-rcl-collection-container\],\s*\.rcl-collection-list\[data-rcl-virtualized\]\s+\.rcl-collection-list__virtual\[data-rcl-virtual-list\]\s*\{\s*display:\s*flex;\s*flex:\s*1 1 0%;\s*flex-direction:\s*column;\s*min-block-size:\s*0;\s*\}/,
    );
  });

  it("leaves an unvirtualized list to size to its rows", () => {
    render(
      <CollectionList items={rows} getKey={(row) => row.id} label="Rows" renderItem={(row) => <span>{row.name}</span>} />,
    );
    expect(document.querySelector(".rcl-collection-list")).not.toHaveAttribute("data-rcl-virtualized");
  });
});

describe("CollectionList 1.3.4 row content", () => {
  it("adds no padding around scenario-owned row content", () => {
    render(
      <CollectionList
        items={rows}
        getKey={(row) => row.id}
        label="Rows"
        renderItem={(row) => <span>{row.name}</span>}
      />,
    );
    expect(screen.getByText("Alpha").closest("[data-rcl-card-content]")).toBeTruthy();
    expect(collectionStyles()).toMatch(
      /\.rcl-collection-list \[data-rcl-card-content\]\s*\{\s*padding:\s*0;\s*\}/,
    );
  });

  it.each([false, true])("separates adjacent rows inside the row box when virtualize=%s", (virtualize) => {
    render(
      <CollectionList
        items={rows}
        getKey={(row) => row.id}
        label="Rows"
        virtualize={virtualize}
        renderItem={(row) => <span>{row.name}</span>}
      />,
    );
    expect(screen.getByText("Alpha").closest("[data-rcl-collection-key]")).toBeTruthy();
    expect(collectionStyles()).toMatch(
      /\.rcl-collection-list \[data-rcl-collection-key\]\s*\{\s*padding-block-end:\s*var\(--space-3xs\);\s*\}/,
    );
  });

  it("lets presses reach controls inside row content without capturing the pointer", () => {
    const setPointerCapture = vi.fn();
    const original = Element.prototype.setPointerCapture;
    Element.prototype.setPointerCapture = setPointerCapture;
    try {
      const onPress = vi.fn();
      render(
        <CollectionList
          items={rows}
          getKey={(row) => row.id}
          label="Rows"
          actions={[{ id: "archive", label: "Archive", onSelect: vi.fn() }]}
          renderItem={(row) => (
            <button type="button" onClick={() => onPress(row.id)}>
              {row.name}
            </button>
          )}
        />,
      );
      const button = screen.getByRole("button", { name: "Alpha" });
      fireEvent.pointerDown(button, { pointerId: 1, pointerType: "mouse", button: 0 });
      fireEvent.pointerUp(button, { pointerId: 1, pointerType: "mouse" });
      fireEvent.click(button);
      expect(onPress).toHaveBeenCalledWith("a");
      expect(setPointerCapture).not.toHaveBeenCalled();
      expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    } finally {
      Element.prototype.setPointerCapture = original;
    }
  });
});
