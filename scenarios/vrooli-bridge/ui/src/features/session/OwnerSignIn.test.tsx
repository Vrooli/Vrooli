import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { EnrollOperatorSessionResponseSchema } from "@vrooli/proto-types/vrooli-bridge/v1/identity/identity_pb";

import { renderWithProviders, seedSession } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

// OwnerSignIn posts same-origin to the bridge's IdentityService; stub the client
// so the test never reaches the network.
vi.mock("../../api/identity", () => ({
  identityClient: { login: vi.fn(), register: vi.fn(), enrollOperatorSession: vi.fn() },
}));

vi.mock("./browser_session", () => ({
  generateBrowserKeyMaterial: vi.fn().mockResolvedValue({ publicKey: new Uint8Array([1, 2, 3]), privateKeyPkcs8: "private" }),
  mintBrowserSession: vi.fn().mockResolvedValue("OS1.local-session"),
  saveBrowserEnrollment: vi.fn().mockResolvedValue(undefined),
  loadBrowserEnrollment: vi.fn().mockResolvedValue(null),
}));

import { identityClient } from "../../api/identity";
import { OwnerSignIn } from "./OwnerSignIn";
import { readOwnerToken } from "./store";

const login = vi.mocked(identityClient.login);
const register = vi.mocked(identityClient.register);
const enrollOperatorSession = vi.mocked(identityClient.enrollOperatorSession);

type LoginResult = Awaited<ReturnType<typeof identityClient.login>>;
type RegisterResult = Awaited<ReturnType<typeof identityClient.register>>;

describe("OwnerSignIn", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  beforeEach(() => {
    enrollOperatorSession.mockResolvedValue(create(EnrollOperatorSessionResponseSchema, {
      enrollmentReference: "enrollment-1",
      operatorId: "owner-1",
      identityProvider: "scenario-authenticator",
      mode: "personal",
      scopeCeiling: ["vrooli-bridge:read"],
      sessionTtlSeconds: 900n,
    }));
  });

  it("shows the same-origin sign-in form when no owner token is present", () => {
    renderWithProviders(<OwnerSignIn />);
    expect(screen.getByTestId(selectors.session.login.form)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.session.owner.status)).not.toBeInTheDocument();
  });

  it("enrolls a browser key and keeps only the local session in memory", async () => {
    const user = userEvent.setup();
    login.mockResolvedValue({
      token: "jwt-1",
      refreshToken: "",
      email: "owner@example.com",
      userId: "u-1",
    } as unknown as LoginResult);

    renderWithProviders(<OwnerSignIn />);
    await user.type(screen.getByTestId(selectors.session.login.emailInput), "owner@example.com");
    await user.type(screen.getByTestId(selectors.session.login.passwordInput), "pw");
    await user.click(screen.getByTestId(selectors.session.login.submit));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.session.owner.status)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.session.owner.signOutButton)).toBeInTheDocument();
    expect(readOwnerToken()).toBe("OS1.local-session");
    expect(login).toHaveBeenCalledWith({ email: "owner@example.com", password: "pw" });
    expect(enrollOperatorSession).toHaveBeenCalled();
    expect(JSON.stringify(window.localStorage)).not.toContain("jwt-1");
  });

  it("maps an unauthenticated Connect error to the invalid-credentials message", async () => {
    const user = userEvent.setup();
    login.mockRejectedValue(new ConnectError("nope", Code.Unauthenticated));

    renderWithProviders(<OwnerSignIn />);
    await user.type(screen.getByTestId(selectors.session.login.emailInput), "owner@example.com");
    await user.type(screen.getByTestId(selectors.session.login.passwordInput), "bad");
    await user.click(screen.getByTestId(selectors.session.login.submit));

    const error = await screen.findByTestId(selectors.session.login.error);
    // cimode renders the key path, so the mapping is asserted against the registry.
    expect(error).toHaveTextContent(strings.session.login.invalidCredentials);
  });

  it("registers through the Create-account tab and lands signed in", async () => {
    const user = userEvent.setup();
    register.mockResolvedValue({
      token: "jwt-new",
      refreshToken: "",
      email: "new@example.com",
      userId: "u-2",
    } as unknown as RegisterResult);

    renderWithProviders(<OwnerSignIn />);
    await user.click(screen.getByTestId(selectors.session.login.tabCreate));
    await user.type(screen.getByTestId(selectors.session.login.emailInput), "new@example.com");
    await user.type(screen.getByTestId(selectors.session.login.passwordInput), "Str0ng!pw");
    await user.click(screen.getByTestId(selectors.session.login.submit));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.session.owner.status)).toBeInTheDocument(),
    );
    expect(register).toHaveBeenCalledWith({ email: "new@example.com", password: "Str0ng!pw", username: "" });
    expect(login).not.toHaveBeenCalled();
    expect(enrollOperatorSession).toHaveBeenCalled();
  });

  it("shows the signed-in status + sign out when an owner token is present", () => {
    seedSession({ ownerToken: "existing-jwt", ownerEmail: "owner@example.com" });
    renderWithProviders(<OwnerSignIn />);
    expect(screen.getByTestId(selectors.session.owner.status)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.session.owner.signOutButton)).toBeInTheDocument();
  });

  it("clears the owner token and returns to the sign-in form on sign out", async () => {
    const user = userEvent.setup();
    seedSession({ ownerToken: "existing-jwt" });
    renderWithProviders(<OwnerSignIn />);

    expect(screen.getByTestId(selectors.session.owner.status)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.session.owner.signOutButton));

    await waitFor(() => expect(screen.getByTestId(selectors.session.login.form)).toBeInTheDocument());
    expect(readOwnerToken()).toBeNull();
  });
});
