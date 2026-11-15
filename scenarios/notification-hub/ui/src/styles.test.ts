import { readFileSync } from "node:fs";
import { createElement } from "react";
import { screen } from "@testing-library/react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "./test-utils";

// jsdom does not apply the cascade between Tailwind's reset and the component
// library's stylesheet, so the rule that keeps library controls outlined is
// pinned in the source stylesheet.
describe("styles.css", () => {
  it("restores the border width Tailwind's reset removes from library controls", () => {
    expect(readFileSync("src/styles.css", "utf8")).toMatch(/\[data-rcl-control\]\s*\{\s*border-width:\s*var\(--border-hairline\);/);
  });

  // The 44px touch floor is the library's: ControlBase raises its sub-44px
  // rungs on a coarse pointer. The Hub used to carry its own override, which
  // also lowered `xl` to 44px. jsdom cannot evaluate the media query, so this
  // pins both halves of the contract: the Hub ships no second copy, and the
  // control the Hub renders carries the library rule and opts into it.
  it("takes the coarse-pointer tap-target floor from the library, not from a local override", () => {
    expect(readFileSync("src/styles.css", "utf8")).not.toMatch(/pointer:\s*coarse|min-block-size/);

    renderWithProviders(createElement(Button, { size: "xs", children: "Open" }));
    expect(screen.getByRole("button", { name: "Open" })).toHaveAttribute("data-rcl-tap-target", "comfortable");
    const sheet = document.querySelector('style[data-rcl-sheet^="react-component-library-controlbase"]');
    expect(sheet?.textContent).toMatch(/@media \(pointer: coarse\)[\s\S]*min-block-size: var\(--tap-target-min\)/);
    expect(readFileSync("src/design-tokens.css", "utf8")).toMatch(/--tap-target-min:\s*44px;/);
  });

  // jsdom cannot lay out a sticky bar, so the contract is pinned in the stylesheet: the dock
  // cancels exactly the gutter the shell adds (so no page content shows below it), the tab bar
  // alone owns the phone's bottom inset, and the pane contains what is absolutely positioned
  // inside it (otherwise the whole document scrolls and slides the tab bar away).
  describe("the pinned answer dock", () => {
    const css = readFileSync("src/styles.css", "utf8");
    const dockRule = /\.packet-dock\s*\{([^}]*)\}/.exec(css)?.[1] ?? "";

    it("cancels the pane's gutter below, and on both sides", () => {
      expect(css).toMatch(/\[data-rcl-app-shell-main\]\s*\{[^}]*--page-gutter:\s*clamp\(var\(--space-sm\), 3vw, var\(--space-lg\)\)/);
      expect(dockRule).toMatch(/position:\s*sticky/);
      expect(dockRule).toMatch(/bottom:\s*calc\(-1 \* var\(--page-gutter\) - var\(--dock-inset\)\)/);
      expect(dockRule).toMatch(/margin-inline:\s*calc\(-1 \* var\(--page-gutter\)\)/);
    });

    it("adds the device's bottom inset only beside a sidebar, never above the phone's tab bar", () => {
      expect(dockRule).toMatch(/--dock-inset:\s*0px/);
      const inset = [...css.matchAll(/--dock-inset:\s*([^;]+);/g)].map((match) => match[1]);
      expect(inset).toEqual(["0px", "env(safe-area-inset-bottom, 0px)", "0px"]);
      expect(/@media \(min-width: 48rem\)\s*\{\s*\.packet-dock\s*\{\s*--dock-inset:\s*env\(safe-area-inset-bottom/.test(css)).toBe(true);
    });

    it("keeps the main pane the containing block of everything inside it", () => {
      expect(css).toMatch(/\[data-rcl-app-shell-main\]\s*\{[^}]*position:\s*relative/);
    });
  });
});
