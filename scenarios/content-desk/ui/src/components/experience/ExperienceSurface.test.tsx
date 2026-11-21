import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ExperienceSurface } from "./ExperienceSurface";
import { renderWithProviders } from "../../test-utils";

describe("ExperienceSurface", () => {
  it("announces an actionable live state while preserving its semantic contract", () => {
    renderWithProviders(<ExperienceSurface surfaceId="draft-review" state="partial" statusMessage="Some evidence needs review"><span>Drafts</span></ExperienceSurface>);
    expect(screen.getByRole("status")).toHaveTextContent("Some evidence needs review");
    expect(screen.getByText("Drafts").closest("section")).toHaveAttribute("data-experience-state", "partial");
  });

  it("marks loading work as busy without inventing a live announcement", () => {
    renderWithProviders(<ExperienceSurface surfaceId="draft-review" state="loading"><span>Loading</span></ExperienceSurface>);
    expect(screen.getByText("Loading").closest("section")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
