/**
 * A page that fails to render stays inside the shell: the router catches the
 * error before the app-level boundary, so the route table must name its own.
 */
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { withExpectedConsoleErrors } from "../test-utils/console";
import { TestAppRouter } from "./routes";

/** The error's own words, which reach the screen verbatim. */
const FAILURE = "desk page failed";

vi.mock("../pages/DeskPage", () => ({
  DeskPage: () => {
    throw new Error("desk page failed");
  },
}));

describe("a page that fails to render", () => {
  it("shows a plain error inside the shell, with the tabs still there", () => {
    // React and the router each report the caught error; those reports are expected here.
    const reported = (args: unknown[]) => args.some((a) => String(a).includes(FAILURE) || String(a).includes("<DeskPage>"));
    withExpectedConsoleErrors(reported, () => renderWithProviders(<TestAppRouter initialEntries={["/desk"]} />, { withoutRouter: true }));
    expect(screen.getByTestId(selectors.errorBoundary.root)).toBeInTheDocument();
    expect(screen.getByText(FAILURE)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.errorBoundary.retryButton)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.layout.navLink({ key: "needsYou" })).length).toBeGreaterThan(0);
  });
});
