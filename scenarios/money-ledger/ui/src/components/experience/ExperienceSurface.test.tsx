import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { createMockTransport } from "../../test-utils/mockTransport";
import { SURFACE_STATES, useSurfaceState, type SurfaceState } from "../../hooks/useSurfaceState";
import { ExperienceSurface } from "./ExperienceSurface";

function SurfaceProbe({ state }: { state: SurfaceState }) {
  const transport = createMockTransport(state);
  const surface = useSurfaceState(transport.surfaceInput);
  return <ExperienceSurface surfaceId="test" state={surface.state} statusMessage={surface.reason} data-testid="surface-test"><span>content</span></ExperienceSurface>;
}

describe("Money Ledger experience surface", () => {
  it.each(SURFACE_STATES)("reaches the %s state through mocked transport", (state) => {
    renderWithProviders(<SurfaceProbe state={state} />);
    expect(screen.getByTestId("surface-test")).toHaveAttribute("data-experience-state", state);
  });

  it.each(["success", "validation-error", "request-error"] as const)("announces %s", (state) => {
    renderWithProviders(<SurfaceProbe state={state} />);
    expect(screen.getByRole("status")).toBeVisible();
  });
});
