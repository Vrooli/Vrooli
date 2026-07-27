import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SigningBadge } from "./SigningBadge";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

const mocks = vi.hoisted(() => ({ getSigningReadiness: vi.fn() }));
vi.mock("../../lib/api/connect", () => ({
  signingConnectClient: { getSigningReadiness: mocks.getSigningReadiness },
}));

describe("SigningBadge", () => {
  beforeEach(() => vi.clearAllMocks());

  it("reports not-ready when readiness is absent or disabled", () => {
    renderWithProviders(<SigningBadge scenarioName="" />);
    expect(screen.getByText("Signing not ready")).toBeInTheDocument();
    expect(mocks.getSigningReadiness).not.toHaveBeenCalled();
  });

  it("reports a ready signing configuration from the generated client", async () => {
    mocks.getSigningReadiness.mockResolvedValue({ ready: true, platforms: [] });
    renderWithProviders(<SigningBadge scenarioName="canvas-lab" />);
    expect(await screen.findByText("Signing ready")).toBeInTheDocument();
    expect(mocks.getSigningReadiness).toHaveBeenCalledWith({
      scenarioName: "canvas-lab",
    });
  });

  it("reports a fetched but unavailable signing configuration", async () => {
    mocks.getSigningReadiness.mockResolvedValue({
      ready: false,
      platforms: [],
    });
    renderWithProviders(<SigningBadge scenarioName="canvas-lab" />);
    expect(await screen.findByText("Signing not ready")).toBeInTheDocument();
  });
});
