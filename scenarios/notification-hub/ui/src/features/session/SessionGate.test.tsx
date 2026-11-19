import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";

import { identityClient } from "../../api/notifications";
import { currentSessionGeneration } from "../../api/sessionGeneration";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { SignInForm, SessionPanel } from "./SessionGate";
import { SignedOutCard } from "./SignedOutCard";
import { sessionQueryKey, useSessionExpiryBoundary } from "./useSession";
import { setLocale } from "../../i18n";
import { en } from "../../i18n/catalog";

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
  it("does not offer sign-in credentials in an insecure context", () => {
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: false });
    renderWithProviders(<SignInForm />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.session.password)).not.toBeInTheDocument();
    expect(identityClient.login).not.toHaveBeenCalled();
  });
  beforeEach(async () => setLocale("en"));
  afterEach(() => vi.clearAllMocks());

  it("trims the login email, clears the password, and invalidates the session after success", async () => {
    const user = userEvent.setup();
    const previousGeneration = currentSessionGeneration();
    vi.mocked(identityClient.login).mockResolvedValue({} as never);
    renderWithProviders(<SignInForm />);

    await user.type(screen.getByTestId(selectors.session.email), " owner@example.test ");
    await user.type(screen.getByTestId(selectors.session.password), "secret");
    await user.click(screen.getByTestId(selectors.session.submit));

    await waitFor(() => expect(identityClient.login).toHaveBeenCalledWith({ email: "owner@example.test", password: "secret" }));
    await waitFor(() => expect(screen.getByTestId(selectors.session.password)).toHaveValue(""));
    expect(currentSessionGeneration()).toBeGreaterThan(previousGeneration);
  });

  it("completes an MFA challenge before treating login as successful", async () => {
    const user = userEvent.setup();
    vi.mocked(identityClient.login).mockResolvedValueOnce({ mfaRequired: true, mfaChallenge: "challenge" } as never).mockResolvedValueOnce({} as never);
    renderWithProviders(<SignInForm />);
    await user.type(screen.getByTestId(selectors.session.email), "owner@example.test");
    await user.type(screen.getByTestId(selectors.session.password), "secret");
    await user.click(screen.getByTestId(selectors.session.submit));
    await user.type(await screen.findByLabelText(en.session.totpCode), "123456");
    await user.click(screen.getByTestId(selectors.session.submit));
    await waitFor(() => expect(identityClient.login).toHaveBeenLastCalledWith({ email: "owner@example.test", password: "secret", mfaChallenge: "challenge", totpCode: "123456" }));
    await waitFor(() => expect(screen.getByTestId(selectors.session.password)).toHaveValue(""));
  });

  it("requests a fresh MFA challenge after the previous challenge is rejected", async () => {
    const user = userEvent.setup();
    vi.mocked(identityClient.login)
      .mockResolvedValueOnce({ mfaRequired: true, mfaChallenge: "expired" } as never)
      .mockRejectedValueOnce(new ConnectError("denied", Code.Unauthenticated))
      .mockResolvedValueOnce({ mfaRequired: true, mfaChallenge: "fresh" } as never);
    renderWithProviders(<SignInForm />);
    await user.type(screen.getByTestId(selectors.session.email), "owner@example.test");
    await user.type(screen.getByTestId(selectors.session.password), "secret");
    await user.click(screen.getByTestId(selectors.session.submit));
    await user.type(await screen.findByLabelText(en.session.totpCode), "123456");
    await user.click(screen.getByTestId(selectors.session.submit));
    await screen.findByRole("alert");
    expect(screen.queryByLabelText(en.session.totpCode)).not.toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.session.submit));
    await waitFor(() => expect(identityClient.login).toHaveBeenLastCalledWith({ email: "owner@example.test", password: "secret" }));
    expect(await screen.findByLabelText(en.session.totpCode)).toHaveValue("");
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

  it("does not mistake an unavailable session check for a signed-out owner", async () => {
    vi.mocked(identityClient.getSession).mockRejectedValue(new ConnectError("service unavailable", Code.Unavailable));
    renderWithProviders(<SessionPanel />);
    expect(await screen.findByText(en.errors.unavailable)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.session.form)).not.toBeInTheDocument();
  });

  it("keeps read-only notification browsing available from an owner-only screen", () => {
    renderWithProviders(<SignedOutCard />);
    expect(screen.getByRole("region", { name: en.session.title })).toHaveTextContent(en.session.readOnlyBrowse);
    expect(screen.getByRole("link", { name: en.layout.nav.dashboard })).toHaveAttribute("href", "/updates");
    expect(screen.getByRole("link", { name: en.pages.delivery.title })).toHaveAttribute("href", "/delivery");
  });

  it("clears recipient data and ends the Hub session immediately after cookie sign-out", async () => {
    const user = userEvent.setup();
    const previousGeneration = currentSessionGeneration();
    vi.mocked(identityClient.getSession)
      .mockResolvedValueOnce({ signedIn: true, subject: "owner-1", email: "owner@example.test" } as never)
      .mockImplementation(() => new Promise(() => undefined));
    vi.mocked(identityClient.logout).mockResolvedValue({} as never);
    const { queryClient } = renderWithProviders(<SessionPanel />);
    await waitFor(() => expect(queryClient.getQueryData(sessionQueryKey)).toMatchObject({ signedIn: true, subject: "owner-1" }));
    queryClient.setQueryData(["asks", "owner-1", "history"], { pages: [{ asks: [{ id: "private-ask" }] }] });
    queryClient.setQueryData(["ask", "private-ask", "owner-1"], { ask: { id: "private-ask" } });
    queryClient.setQueryData(["ask-explainer", "owner-1", "private-ask", "fingerprint"], { html: "private attachment rendering" });
    queryClient.setQueryData(["quiet-windows"], { windows: [{ id: "owner-private-window" }] });
    queryClient.setQueryData(["escalation-chain"], { steps: [{ channel: "owner-private-channel" }] });
    queryClient.setQueryData(["recipient-channel-addresses"], { addresses: [{ address: "private@example.test" }] });
    queryClient.setQueryData(["device-apps", "owner-1"], { apps: [{ id: "private-app" }] });
    const signOut = await screen.findByTestId(selectors.session.signOut);
    await user.click(signOut);
    expect(identityClient.logout).not.toHaveBeenCalled();
    expect(await screen.findByRole("alertdialog", { name: "Sign out of this Hub session?" })).toBeInTheDocument();
    expect(screen.getByText(en.session.signOutOtherAuthTitle)).toBeInTheDocument();
    expect(screen.getByText(en.session.signOutDescription)).toBeInTheDocument();
    expect(screen.getByText(/does not revoke issued tokens or sign you out of Cloudflare Access/)).toBeInTheDocument();
    const confirmation = screen.getByRole("alertdialog");
    await user.click(within(confirmation).getByRole("button", { name: "Sign out of Hub" }));
    await waitFor(() => expect(identityClient.logout).toHaveBeenCalledTimes(1));
    expect(identityClient.logout).toHaveBeenCalledWith({});
    expect(await screen.findByText(en.session.hubSessionInactive)).toBeInTheDocument();
    expect(identityClient.getSession).toHaveBeenCalledTimes(1);
    expect(currentSessionGeneration()).toBeGreaterThan(previousGeneration);
    expect(queryClient.getQueryData(sessionQueryKey)).toMatchObject({ signedIn: false, subject: "", email: "" });
    expect(queryClient.getQueryCache().findAll({ queryKey: ["asks"] })).toHaveLength(0);
    expect(queryClient.getQueryCache().findAll({ queryKey: ["ask", "private-ask", "owner-1"] })).toHaveLength(0);
    expect(queryClient.getQueryCache().findAll({ queryKey: ["ask-explainer", "owner-1"] })).toHaveLength(0);
    expect(queryClient.getQueryData(["quiet-windows"])).toBeUndefined();
    expect(queryClient.getQueryData(["escalation-chain"])).toBeUndefined();
    expect(queryClient.getQueryData(["recipient-channel-addresses"])).toBeUndefined();
    expect(queryClient.getQueryCache().findAll({ queryKey: ["device-apps"] })).toHaveLength(0);
    expect(screen.queryByTestId(selectors.session.signedIn)).not.toBeInTheDocument();
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("lets the owner keep the Hub session without sending a logout request", async () => {
    const user = userEvent.setup();
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, email: "owner@example.test" } as never);
    renderWithProviders(<SessionPanel />);

    await user.click(await screen.findByTestId(selectors.session.signOut));
    await user.click(screen.getByRole("button", { name: en.session.keepSession }));

    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
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
