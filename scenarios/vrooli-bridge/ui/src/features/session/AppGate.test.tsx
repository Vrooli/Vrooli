import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders, seedSession } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { AppGate } from "./AppGate";
import * as browserSession from "./browser_session";
import { clearSession } from "./store";

describe("AppGate", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    clearSession();
    window.localStorage.clear();
  });

  it("renders the sign-in screen (not the app) when no owner token is present", async () => {
    renderWithProviders(
      <AppGate>
        <div data-testid="gated-app" />
      </AppGate>,
    );
    expect(await screen.findByTestId(selectors.session.signInScreen)).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.session.login.form)).toBeInTheDocument();
    expect(screen.queryByTestId("gated-app")).not.toBeInTheDocument();
  });

  it("renders the app (not the sign-in screen) once an owner token is present", async () => {
    seedSession({ ownerToken: "existing-jwt" });
    renderWithProviders(
      <AppGate>
        <div data-testid="gated-app" />
      </AppGate>,
    );
    expect(await screen.findByTestId("gated-app")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.session.signInScreen)).not.toBeInTheDocument();
  });

  it("pauses sign-in and retries the same saved enrollment after storage returns", async () => {
    const user = userEvent.setup();
    window.localStorage.setItem("vrooli-bridge.operator-session", JSON.stringify({
      operatorId: "owner-1",
      identityProvider: "scenario-authenticator",
      mode: "personal",
      reference: "enrollment-1",
      enrolledAt: "2026-08-17T00:00:00Z",
      scopeCeiling: ["vrooli-bridge:read"],
      privateKeyPkcs8: "private",
    }));
    const originalGetItem = window.localStorage.getItem.bind(window.localStorage);
    let enrollmentReads = 0;
    vi.spyOn(Storage.prototype, "getItem").mockImplementation((key) => {
      if (key === "vrooli-bridge.operator-session" && ++enrollmentReads === 2) {
        throw new DOMException("storage temporarily blocked", "SecurityError");
      }
      return originalGetItem(key);
    });
    const mint = vi.spyOn(browserSession, "mintBrowserSession").mockResolvedValue("OS1.restored");

    renderWithProviders(
      <AppGate>
        <div data-testid="gated-app" />
      </AppGate>,
    );

    expect(await screen.findByTestId(selectors.session.restore.retry)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.session.restore.status)).toHaveTextContent(
      "session.restore.unavailable",
    );
    expect(screen.queryByTestId(selectors.session.login.form)).not.toBeInTheDocument();
    expect(screen.queryByTestId("gated-app")).not.toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.session.restore.retry));

    await waitFor(() => expect(screen.getByTestId("gated-app")).toBeInTheDocument());
    expect(mint).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId(selectors.session.login.form)).not.toBeInTheDocument();
  });
});
