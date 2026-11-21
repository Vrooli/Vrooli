import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { WriteRefusal } from "@vrooli/proto-types/money-ledger/v1/access/access_pb";

import { fetchWriteAccess, signIn, signOut } from "../../api/access";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { AccessNotice } from "./AccessNotice";
import { AccessPanel } from "./AccessPanel";
import { WriteFieldset } from "./WriteFieldset";
import { makeSignInResponse, makeWriteAccess } from "./mocks/accessMocks";

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.mocked(fetchWriteAccess).mockResolvedValue(makeWriteAccess());
});

const viewingOnly = (refusal: WriteRefusal, signedInAs = "") => {
  vi.mocked(fetchWriteAccess).mockResolvedValue(makeWriteAccess({ canWrite: false, refusal, signedInAs, browserSession: Boolean(signedInAs) }));
};

async function openSignIn(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByTestId(`${selectors.access.notice}-sign-in`));
  return screen.findByTestId(selectors.access.signInForm);
}

describe("write access notice", () => {
  it("says nothing while the owner may write", async () => {
    renderWithProviders(<AccessNotice />);
    await waitFor(() => expect(fetchWriteAccess).toHaveBeenCalled());
    expect(screen.queryByTestId(selectors.access.notice)).not.toBeInTheDocument();
  });

  it("explains a signed-out browser is viewing only and offers the sign-in", async () => {
    viewingOnly(WriteRefusal.SIGNED_OUT);
    renderWithProviders(<AccessNotice />);
    const notice = await screen.findByTestId(selectors.access.notice);
    expect(notice).toHaveTextContent(strings.access.viewingOnly);
    expect(notice).toHaveTextContent(strings.access.refusal.signedOut);
    expect(notice).toHaveTextContent(strings.access.cliHint);
    expect(within(notice).getByTestId(`${selectors.access.notice}-sign-in`)).toBeInTheDocument();
  });

  it("offers no sign-in when signing in cannot help", async () => {
    viewingOnly(WriteRefusal.AGENT);
    renderWithProviders(<AccessNotice />);
    const notice = await screen.findByTestId(selectors.access.notice);
    expect(notice).toHaveTextContent(strings.access.refusal.agent);
    expect(within(notice).queryByTestId(`${selectors.access.notice}-sign-in`)).not.toBeInTheDocument();
  });
});

describe("browser sign-in", () => {
  it("signs the owner in and closes once the API admits them", async () => {
    viewingOnly(WriteRefusal.SIGNED_OUT);
    const user = userEvent.setup();
    renderWithProviders(<AccessNotice />);
    await openSignIn(user);
    await user.type(screen.getByTestId(selectors.access.email), " owner@example.test ");
    await user.type(screen.getByTestId(`${selectors.access.password}-input`), "secret");
    vi.mocked(fetchWriteAccess).mockResolvedValue(makeWriteAccess());
    await user.click(screen.getByTestId(selectors.access.signInSubmit));
    await waitFor(() => expect(screen.queryByTestId(selectors.access.signInForm)).not.toBeInTheDocument());
    expect(signIn).toHaveBeenCalledWith({ email: "owner@example.test", password: "secret", mfaChallenge: "", totpCode: "", recoveryCode: "" });
    await waitFor(() => expect(screen.queryByTestId(selectors.access.notice)).not.toBeInTheDocument());
  });

  it("asks for the second factor and sends a recovery code as one", async () => {
    viewingOnly(WriteRefusal.SIGNED_OUT);
    vi.mocked(signIn).mockResolvedValueOnce(makeSignInResponse({ mfaRequired: true, mfaChallenge: "challenge-1", access: undefined }));
    const user = userEvent.setup();
    renderWithProviders(<AccessNotice />);
    await openSignIn(user);
    await user.type(screen.getByTestId(selectors.access.email), "owner@example.test");
    await user.type(screen.getByTestId(`${selectors.access.password}-input`), "secret");
    await user.click(screen.getByTestId(selectors.access.signInSubmit));
    const code = await screen.findByTestId(selectors.access.code);
    await waitFor(() => expect(code).toHaveFocus());
    await user.type(code, "abcd-efgh");
    await user.click(screen.getByTestId(selectors.access.signInSubmit));
    await waitFor(() => expect(signIn).toHaveBeenLastCalledWith({ email: "owner@example.test", password: "secret", mfaChallenge: "challenge-1", totpCode: "", recoveryCode: "abcd-efgh" }));
  });

  it("keeps the dialog open to say a signed-in person is not the owner", async () => {
    viewingOnly(WriteRefusal.SIGNED_OUT);
    vi.mocked(signIn).mockResolvedValueOnce(makeSignInResponse({ access: makeWriteAccess({ canWrite: false, refusal: WriteRefusal.NOT_OWNER, signedInAs: "someone@example.test" }) }));
    const user = userEvent.setup();
    renderWithProviders(<AccessNotice />);
    await openSignIn(user);
    await user.type(screen.getByTestId(selectors.access.email), "someone@example.test");
    await user.type(screen.getByTestId(`${selectors.access.password}-input`), "secret");
    await user.click(screen.getByTestId(selectors.access.signInSubmit));
    expect(await screen.findByTestId(selectors.access.signInRefusal)).toHaveTextContent(strings.access.refusal.notOwner);
    expect(screen.getByTestId(selectors.access.signInForm)).toBeInTheDocument();
  });

  it.each([
    [Code.Unauthenticated, strings.access.signIn.invalid],
    [Code.ResourceExhausted, strings.access.signIn.throttled],
    [Code.Unavailable, strings.access.signIn.unavailable],
  ])("names a refused sign-in (%s)", async (code, message) => {
    viewingOnly(WriteRefusal.SIGNED_OUT);
    vi.mocked(signIn).mockRejectedValueOnce(new ConnectError("refused", code));
    const user = userEvent.setup();
    renderWithProviders(<AccessNotice />);
    await openSignIn(user);
    await user.type(screen.getByTestId(selectors.access.email), "owner@example.test");
    await user.type(screen.getByTestId(`${selectors.access.password}-input`), "wrong");
    await user.click(screen.getByTestId(selectors.access.signInSubmit));
    expect(await screen.findByTestId(selectors.access.signInError)).toHaveTextContent(message);
  });
});

describe("access settings", () => {
  it("names the signed-in writer and signs out", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AccessPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.access.status)).toHaveTextContent(strings.access.writer));
    vi.mocked(fetchWriteAccess).mockResolvedValue(makeWriteAccess({ canWrite: false, refusal: WriteRefusal.SIGNED_OUT }));
    await user.click(screen.getByTestId(selectors.access.signOut));
    expect(signOut).toHaveBeenCalled();
    expect(await screen.findByText(strings.access.signedOut)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId(selectors.access.status)).toHaveTextContent(strings.access.refusal.signedOut));
    expect(screen.queryByTestId(selectors.access.signOut)).not.toBeInTheDocument();
  });

  it("says when the tunnel sign-in is what admits the owner", async () => {
    vi.mocked(fetchWriteAccess).mockResolvedValue(makeWriteAccess({ browserSession: false }));
    renderWithProviders(<AccessPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.access.status)).toHaveTextContent(strings.access.tunnelWriter));
    expect(screen.queryByTestId(selectors.access.signOut)).not.toBeInTheDocument();
  });
});

describe("write controls", () => {
  it("stay disabled until the API says this browser may write", async () => {
    viewingOnly(WriteRefusal.SIGNED_OUT);
    renderWithProviders(<WriteFieldset><input data-testid="amount" /><button type="submit" data-testid="save" /></WriteFieldset>);
    await waitFor(() => expect(fetchWriteAccess).toHaveBeenCalled());
    expect(screen.getByTestId("save")).toBeDisabled();
    expect(screen.getByTestId("amount")).toBeDisabled();
  });

  it("are enabled for the owner", async () => {
    renderWithProviders(<WriteFieldset><button type="submit" data-testid="save" /></WriteFieldset>);
    await waitFor(() => expect(screen.getByTestId("save")).toBeEnabled());
  });
});
