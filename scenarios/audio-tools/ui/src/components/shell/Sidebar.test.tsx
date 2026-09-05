import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { Sidebar } from "./Sidebar";
import { NAV_ITEMS } from "./nav-items";
import { strings } from "../../consts/strings";

afterEach(cleanup);

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

function renderSidebar(initialPath = "/") {
  return renderWithProviders(
    <MemoryRouter initialEntries={[initialPath]} future={routerFuture}>
      <Sidebar />
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("Sidebar", () => {
  it("renders an aside with primaryNav aria-label", () => {
    renderSidebar();
    const aside = screen.getByRole("complementary", { name: strings.shell.primaryNav });
    expect(aside).toBeInTheDocument();
  });

  it("renders all nav items as links", () => {
    renderSidebar();
    const links = screen.getAllByRole("link");
    expect(links.length).toBe(NAV_ITEMS.length);
  });

  it("nav links point to their configured paths", () => {
    renderSidebar();
    const links = screen.getAllByRole("link");
    const hrefs = links.map((l) => l.getAttribute("href")).sort();
    const expected = NAV_ITEMS.map((i) => i.to).sort();
    expect(hrefs).toEqual(expected);
  });

  it("marks the active route link with aria-selected pattern (class change)", () => {
    renderSidebar("/diagnostics");
    // The diagnostics link should have the active class containing text-app-primary
    const diagLink = screen
      .getAllByRole("link")
      .find((l) => l.getAttribute("href") === "/diagnostics");
    expect(diagLink).toBeDefined();
    expect(diagLink?.className).toContain("text-app-primary");
  });

  it("renders version tag in footer", () => {
    renderSidebar();
    expect(screen.getByText(strings.app.versionTag)).toBeInTheDocument();
  });
});
