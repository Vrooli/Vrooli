import { screen } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1";

// Fixture children the surface is asked to wrap, named once so the queries
// match the fixture instead of restating its copy.
const readyContent = "Ready results";
const loadingContent = "Loading results";

describe("ExperienceSurface", () => {
  it("exposes the stable semantic lifecycle contract without prescribing layout", () => {
    renderWithProviders(
      <ExperienceSurface
        surfaceId="results"
        state="ready"
        className="custom-layout"
      >
        <p>{readyContent}</p>
      </ExperienceSurface>,
    );
    const surface = screen.getByText(readyContent).closest("section");
    expect(surface).toHaveAttribute("data-experience-surface", "results");
    expect(surface).toHaveAttribute(
      "data-testid",
      "experience-surface-results",
    );
    expect(surface).toHaveAttribute("data-experience-state", "ready");
    expect(surface).toHaveClass("custom-layout");
  });

  it("announces transient and failure state only when a message is declared", () => {
    renderWithProviders(
      <ExperienceSurface
        surfaceId="results"
        state="error"
        statusMessage="Results could not load"
      >
        <p>Try again</p>
      </ExperienceSurface>,
    );
    expect(
      screen.getByRole("status", { name: "Results could not load" }),
    ).toHaveAttribute("aria-live", "polite");
  });

  it("marks loading surfaces busy for assistive technology", () => {
    renderWithProviders(
      <ExperienceSurface surfaceId="results" state="loading">
        <p>{loadingContent}</p>
      </ExperienceSurface>,
    );
    expect(screen.getByText(loadingContent).closest("section")).toHaveAttribute(
      "aria-busy",
      "true",
    );
  });
});
