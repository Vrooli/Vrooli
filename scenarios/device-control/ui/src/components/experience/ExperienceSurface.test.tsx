import { cleanup, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { ExperienceSurface } from "./ExperienceSurface";

describe("ExperienceSurface", () => {
  it("exposes stable state metadata and live status only for live states", () => {
    for (const state of ["loading", "partial", "error"] as const) {
      renderWithProviders(
        <ExperienceSurface surfaceId={`surface-${state}`} state={state} statusMessage={`${state} status`}>
          <span>{state}</span>
        </ExperienceSurface>,
      );
      const surface = document.querySelector(`[data-experience-surface="surface-${state}"]`);
      expect(surface).not.toBeNull();
      expect(surface).toHaveAttribute("data-experience-state", state);
      if (state === "loading") {
        expect(surface).toHaveAttribute("aria-busy", "true");
      } else {
        expect(surface).not.toHaveAttribute("aria-busy");
      }
      expect(screen.getByRole("status")).toHaveTextContent(`${state} status`);
      cleanup();
    }
  });

  it("omits live status for ready, empty, and static states", () => {
    for (const state of ["ready", "empty", "static"] as const) {
      renderWithProviders(
        <ExperienceSurface surfaceId={`surface-${state}`} state={state} statusMessage="should be hidden">
          <span>{state}</span>
        </ExperienceSurface>,
      );
      const surface = document.querySelector(`[data-experience-surface="surface-${state}"]`);
      expect(surface).not.toHaveAttribute("aria-busy");
      expect(screen.queryByRole("status")).not.toBeInTheDocument();
      cleanup();
    }
  });
});
