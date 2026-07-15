import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders, seedSession } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { AppGate } from "./AppGate";

describe("AppGate", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the sign-in screen (not the app) when no owner token is present", () => {
    renderWithProviders(
      <AppGate>
        <div data-testid="gated-app" />
      </AppGate>,
    );
    expect(screen.getByTestId(selectors.session.signInScreen)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.session.login.form)).toBeInTheDocument();
    expect(screen.queryByTestId("gated-app")).not.toBeInTheDocument();
  });

  it("renders the app (not the sign-in screen) once an owner token is present", () => {
    seedSession({ ownerToken: "existing-jwt" });
    renderWithProviders(
      <AppGate>
        <div data-testid="gated-app" />
      </AppGate>,
    );
    expect(screen.getByTestId("gated-app")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.session.signInScreen)).not.toBeInTheDocument();
  });
});
