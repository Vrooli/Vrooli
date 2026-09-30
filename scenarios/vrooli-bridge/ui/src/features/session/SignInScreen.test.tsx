import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

import { SignInScreen } from "./SignInScreen";

describe("SignInScreen", () => {
  afterEach(() => {
    cleanup();
  });

  it("makes the first-run path obvious and keeps create-account one click away", () => {
    renderWithProviders(<SignInScreen />);
    // The first-time note orients a newcomer toward creating the owner account.
    expect(screen.getByTestId(selectors.session.firstTimeNote)).toBeInTheDocument();
    // The Create-account tab is visible from the default view (one click).
    expect(screen.getByTestId(selectors.session.login.tabCreate)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.session.login.form)).toBeInTheDocument();
  });
});
