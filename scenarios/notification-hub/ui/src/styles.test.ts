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
});
