import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1.5.1";
import { FullPageDrawer as FullPageDrawer147 } from "@vrooli/react-component-library/FullPageDrawer/1.4.7";

// 1.4.5 through 1.5.0 keyed their stylesheet as 1.4.3 while shipping
// different CSS, so mounting two of them on one page collided (the shared
// setup fails a test on the collision's console.error).
describe("FullPageDrawer stylesheet identity", () => {
  it("keys its stylesheet by its own version and coexists with an older release", () => {
    renderWithProviders(
      <>
        <FullPageDrawer open title="Current" closeLabel="Close" testId="current">body</FullPageDrawer>
        <FullPageDrawer147 open title="Older" closeLabel="Close" testId="older">body</FullPageDrawer147>
      </>,
    );
    const keys = [...document.querySelectorAll("style[data-rcl-sheet]")]
      .map((style) => style.getAttribute("data-rcl-sheet"))
      .filter((key) => key?.startsWith("react-component-library-fullpagedrawer-"));
    expect(keys).toContain("react-component-library-fullpagedrawer-1.5.1");
    expect(new Set(keys).size).toBe(keys.length);
  });
});
