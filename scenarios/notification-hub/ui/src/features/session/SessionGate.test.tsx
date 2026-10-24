import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { identityClient } from "../../api/notifications";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { SignInForm, SessionPanel } from "./SessionGate";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(), login: vi.fn(), logout: vi.fn() },
}));

describe("SessionGate", () => {
  afterEach(() => vi.clearAllMocks());

  it("trims the login email, clears the password, and invalidates the session after success", async () => {
    const user = userEvent.setup();
    vi.mocked(identityClient.login).mockResolvedValue({} as never);
    renderWithProviders(<SignInForm />);

    await user.type(screen.getByTestId(selectors.session.email), " owner@example.test ");
    await user.type(screen.getByTestId(selectors.session.password), "secret");
    await user.click(screen.getByTestId(selectors.session.submit));

    await waitFor(() => expect(identityClient.login).toHaveBeenCalledWith({ email: "owner@example.test", password: "secret" }));
    await waitFor(() => expect(screen.getByTestId(selectors.session.password)).toHaveValue(""));
  });

  it("does not submit incomplete credentials and explains rejected credentials", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SignInForm />);
    await user.click(screen.getByTestId(selectors.session.submit));
    expect(identityClient.login).not.toHaveBeenCalled();

    vi.mocked(identityClient.login).mockRejectedValue(new ConnectError("denied", Code.Unauthenticated));
    await user.type(screen.getByTestId(selectors.session.email), "owner@example.test");
    await user.type(screen.getByTestId(selectors.session.password), "bad");
    await user.click(screen.getByTestId(selectors.session.submit));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });

  it("shows the sign-in form for a signed-out owner", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    renderWithProviders(<SessionPanel />);
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
  });

  it("settles sign-out by refreshing session state", async () => {
    const user = userEvent.setup();
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, email: "owner@example.test" } as never);
    vi.mocked(identityClient.logout).mockResolvedValue({} as never);
    renderWithProviders(<SessionPanel />);
    const signOut = await screen.findByTestId(selectors.session.signOut);
    await user.click(signOut);
    await waitFor(() => expect(identityClient.logout).toHaveBeenCalled());
  });
});
