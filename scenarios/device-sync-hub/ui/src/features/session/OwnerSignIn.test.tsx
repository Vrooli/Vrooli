import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders, seedSession } from "../../test-utils";
import { selectors } from "../../consts/selectors";

// OwnerSignIn renders OwnerLoginForm (same-origin IdentityService) when not
// signed in; stub the client so the test never reaches the network.
vi.mock("../../api/identity", () => ({ identityClient: { login: vi.fn(), register: vi.fn() } }));

import { OwnerSignIn } from "./OwnerSignIn";

describe("OwnerSignIn", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the same-origin sign-in form when no owner token is present", () => {
    renderWithProviders(<OwnerSignIn />);
    expect(screen.getByTestId(selectors.login.form)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.owner.status)).not.toBeInTheDocument();
  });

  it("shows the signed-in status + sign out when an owner token is present", () => {
    seedSession({ ownerToken: "existing-jwt" });
    renderWithProviders(<OwnerSignIn />);
    expect(screen.getByTestId(selectors.owner.status)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.owner.signOutButton)).toBeInTheDocument();
  });

  it("clears the owner token and returns to the sign-in form", async () => {
    const user = userEvent.setup();
    seedSession({ ownerToken: "existing-jwt" });
    renderWithProviders(<OwnerSignIn />);

    expect(screen.getByTestId(selectors.owner.status)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.owner.signOutButton));

    await waitFor(() => expect(screen.getByTestId(selectors.login.form)).toBeInTheDocument());
  });
});
