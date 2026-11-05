import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { ExperienceSurface } from "./ExperienceSurface";

describe("ExperienceSurface", () => {
  it("exposes loading status to assistive technology", () => {
    renderWithProviders(<ExperienceSurface surfaceId="journal" state="loading" statusMessage="Loading memories">Timeline</ExperienceSurface>);
    expect(screen.getByRole("status")).toHaveTextContent("Loading memories");
    expect(screen.getByText("Timeline").closest("section")).toHaveAttribute("aria-busy", "true");
  });

  it("omits live status for a settled surface", () => {
    renderWithProviders(<ExperienceSurface surfaceId="journal" state="ready">Timeline</ExperienceSurface>);
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.getByText("Timeline").closest("section")).not.toHaveAttribute("aria-busy");
  });
});
