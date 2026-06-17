import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders, seedSession } from "../../test-utils";
import { OwnerSignIn } from "./OwnerSignIn";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

describe("OwnerSignIn", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("validates a missing token and shows an error without persisting", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OwnerSignIn />);

    await user.click(screen.getByTestId(selectors.owner.signInButton));

    expect(screen.getByText(strings.owner.missingToken)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.owner.status)).not.toBeInTheDocument();
  });

  it("stores a pasted token (trimmed) and flips to the signed-in state", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OwnerSignIn />);

    await user.type(screen.getByTestId(selectors.owner.tokenInput), "  jwt-token  ");
    await user.click(screen.getByTestId(selectors.owner.signInButton));

    await waitFor(() => expect(screen.getByTestId(selectors.owner.status)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.owner.signOutButton)).toBeInTheDocument();
  });

  it("clears the owner token from the signed-in state", async () => {
    const user = userEvent.setup();
    seedSession({ ownerToken: "existing-jwt" });
    renderWithProviders(<OwnerSignIn />);

    expect(screen.getByTestId(selectors.owner.status)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.owner.signOutButton));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.owner.tokenInput)).toBeInTheDocument(),
    );
  });
});
