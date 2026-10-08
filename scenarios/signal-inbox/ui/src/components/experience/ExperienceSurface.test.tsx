import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { ExperienceSurface } from "./ExperienceSurface";

describe("ExperienceSurface", () => {
  it("exposes live status and busy state for loading", () => {
    renderWithProviders(<ExperienceSurface surfaceId="signals" state="loading" statusMessage="Loading signals">Content</ExperienceSurface>);
    expect(screen.getByRole("status")).toHaveTextContent("Loading signals");
    expect(screen.getByText("Content").closest("section")).toHaveAttribute("aria-busy", "true");
  });

  it("keeps static content quiet", () => {
    renderWithProviders(<ExperienceSurface surfaceId="settings" state="static">Settings</ExperienceSurface>);
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.getByText("Settings").closest("section")).toHaveAttribute("data-experience-state", "static");
  });
});
