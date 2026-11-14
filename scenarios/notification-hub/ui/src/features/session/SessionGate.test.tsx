import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";

import { identityClient } from "../../api/notifications";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { sessionQueryKey, SignInForm, SessionPanel, useSessionExpiryBoundary } from "./SessionGate";
import { setLocale } from "../../i18n";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(), login: vi.fn(), logout: vi.fn() },
}));

function SessionExpiryProbe() {
  useSessionExpiryBoundary();
  const queryClient = useQueryClient();
  const [snapshot, setSnapshot] = useState("");

  useEffect(() => {
    queryClient.setQueryData(sessionQueryKey, {
      signedIn: true,
      subject: "owner-private",
      email: "owner@example.test",
    });
    window.dispatchEvent(new Event("notification-hub:session-expired"));
    setSnapshot(JSON.stringify(queryClient.getQueryData(sessionQueryKey)));
  }, [queryClient]);

  return <output data-testid="session-expiry-snapshot">{snapshot}</output>;
}

describe("SessionGate", () => {
  beforeEach(async () => setLocale("en"));
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

  it("clears only the Hub session after cookie sign-out and refreshes verified identity", async () => {
    const user = userEvent.setup();
    vi.mocked(identityClient.getSession)
      .mockResolvedValueOnce({ signedIn: true, subject: "owner-1", email: "owner@example.test" } as never)
      .mockResolvedValueOnce({ signedIn: false, subject: "", email: "" } as never);
    vi.mocked(identityClient.logout).mockResolvedValue({} as never);
    renderWithProviders(<SessionPanel />);
    const signOut = await screen.findByTestId(selectors.session.signOut);
    await user.click(signOut);
    expect(identityClient.logout).not.toHaveBeenCalled();
    expect(await screen.findByRole("alertdialog", { name: "Sign out of this Hub session?" })).toBeInTheDocument();
    expect(screen.getByText("Chimesail · Hub session")).toBeInTheDocument();
    expect(screen.getByText("Other sign-ins stay separate")).toBeInTheDocument();
    expect(screen.getByText("This clears Hub cookies in this browser.")).toBeInTheDocument();
    expect(screen.getByText(/does not revoke issued tokens or sign you out of Cloudflare Access/)).toBeInTheDocument();
    const confirmation = screen.getByRole("alertdialog");
    await user.click(within(confirmation).getByRole("button", { name: "Sign out of Hub" }));
    await waitFor(() => expect(identityClient.logout).toHaveBeenCalledTimes(1));
    expect(identityClient.logout).toHaveBeenCalledWith({});
    expect(await screen.findByText("No Hub session is active")).toBeInTheDocument();
    await waitFor(() => expect(identityClient.getSession).toHaveBeenCalledTimes(2));
    expect(screen.queryByTestId(selectors.session.signedIn)).not.toBeInTheDocument();
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("lets the owner keep the Hub session without sending a logout request", async () => {
    const user = userEvent.setup();
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, email: "owner@example.test" } as never);
    renderWithProviders(<SessionPanel />);

    await user.click(await screen.findByTestId(selectors.session.signOut));
    await user.click(screen.getByRole("button", { name: "Keep session" }));

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(identityClient.logout).not.toHaveBeenCalled();
  });

  it("localizes the sign-out confirmation in Arabic and keeps the page right-to-left", async () => {
    const user = userEvent.setup();
    await setLocale("ar");
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, email: "owner@example.test" } as never);
    renderWithProviders(<SessionPanel />);

    await user.click(await screen.findByTestId(selectors.session.signOut));
    expect(await screen.findByRole("alertdialog", { name: "هل تريد تسجيل الخروج من جلسة Hub هذه؟" })).toBeInTheDocument();
    expect(screen.getByText(/لن يلغي الرموز الصادرة أو يسجّل خروجك من Cloudflare Access/)).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute("dir", "rtl");
    await setLocale("en");
  });

  it("clears the verified identity when the shared session expires", async () => {
    renderWithProviders(<SessionExpiryProbe />);

    const snapshot = await screen.findByTestId("session-expiry-snapshot");
    await waitFor(() => expect(snapshot).toHaveTextContent('"signedIn":false'));
    expect(snapshot).toHaveTextContent('"subject":""');
    expect(snapshot).toHaveTextContent('"email":""');
    expect(snapshot).not.toHaveTextContent("owner-private");
    expect(snapshot).not.toHaveTextContent("owner@example.test");
  });
});
