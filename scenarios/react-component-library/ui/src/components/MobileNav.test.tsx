import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { MobileNav } from "./MobileNav";

describe("MobileNav", () => {
  afterEach(() => cleanup());

  it("renders the four primary destinations", () => {
    renderWithProviders(<MobileNav />);
    expect(screen.getByTestId("mobile-nav")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-nav-dashboard")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("mobile-nav-components")).toHaveAttribute("href", "/components");
    expect(screen.getByTestId("mobile-nav-adoptions")).toHaveAttribute("href", "/adoptions");
    expect(screen.getByTestId("mobile-nav-settings")).toHaveAttribute("href", "/settings");
  });

  it("marks the current route as active", () => {
    renderWithProviders(<MobileNav />, { routerEntries: ["/components/cmp-1"] });
    expect(screen.getByTestId("mobile-nav-components")).toHaveAttribute("aria-current", "page");
  });
});
