import { screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ObservatoryScene } from "./ObservatoryScene";
import { renderWithProviders } from "../test-utils";
import { ThemeProvider } from "../theme/ThemeProvider";

afterEach(() => vi.unstubAllGlobals());

describe("ObservatoryScene", () => {
  it("renders the shared scene layers for an interior surface", () => {
    vi.stubGlobal("matchMedia", () => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    renderWithProviders(<ObservatoryScene kind="focus"><p>Focus content</p></ObservatoryScene>);
    expect(screen.getByText("Focus content")).toBeInTheDocument();
    expect(document.querySelector(".observatory-sky-day")).toBeTruthy();
    expect(document.querySelector(".observatory-sky-night")).toBeTruthy();
    expect(document.querySelector(".scene-comets")).toBeTruthy();
    expect(document.querySelector(".scene-plate")).toBeTruthy();
  });

  it("selects mobile composition and renders the exterior world variants", () => {
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    renderWithProviders(<><ObservatoryScene kind="plan"><span>Plan</span></ObservatoryScene><ObservatoryScene kind="goals"><span>Goals</span></ObservatoryScene><ObservatoryScene kind="review"><span>Review</span></ObservatoryScene><ObservatoryScene kind="settings"><span>Settings</span></ObservatoryScene></>);
    expect(document.querySelectorAll(".scene-mobile")).toHaveLength(4);
    expect(screen.getByText("Plan")).toBeInTheDocument();
    expect(screen.getByText("Goals")).toBeInTheDocument();
  });

  it("keeps the scene appearance aligned with the resolved Night theme", () => {
    renderWithProviders(<ThemeProvider initialChoice="night"><ObservatoryScene kind="review"><span>Night review</span></ObservatoryScene></ThemeProvider>);
    expect(document.querySelector(".appearance-night")).toBeTruthy();
  });
});
