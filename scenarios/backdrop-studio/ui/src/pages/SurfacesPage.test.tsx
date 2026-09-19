import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { makeStudioMocks, makeSurface, renderWithProviders } from "../test-utils";
import { SurfacesPage } from "./SurfacesPage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

describe("SurfacesPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists each surface with the citation its geometry came from", async () => {
    const { listSurfaces } = await import("../api/studio");
    vi.mocked(listSurfaces).mockResolvedValue([
      makeSurface(),
      makeSurface({
        id: "play.feature-graphic",
        kind: "store",
        width: 1024,
        height: 500,
        placements: ["feature_graphic"],
        authority: "Google Play Console Help — Graphic assets",
      }),
    ]);
    renderWithProviders(<SurfacesPage />);
    expect(await screen.findByTestId("surfaces-table")).toBeInTheDocument();
    // The authority is the reason this page exists: some geometry is ours to
    // choose and some is a vendor's, and only the citation tells them apart.
    expect(screen.getByText(/Google Play Console Help/)).toBeInTheDocument();
    expect(screen.getByText("1024×500")).toBeInTheDocument();
  });

  it("announces loading before the surfaces arrive", () => {
    renderWithProviders(<SurfacesPage />);
    expect(screen.getByTestId(selectors.pages.surfaces)).toHaveAttribute(
      "data-experience-state",
      "loading",
    );
  });

  it("reports an unreadable surface registry", async () => {
    const { listSurfaces } = await import("../api/studio");
    vi.mocked(listSurfaces).mockRejectedValue(new Error("surfaces unreachable"));
    renderWithProviders(<SurfacesPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("pages.surfaces.error");
  });

  it("shows an empty state when nothing is seeded", async () => {
    const { listSurfaces } = await import("../api/studio");
    vi.mocked(listSurfaces).mockResolvedValue([]);
    renderWithProviders(<SurfacesPage />);
    expect(await screen.findByText(strings.pages.surfaces.empty)).toBeInTheDocument();
  });
});
