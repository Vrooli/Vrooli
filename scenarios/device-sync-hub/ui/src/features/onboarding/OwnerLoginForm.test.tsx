import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

const { loginOwner, resolveAuthenticatorBaseUrl } = vi.hoisted(() => ({
  loginOwner: vi.fn(),
  resolveAuthenticatorBaseUrl: vi.fn((): string | null => "http://auth.example"),
}));

vi.mock("../../api/authenticator", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/authenticator")>();
  return { ...actual, loginOwner, resolveAuthenticatorBaseUrl };
});

import { OwnerLoginForm } from "./OwnerLoginForm";
import { AuthError } from "../../api/authenticator";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { loadSession, clearSession } from "../session/store";

describe("OwnerLoginForm", () => {
  afterEach(() => {
    cleanup();
    clearSession();
    vi.clearAllMocks();
    resolveAuthenticatorBaseUrl.mockReturnValue("http://auth.example");
  });

  it("requires email and password before calling login", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OwnerLoginForm onBack={vi.fn()} />);
    await user.click(screen.getByTestId(selectors.login.submit));
    expect(loginOwner).not.toHaveBeenCalled();
    expect(screen.getByTestId(selectors.login.error)).toHaveTextContent(strings.login.missingFields);
  });

  // [REQ:REQ-P0-005] Owner sign-in delegates to scenario-authenticator and stores the JWT.
  it("logs in and stores the owner token + email", async () => {
    const user = userEvent.setup();
    loginOwner.mockResolvedValueOnce({ token: "jwt-9", email: "owner@example.com" });
    renderWithProviders(<OwnerLoginForm onBack={vi.fn()} />);

    await user.type(screen.getByTestId(selectors.login.emailInput), "owner@example.com");
    await user.type(screen.getByTestId(selectors.login.passwordInput), "hunter2");
    await user.click(screen.getByTestId(selectors.login.submit));

    await waitFor(() => {
      expect(loadSession().ownerToken).toBe("jwt-9");
    });
    expect(loadSession().ownerEmail).toBe("owner@example.com");
    expect(loginOwner).toHaveBeenCalledWith("http://auth.example", {
      email: "owner@example.com",
      password: "hunter2",
    });
  });

  it("surfaces invalid credentials distinctly", async () => {
    const user = userEvent.setup();
    loginOwner.mockRejectedValueOnce(new AuthError("invalid_credentials", "bad"));
    renderWithProviders(<OwnerLoginForm onBack={vi.fn()} />);

    await user.type(screen.getByTestId(selectors.login.emailInput), "owner@example.com");
    await user.type(screen.getByTestId(selectors.login.passwordInput), "wrong");
    await user.click(screen.getByTestId(selectors.login.submit));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.login.error)).toHaveTextContent(strings.login.invalidCredentials);
    });
    expect(loadSession().ownerToken).toBeNull();
  });

  it("accepts a pasted owner token via the advanced fallback", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OwnerLoginForm onBack={vi.fn()} />);

    await user.click(screen.getByTestId(selectors.login.advancedToggle));
    await user.type(screen.getByTestId(selectors.owner.tokenInput), "pasted-jwt");
    await user.click(screen.getByTestId(selectors.owner.signInButton));

    await waitFor(() => {
      expect(loadSession().ownerToken).toBe("pasted-jwt");
    });
    expect(loginOwner).not.toHaveBeenCalled();
  });

  it("degrades to the token paste when the authenticator URL is unresolved", () => {
    resolveAuthenticatorBaseUrl.mockReturnValue(null);
    renderWithProviders(<OwnerLoginForm onBack={vi.fn()} />);
    expect(screen.queryByTestId(selectors.login.submit)).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.owner.tokenInput)).toBeInTheDocument();
  });
});
