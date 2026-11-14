import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { strings } from "../consts/strings";
import { AccountSession } from "./AccountSession";
import { sessionApi } from "../api/session";
import { renderWithProviders } from "../test-utils";

const copy = strings.lightDesk.account;

vi.mock("../api/session", () => ({ sessionApi: { status: vi.fn(), login: vi.fn(), refresh: vi.fn(), logout: vi.fn() } }));
afterEach(() => vi.resetAllMocks());
describe("AccountSession", () => {
  it("keeps MFA distinct from an authenticated session and verifies sign-in status", async () => {
    vi.mocked(sessionApi.status).mockResolvedValueOnce({ authenticated: false }).mockResolvedValueOnce({ authenticated: true, email: "owner@example.test" });
    vi.mocked(sessionApi.login).mockResolvedValueOnce({ authenticated: false, mfaRequired: true, mfaChallenge: "challenge" }).mockResolvedValueOnce({ authenticated: false });
    renderWithProviders(<AccountSession />);
    await screen.findByText(copy.signedOut);
    fireEvent.change(screen.getByLabelText(copy.email), { target: { value: "owner@example.test" } });
    fireEvent.change(screen.getByLabelText(copy.password), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: copy.signIn }));
    await screen.findByLabelText(copy.totpCode);
    expect(screen.queryByText(copy.signedIn)).toBeNull();
    fireEvent.change(screen.getByLabelText(copy.totpCode), { target: { value: "123456" } });
    fireEvent.click(screen.getByRole("button", { name: copy.signIn }));
    await screen.findByText(copy.signedIn);
    expect(sessionApi.login).toHaveBeenLastCalledWith(expect.objectContaining({ mfaChallenge: "challenge", totpCode: "123456" }));
    expect(screen.queryByLabelText(copy.password)).toBeNull();
  });
  it("renews an expired session and signs out through the provider", async () => {
    vi.mocked(sessionApi.status).mockResolvedValueOnce({ authenticated: false }).mockResolvedValueOnce({ authenticated: true, subject: "owner" }).mockResolvedValueOnce({ authenticated: false });
    vi.mocked(sessionApi.refresh).mockResolvedValue({ authenticated: false });
    vi.mocked(sessionApi.logout).mockResolvedValue({ authenticated: false });
    renderWithProviders(<AccountSession />);
    await screen.findByText(copy.signedOut);
    fireEvent.click(screen.getByRole("button", { name: copy.renew }));
    await screen.findByText(copy.signedIn);
    fireEvent.click(screen.getByRole("button", { name: copy.signOut }));
    await waitFor(() => expect(sessionApi.logout).toHaveBeenCalledTimes(1));
    await screen.findByText(copy.signedOut);
  });
  it("reports unavailable identity provider without claiming authentication", async () => {
    vi.mocked(sessionApi.status).mockResolvedValue({ authenticated: false });
    vi.mocked(sessionApi.refresh).mockRejectedValue(new Error("The installation sign-in provider is unavailable."));
    renderWithProviders(<AccountSession />);
    await screen.findByText(copy.signedOut);
    fireEvent.click(screen.getByRole("button", { name: copy.renew }));
    expect(await screen.findByRole("alert")).toHaveTextContent("provider is unavailable");
    expect(screen.queryByText(copy.signedIn)).toBeNull();
  });
});
