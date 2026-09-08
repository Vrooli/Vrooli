import { afterEach, describe, expect, it, vi } from "vitest";
import { Code, ConnectError } from "@connectrpc/connect";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { TestAppRouter } from "../app/routes";
import { accountsClient } from "../api/client";

vi.mock("../api/client", () => ({
  accountsClient: {
    login: vi.fn(),
    register: vi.fn(),
  },
}));

const login = vi.mocked(accountsClient.login);
const register = vi.mocked(accountsClient.register);

function renderLogin() {
  return renderWithProviders(<TestAppRouter initialEntries={["/auth/login?return_to=%2Fdesktop"]} />, { withoutRouter: true });
}

function last<T>(items: T[]): T {
  const item = items[items.length - 1];
  if (!item) throw new Error("expected at least one matching element");
  return item;
}

describe("HostedLoginPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("requires the sign-in fields and shows the requested return target", async () => {
    const user = userEvent.setup();
    renderLogin();

    expect(screen.getByTestId("auth-return-target")).toBeInTheDocument();
    await user.click(screen.getByTestId("auth-submit"));

    expect(screen.getByRole("alert")).toHaveTextContent("auth.requiredFields");
    expect(login).not.toHaveBeenCalled();
  });

  it("signs in without exposing the bearer token and returns to the app", async () => {
    const user = userEvent.setup();
    login.mockResolvedValue({
      account: { email: "operator@example.test" },
      tokens: { accessToken: "secret-access-token" },
    } as never);
    renderLogin();

    await user.type(screen.getByTestId("auth-email"), "operator@example.test");
    await user.type(screen.getByTestId("auth-password"), "password");
    await user.click(screen.getByTestId("auth-submit"));

    expect(await screen.findByTestId("hosted-login-success")).toBeInTheDocument();
    expect(screen.getByTestId("hosted-login-success")).not.toHaveTextContent("secret-access-token");
    expect(login).toHaveBeenCalledOnce();
  });

  it("registers a new account through the same hosted surface", async () => {
    const user = userEvent.setup();
    register.mockResolvedValue({
      account: { email: "new@example.test" },
      tokens: { accessToken: "registration-token" },
    } as never);
    renderLogin();

    await user.click(screen.getByTestId("auth-register-tab"));
    await user.type(screen.getByTestId("auth-email"), "new@example.test");
    await user.type(screen.getByTestId("auth-password"), "password");
    await user.type(screen.getByTestId("auth-username"), "new-user");
    await user.click(screen.getByTestId("auth-submit"));

    expect(await screen.findByTestId("hosted-login-success")).toBeInTheDocument();
    expect(register).toHaveBeenCalledOnce();
    expect(login).not.toHaveBeenCalled();
  });

  it("maps provider refusal and incomplete responses to safe user-facing errors", async () => {
    const user = userEvent.setup();
    login.mockRejectedValueOnce(new ConnectError("denied", Code.Unauthenticated));
    renderLogin();
    await user.type(screen.getByTestId("auth-email"), "operator@example.test");
    await user.type(screen.getByTestId("auth-password"), "password");
    await user.click(screen.getByTestId("auth-submit"));
    expect(await screen.findByRole("alert")).toHaveTextContent("auth.invalidCredentials");

    cleanup();
    login.mockResolvedValueOnce({ tokens: {} } as never);
    renderLogin();
    const emailInputs = screen.getAllByTestId("auth-email");
    const passwordInputs = screen.getAllByTestId("auth-password");
    await user.type(last(emailInputs), "operator@example.test");
    await user.type(last(passwordInputs), "password");
    await user.click(last(screen.getAllByTestId("auth-submit")));
    await vi.waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(last(alerts)).toHaveTextContent("auth.unableToComplete");
    });
  });
});
