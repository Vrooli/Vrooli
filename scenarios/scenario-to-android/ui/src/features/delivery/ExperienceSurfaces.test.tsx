import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders, type ProviderRenderResult } from "../../test-utils";

import { DistributionSurface, DeliveryOverview, ReadinessSurface, RunReviewSurface, TargetDetailSurface } from "./ExperienceSurfaces";

let activeRender: ProviderRenderResult | undefined;

afterEach(() => {
  activeRender?.unmount();
  activeRender = undefined;
  cleanup();
  window.history.replaceState({}, "", "/");
});

function renderAt(element: React.ReactElement, path: string) {
  activeRender?.unmount();
  cleanup();
  window.history.replaceState({}, "", path);
  activeRender = renderWithProviders(element);
}

describe("Android delivery experience surfaces", () => {
  it("renders dashboard evidence grades and blocked states", () => {
    renderAt(<DeliveryOverview />, "/?fixture=green-pixel");
    expect(screen.getByTestId("dashboard-row-promotability")).toHaveTextContent("pixel-grade");
    cleanup();
    renderAt(<DeliveryOverview />, "/?fixture=partial");
    expect(screen.getByTestId("dashboard-target-disposition")).toHaveTextContent("unavailable");
  });

  it("renders independent distribution and readiness states", () => {
    renderAt(<DistributionSurface />, "/distribution?fixture=verification-pending");
    expect(screen.getAllByText("verification-pending").length).toBeGreaterThan(0);
    cleanup();
    renderAt(<DistributionSurface />, "/distribution?fixture=non-promotable-artifact");
    expect(screen.getByTestId("distribution-promotability-block")).toBeInTheDocument();
    cleanup();
    renderAt(<ReadinessSurface />, "/readiness?fixture=verification-pending");
    expect(screen.getAllByText("verification-pending").length).toBeGreaterThan(0);
    cleanup();
    renderAt(<DistributionSurface />, "/distribution?fixture=target-api-below-floor");
    expect(screen.getAllByText("blocked").length).toBeGreaterThan(0);
    cleanup();
    renderAt(<ReadinessSurface />, "/readiness");
    expect(screen.getByText("registration")).toBeInTheDocument();
  });

  it("renders run interruptions and target-specific remediation", () => {
    renderAt(<RunReviewSurface />, "/runs/lease-lost");
    expect(screen.getByTestId("run-review-lease-interruption")).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/local-emulator?fixture=missing-runtime-variant");
    expect(screen.getByText(/x86_64 simulator runtime/)).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/local-emulator?fixture=missing-toolchain");
    expect(screen.getByTestId("target-detail-missing-capability")).toHaveTextContent("Android SDK toolchain");
    expect(screen.getByRole("button", { name: "install the Android SDK toolchain" })).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/local-emulator?fixture=no-acceleration");
    expect(screen.getByTestId("target-detail-missing-capability")).toHaveTextContent("/dev/kvm hardware acceleration");
    expect(screen.getByRole("button", { name: "enable \/dev\/kvm hardware acceleration" })).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/android-physical?fixture=physical-unpaired");
    expect(screen.getByTestId("target-detail-missing-capability")).toHaveTextContent("USB debugging authorization");
    expect(screen.getByRole("button", { name: "authorize USB debugging on the phone" })).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/android-physical?fixture=physical-wireless-expired");
    expect(screen.getByTestId("target-detail-missing-capability")).toHaveTextContent("wireless debugging pairing");
    expect(screen.getByRole("button", { name: "renew wireless debugging pairing" })).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/local-native?fixture=terminal-linux");
    expect(screen.queryByTestId("target-detail-next-action")).not.toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/phone?fixture=device-leased-elsewhere");
    expect(screen.getByText(/another consumer/)).toBeInTheDocument();
    cleanup();
    renderAt(<TargetDetailSurface />, "/targets/emulator?fixture=no-gui-session");
    expect(screen.getByRole("button", { name: "log in to the GUI session" })).toBeInTheDocument();
  });
});
