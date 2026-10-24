import { screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AdaptivePageHeader } from "./AdaptivePageHeader";
import { renderWithProviders } from "../test-utils";

function setMobile(matches: boolean) {
  vi.stubGlobal("matchMedia", () => ({ matches, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
}

afterEach(() => vi.unstubAllGlobals());

describe("AdaptivePageHeader", () => {
  it("uses the desktop library header on wide screens", () => {
    setMobile(false);
    renderWithProviders(<AdaptivePageHeader headingId="heading" eyebrow="Plan" title="Today" description="A useful day" />);
    expect(screen.getByRole("heading", { name: "Today" })).toBeInTheDocument();
    expect(document.querySelector(".adaptive-header-desktop")).toBeTruthy();
  });

  it("selects a distinct mobile header component on narrow screens", () => {
    setMobile(true);
    renderWithProviders(<AdaptivePageHeader headingId="heading" eyebrow="Plan" title="Today" description="A useful day" />);
    expect(document.querySelector(".adaptive-header-mobile")).toBeTruthy();
    expect(screen.getByText("A useful day")).toBeInTheDocument();
  });
});
