import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders, seedSession, makeDevice } from "../../test-utils";
import { SessionPanel } from "./SessionPanel";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

describe("SessionPanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the not-paired hint when no device token is present", () => {
    renderWithProviders(<SessionPanel />);
    expect(screen.getByText(strings.session.notPaired)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.session.signOutButton)).not.toBeInTheDocument();
  });

  it("shows the paired state with the device name and a sign-out control", async () => {
    const user = userEvent.setup();
    seedSession({ device: makeDevice({ name: "My Laptop" }) });
    renderWithProviders(<SessionPanel />);

    expect(screen.getByText(strings.session.tokenPresent, { exact: false })).toBeInTheDocument();
    expect(screen.getByText(/My Laptop/)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.session.signOutButton));
    // Sign-out drops the device token: the panel returns to the unpaired hint.
    await waitFor(() => expect(screen.getByText(strings.session.notPaired)).toBeInTheDocument());
  });
});
