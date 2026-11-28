import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";

import { CaptureGalleryPage } from "./CaptureGalleryPage";

describe("CaptureGalleryPage", () => {
  it("renders capture cards with violation overlays", () => {
    renderWithProviders(<CaptureGalleryPage />);

    expect(screen.getByTestId(selectors.captures.gallery)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.captures.captureCard({ captureId: "experience-manager-fleet-mobile" })),
    ).toBeInTheDocument();
    expect(screen.getByTestId(selectors.captures.overlay({ index: 0 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.captures.overlayList)).toBeInTheDocument();
  });

  it("filters captures by scenario", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CaptureGalleryPage />);

    await user.selectOptions(
      screen.getByTestId(selectors.captures.scenarioSelect),
      "ui-health",
    );

    expect(
      screen.queryByTestId(selectors.captures.captureCard({ captureId: "experience-manager-fleet-mobile" })),
    ).not.toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.captures.captureCard({ captureId: "ui-health-dashboard-desktop" })),
    ).toBeInTheDocument();
  });

  it("shows a clean state for captures without violations", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CaptureGalleryPage />);

    await user.selectOptions(
      screen.getByTestId(selectors.captures.scenarioSelect),
      "ui-health",
    );

    expect(screen.getByText(strings.pages.captures.violations.empty)).toBeInTheDocument();
  });
});
