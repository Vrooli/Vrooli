import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import App from "./App";
import { renderWithProviders } from "./test-utils";
import { selectors } from "./consts/selectors";

describe("App composition", () => {
  afterEach(cleanup);

  it("renders the production app with its browser router and providers", () => {
    window.history.replaceState({}, "", "/");
    renderWithProviders(<App />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });
});
