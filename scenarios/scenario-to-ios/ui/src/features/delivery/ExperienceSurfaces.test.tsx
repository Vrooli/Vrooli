import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";

import { DistributionSurface, DeliveryOverview, ReadinessSurface, RunReviewSurface, TargetDetailSurface } from "./ExperienceSurfaces";

afterEach(() => { cleanup(); window.history.replaceState({}, "", "/"); });

function renderAt(element: React.ReactElement, path: string) {
  window.history.replaceState({}, "", path);
  renderWithProviders(element);
}

describe("iOS delivery experience surfaces", () => {
  it("renders terminal, validate-only, and evidence-grade dashboard states", () => {
    renderAt(<DeliveryOverview />, "/?fixture=green-pixel");
    expect(screen.getByTestId("dashboard-row-promotability")).toHaveTextContent(strings.experience.pixelGradeMirror);
    cleanup();
    renderAt(<DeliveryOverview />, "/?fixture=terminal-and-blocked");
    expect(screen.getByTestId("dashboard-target-disposition")).toHaveTextContent(strings.experience.unsupported);
    cleanup();
    renderAt(<DeliveryOverview />, "/?fixture=no-mac-node");
    expect(screen.getByTestId("dashboard-readiness-summary")).toHaveTextContent(strings.experience.blockedOnMacOSBridge);
    cleanup();
    renderAt(<DeliveryOverview />, "/?fixture=mirror-passed");
    expect(screen.getByTestId("dashboard-row-promotability")).toHaveTextContent(strings.experience.pixelGradeMirror);
  });

  it("renders Apple distribution and readiness blockers verbatim", () => {
    renderAt(<DistributionSurface />, "/distribution?fixture=verification-pending");
    expect(screen.getAllByText(strings.experience.waitingOnApple).length).toBeGreaterThan(0);
    cleanup();
    renderAt(<ReadinessSurface />, "/readiness?fixture=verification-pending");
    expect(screen.getByTestId("readiness-mirror-path-note")).toBeInTheDocument();
    cleanup();
    renderAt(<DistributionSurface />, "/distribution?fixture=non-promotable-artifact");
    expect(screen.getByTestId("distribution-promotability-block")).toBeInTheDocument();
    cleanup();
    renderAt(<ReadinessSurface />, "/readiness");
    expect(screen.getByText(strings.experience.appleID)).toBeInTheDocument();
    cleanup();
    renderAt(<DistributionSurface />, "/distribution?fixture=no-mac-node");
    expect(screen.getAllByText(strings.experience.blockedNoMacNode).length).toBeGreaterThan(0);
    cleanup();
    renderAt(<DistributionSurface />, "/distribution?fixture=no-enrollment");
    expect(screen.getAllByText(strings.experience.enrollDeveloperProgram).length).toBeGreaterThan(0);
    cleanup();
    renderAt(<DistributionSurface />, "/distribution?fixture=testflight-ready");
    expect(screen.getAllByTestId(selectors.distribution.channelAvailability).some((node) => node.textContent.includes(`${strings.experience.testFlight}: ${strings.experience.available}`))).toBe(true);
    cleanup();
    renderAt(<DistributionSurface />, "/distribution?fixture=review-pending");
    expect(screen.getAllByText(strings.experience.waitingOnApple).length).toBeGreaterThan(0);
    cleanup();
    renderAt(<ReadinessSurface />, "/readiness?fixture=no-mac-node");
    expect(screen.getAllByText(strings.experience.blockedNoMacNode).length).toBeGreaterThan(0);
    cleanup();
    renderAt(<ReadinessSurface />, "/readiness?fixture=nothing-started");
    expect(screen.getAllByText(strings.experience.blockedOnAppleID).length).toBeGreaterThan(0);
    cleanup();
    renderAt(<ReadinessSurface />, "/readiness?fixture=enrollment-pending");
    expect(screen.getAllByText(strings.experience.waitingOnApple).length).toBeGreaterThan(0);
  });

  it("distinguishes non-promotable runs and terminal targets", () => {
    renderAt(<RunReviewSurface />, "/runs/mirror-run");
    expect(screen.getByTestId("run-review-promotability-notice")).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/local-native?fixture=terminal-linux");
    expect(screen.queryByTestId("target-detail-next-action")).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.targetDetail.missingCapability)).toHaveTextContent(strings.experience.appleToolchain);
    cleanup();
    renderAt(<RunReviewSurface />, "/runs/partial");
    expect(screen.getByText(strings.experience.wkwebviewUnavailable)).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/iphone?fixture=device-leased-elsewhere");
    expect(screen.getByText(strings.experience.leaseHolderOtherConsumer)).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/iphone?fixture=no-gui-session");
    expect(screen.getByTestId(selectors.targetDetail.missingCapability)).toHaveTextContent(strings.experience.loggedInGuiSession);
  });
});
