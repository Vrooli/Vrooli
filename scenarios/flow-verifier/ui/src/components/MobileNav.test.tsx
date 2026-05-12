import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { MobileNav } from "./MobileNav";

describe("MobileNav", () => {
  afterEach(() => cleanup());

  it("renders the three primary destinations", () => {
    renderWithProviders(<MobileNav />);
    expect(screen.getByTestId("mobile-nav")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-nav-dashboard")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("mobile-nav-flows")).toHaveAttribute("href", "/flows");
    expect(screen.getByTestId("mobile-nav-settings")).toHaveAttribute("href", "/settings");
  });
});
