import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { renderWithProviders } from "../../test-utils";
import { SessionProvider } from "../session/SessionProvider";

const { login, register } = vi.hoisted(() => ({ login: vi.fn(), register: vi.fn() }));

// The hub's own same-origin IdentityService client — never scenario-authenticator.
vi.mock("../../api/identity", () => ({ identityClient: { login, register } }));

import { OwnerLoginForm } from "./OwnerLoginForm";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { loadSession, clearSession } from "../session/store";

function renderLoginForm() {
  return renderWithProviders(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <SessionProvider>
        <OwnerLoginForm onBack={vi.fn()} />
      </SessionProvider>
    </QueryClientProvider>,
  );
}

describe("OwnerLoginForm", () => {
  afterEach(() => {
    cleanup();
    clearSession();
    vi.clearAllMocks();
  });

  it("requires email and password before calling the API", async () => {
    const user = userEvent.setup();
    renderLoginForm();
    await user.click(screen.getByTestId(selectors.login.submit));
    expect(login).not.toHaveBeenCalled();
    expect(screen.getByTestId(selectors.login.error)).toHaveTextContent(strings.login.missingFields);
  });

  // [REQ:REQ-P0-005] Owner sign-in calls the hub's same-origin IdentityService;
  // the server owns the JWT in an HttpOnly cookie.
  it("signs in without persisting the owner token", async () => {
    const user = userEvent.setup();
    login.mockResolvedValueOnce({ token: "jwt-9", email: "owner@example.com", userId: "u-1" });
    renderLoginForm();

    await user.type(screen.getByTestId(selectors.login.emailInput), "owner@example.com");
    await user.type(screen.getByTestId(selectors.login.passwordInput), "hunter2");
    await user.click(screen.getByTestId(selectors.login.submit));

    await waitFor(() => {
      expect(loadSession().ownerToken).toBeNull();
    });
    expect(loadSession().ownerEmail).toBe("owner@example.com");
    expect(login).toHaveBeenCalledWith({ email: "owner@example.com", password: "hunter2" });
    expect(register).not.toHaveBeenCalled();
  });

  it("surfaces invalid credentials distinctly", async () => {
    const user = userEvent.setup();
    login.mockRejectedValueOnce(new ConnectError("bad", Code.Unauthenticated));
    renderLoginForm();

    await user.type(screen.getByTestId(selectors.login.emailInput), "owner@example.com");
    await user.type(screen.getByTestId(selectors.login.passwordInput), "wrong");
    await user.click(screen.getByTestId(selectors.login.submit));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.login.error)).toHaveTextContent(strings.login.invalidCredentials);
    });
    expect(loadSession().ownerToken).toBeNull();
  });

  // [REQ:REQ-P0-005] A brand-new owner creates an account same-origin and is signed in.
  it("creates an account via the Create-account tab without storing the token", async () => {
    const user = userEvent.setup();
    register.mockResolvedValueOnce({ token: "jwt-new", email: "new@example.com", userId: "u-2" });
    renderLoginForm();

    await user.click(screen.getByTestId(selectors.login.tabCreate));
    await user.type(screen.getByTestId(selectors.login.emailInput), "new@example.com");
    await user.type(screen.getByTestId(selectors.login.passwordInput), "Str0ng!pw");
    await user.type(screen.getByTestId(selectors.login.usernameInput), "Alex");
    await user.click(screen.getByTestId(selectors.login.submit));

    await waitFor(() => {
      expect(loadSession().ownerToken).toBeNull();
    });
    expect(register).toHaveBeenCalledWith({ email: "new@example.com", password: "Str0ng!pw", username: "Alex" });
  });

  it("reports a duplicate email on registration", async () => {
    const user = userEvent.setup();
    register.mockRejectedValueOnce(new ConnectError("taken", Code.AlreadyExists));
    renderLoginForm();

    await user.click(screen.getByTestId(selectors.login.tabCreate));
    await user.type(screen.getByTestId(selectors.login.emailInput), "dup@example.com");
    await user.type(screen.getByTestId(selectors.login.passwordInput), "Str0ng!pw");
    await user.click(screen.getByTestId(selectors.login.submit));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.login.error)).toHaveTextContent(strings.register.emailTaken);
    });
    expect(loadSession().ownerToken).toBeNull();
  });
});
