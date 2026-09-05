import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { MobileNav } from "./MobileNav";
import { NAV_ITEMS } from "./nav-items";

afterEach(cleanup);

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

function renderNav() {
  return renderWithProviders(
    <MemoryRouter future={routerFuture}>
      <MobileNav />
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("MobileNav", () => {
  it("renders exactly the nav items flagged mobile", () => {
    renderNav();
    const links = screen.getAllByRole("link");
    const hrefs = links.map((l) => l.getAttribute("href")).sort();
    const expected = NAV_ITEMS.filter((i) => i.mobile)
      .map((i) => i.to)
      .sort();
    expect(hrefs).toEqual(expected);
  });

  it("exposes Dictation Studio on mobile so voice capture is reachable on phones", () => {
    renderNav();
    // Voice capture is most useful on phones; the studio must not be
    // desktop-sidebar-only. Guards against the regression where
    // /dictation-studio shipped without a mobile slot.
    const studio = screen
      .getAllByRole("link")
      .find((l) => l.getAttribute("href") === "/dictation-studio");
    expect(studio).toBeDefined();
  });

  it("keeps the mobile bar within a tappable item budget (<=6 slots)", () => {
    renderNav();
    // The bottom nav uses fixed flex slots with no overflow affordance;
    // more than 6 items makes targets too small. If this fails, add a
    // mobile "More" overflow rather than silently cramming.
    expect(screen.getAllByRole("link").length).toBeLessThanOrEqual(6);
  });
});
