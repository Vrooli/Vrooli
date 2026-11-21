import { describe, expect, it } from "vitest";
import { fireEvent } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { ResizableMarkdownTable } from "@vrooli/react-component-library/markdown-renderer/0.4.5";

function table() {
  return renderWithProviders(
    <ResizableMarkdownTable>
      <thead>
        <tr><th>Plan</th><th>Id</th><th>Phases</th></tr>
      </thead>
      <tbody>
        <tr><td>Platform</td><td>ede4a896</td><td>11</td></tr>
      </tbody>
    </ResizableMarkdownTable>,
  );
}

const injectedCss = () => [...document.head.querySelectorAll("style")].map((node) => node.textContent ?? "").join("\n");

describe("ResizableMarkdownTable 0.4.5", () => {
  // The defect: 0.4.4 kept these rules in the renderer's sheet, scoped to the
  // renderer's root, so a consumer rendering only the table got a zero-size,
  // unstyled handle that could not be dragged by mouse or touch.
  it("brings its own stylesheet, so its handles work outside this package's renderer", () => {
    const { container } = table();
    expect(container.querySelector("[data-rcl-markdown]")).toBeNull();
    expect(container.querySelector("[data-rcl-md-table]")).not.toBeNull();
    expect(container.querySelectorAll("[data-rcl-md-resize-handle]")).toHaveLength(2);

    const css = injectedCss();
    expect(css).toMatch(/\[data-rcl-md-resize-handle\]\s*\{[^}]*position:\s*absolute[^}]*touch-action:\s*none/);
    expect(css).toMatch(/\[data-rcl-md-table\]\s+th\s*\{[^}]*position:\s*relative/);
    expect(css).not.toMatch(/\[data-rcl-markdown\][^{,]*\[data-rcl-md-resize-handle\]/);
  });

  it("gives a coarse pointer a wider handle with a grip that is always drawn", () => {
    table();
    expect(injectedCss()).toMatch(/@media \(pointer: coarse\)\s*\{[^@]*\[data-rcl-md-resize-handle\]\s*\{[^}]*inline-size:\s*24px/);
  });

  it("moves width between the two columns beside a handle, leaving the table's total alone", () => {
    const { container } = table();
    const handle = container.querySelector<HTMLElement>("[data-rcl-md-resize-handle]");
    if (!handle) throw new Error("no resize handle");
    fireEvent.keyDown(handle, { key: "ArrowRight" });
    const widths = [...container.querySelectorAll<HTMLTableColElement>("col")].map((col) => Number.parseFloat(col.style.width));
    expect(widths).toHaveLength(3);
    expect((widths[0] ?? 0) - (widths[2] ?? 0)).toBe(16);
    expect((widths[2] ?? 0) - (widths[1] ?? 0)).toBe(16);
  });
});
