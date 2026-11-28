import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../ui/src/test-utils";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1.3.5";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1.4.5";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1.3.5";

const subjects = [
  { Component: BottomSheet, attribute: "data-rcl-bottom-sheet" },
  { Component: FullPageDrawer, attribute: "data-rcl-full-page-drawer" },
  { Component: ResponsiveDialog, attribute: "data-rcl-responsive-dialog" },
];

describe("overlay canvas selector scope", () => {
  for (const { Component, attribute } of subjects) {
    it(`${attribute} paints for open portal children and ignores transcript descendants`, () => {
      const { rerender, container } = renderWithProviders(
        <Component open title="Canvas" closeLabel="Close" testId="canvas-overlay">body</Component>,
      );
      const css = [...document.head.querySelectorAll("style")].map((node) => node.textContent ?? "").join("\n");
      const selector = css.includes("html[data-rcl-overlay-canvas]") ? "html[data-rcl-overlay-canvas]" : undefined;
      expect(selector).toBeDefined();
      expect(document.querySelector(selector ?? "never")).toBe(document.documentElement);

      rerender(<Component open={false} title="Canvas" closeLabel="Close" testId="canvas-overlay">body</Component>);
      const nested = document.createElement("div");
      nested.setAttribute(attribute, "");
      nested.dataset.state = "open";
      container.append(nested);
      expect(document.querySelector(selector ?? "never")).toBeNull();
    });
  }
});
