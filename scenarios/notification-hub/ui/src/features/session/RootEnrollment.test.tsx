import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { en } from "../../i18n/catalog";
import { setLocale } from "../../i18n";
import { RootEnrollment } from "./RootEnrollment";

describe("owner account connection", () => {
  beforeEach(async () => { await setLocale("en"); Object.defineProperty(window, "isSecureContext", { configurable: true, value: true }); });
  afterEach(() => vi.unstubAllGlobals());
  it("refuses to display credentials in an insecure context", () => {
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: false });
    renderWithProviders(<RootEnrollment />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.queryByLabelText(en.session.password)).not.toBeInTheDocument();
  });
  it("confirms before disconnecting all accounts and reports success", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    renderWithProviders(<RootEnrollment />);
    await user.click(screen.getByRole("button", { name: en.session.revokeAction }));
    expect(fetchMock).not.toHaveBeenCalled();
    const dialog = screen.getByRole("alertdialog");
    expect(dialog).toHaveTextContent(en.session.revokeDescription);
    await user.click(within(dialog).getByRole("button", { name: en.session.revokeAction }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent(en.session.revokeSuccess));
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/identity/root-enrollment", { method: "DELETE", credentials: "same-origin" });
  });
  it("submits a recovery code and starts a fresh challenge after rejection", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ mfaRequired: true, mfaChallenge: "old" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ mfaRequired: true, mfaChallenge: "fresh" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    renderWithProviders(<RootEnrollment />);
    await user.click(screen.getByRole("button", { name: en.session.enrollmentAction }));
    await user.type(screen.getByLabelText(en.session.email), "owner@example.test");
    await user.type(screen.getByLabelText(en.session.password), "secret");
    await user.click(screen.getByRole("button", { name: en.session.enrollmentAction }));
    await user.click(await screen.findByRole("button", { name: en.session.useRecovery }));
    await user.type(screen.getByLabelText(en.session.recoveryCode), "recovery-code");
    await user.click(screen.getByRole("button", { name: en.session.enrollmentAction }));
    expect(await screen.findByRole("alert")).toHaveTextContent(en.session.enrollmentError);
    expect(JSON.parse(String(fetchMock.mock.calls[1]![1]!.body))).toMatchObject({ mfaChallenge: "old", recoveryCode: "recovery-code", totpCode: "" });
    expect(screen.queryByLabelText(en.session.recoveryCode)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: en.session.enrollmentAction }));
    expect(JSON.parse(String(fetchMock.mock.calls[2]![1]!.body))).toMatchObject({ mfaChallenge: "", recoveryCode: "" });
    expect(await screen.findByLabelText(en.session.recoveryCode)).toHaveValue("");
  });
  it("reports an unconfirmed disconnect on a failed request", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network unavailable")));
    const user = userEvent.setup();
    renderWithProviders(<RootEnrollment />);
    await user.click(screen.getByRole("button", { name: en.session.revokeAction }));
    await user.click(within(screen.getByRole("alertdialog")).getByRole("button", { name: en.session.revokeAction }));
    expect(await screen.findByRole("alert")).toHaveTextContent(en.session.revokeError);
    expect(screen.queryByText(en.session.revokeSuccess)).not.toBeInTheDocument();
  });
  it("requires intent, handles MFA and clears credentials after connection", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ mfaRequired: true, mfaChallenge: "challenge" }), { status: 200 })).mockResolvedValueOnce(new Response(JSON.stringify({ mfaRequired: false }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    renderWithProviders(<RootEnrollment />);
    expect(fetchMock).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: en.session.enrollmentAction }));
    await user.type(screen.getByLabelText(en.session.email), "owner@example.test");
    await user.type(screen.getByLabelText(en.session.password), "secret");
    await user.click(screen.getByRole("button", { name: en.session.enrollmentAction }));
    await user.type(await screen.findByLabelText(en.session.totpCode), "123456");
    await user.click(screen.getByRole("button", { name: en.session.enrollmentAction }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Account connected."));
    expect(JSON.parse(String(fetchMock.mock.calls[1]![1]!.body))).toMatchObject({ mfaChallenge: "challenge", totpCode: "123456" });
    expect(screen.queryByLabelText(en.session.password)).not.toBeInTheDocument();
  });
});
