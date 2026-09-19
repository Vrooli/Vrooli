import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders, expectNoA11yViolations } from "../../test-utils";

// OwnerLoginForm posts to the hub's own same-origin IdentityService; stub the
// client so importing the onboarding tree never reaches the network.
vi.mock("../../api/identity", () => ({ identityClient: { login: vi.fn(), register: vi.fn() } }));

import { OnboardingScreen } from "./OnboardingScreen";
import { selectors } from "../../consts/selectors";
import { saveSession, clearSession } from "../session/store";

describe("OnboardingScreen", () => {
  afterEach(() => {
    cleanup();
    clearSession();
    vi.clearAllMocks();
  });

  it("shows the welcome chooser when nobody is signed in", () => {
    renderWithProviders(<OnboardingScreen />);
    expect(screen.getByTestId(selectors.onboarding.screen)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.onboarding.setupChoice)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.onboarding.joinChoice)).toBeInTheDocument();
  });

  it("routes the setup choice to the owner login form", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardingScreen />);
    await user.click(screen.getByTestId(selectors.onboarding.setupChoice));
    expect(screen.getByTestId(selectors.login.form)).toBeInTheDocument();
  });

  it("routes the join choice to the join-hub form", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardingScreen />);
    await user.click(screen.getByTestId(selectors.onboarding.joinChoice));
    expect(screen.getByTestId(selectors.join.screen)).toBeInTheDocument();
  });

  it("jumps straight to device setup when an owner is already signed in", () => {
    saveSession({ deviceToken: null, device: null, ownerToken: "owner-jwt", ownerEmail: "o@e.com" });
    renderWithProviders(<OnboardingScreen />);
    expect(screen.getByTestId(selectors.setupDevice.panel)).toBeInTheDocument();
  });

  it("the welcome chooser has no a11y violations", async () => {
    const { container } = renderWithProviders(<OnboardingScreen />);
    await expectNoA11yViolations(container);
  });
});
