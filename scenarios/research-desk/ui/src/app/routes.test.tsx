/**
 * Routing smoke: each destination and drill-in renders its page against an
 * empty API (every read answers with an empty message).
 */
import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { TestAppRouter } from "./routes";

const ROUTES: [string, string][] = [
  ["/", selectors.pages.now],
  ["/needs-you", selectors.pages.needsYou],
  ["/topics", selectors.pages.topics],
  ["/topics/debt-reduction", selectors.pages.topic],
  ["/commissions", selectors.pages.commissions],
  ["/commissions/programme-four-20261110", selectors.pages.commission],
  ["/launch", selectors.pages.launch],
  ["/launch/standing-teams-w1-20261110", selectors.pages.launchStatus],
  ["/reviews", selectors.pages.reviews],
  ["/reviews/r1", selectors.pages.review],
  ["/files/topics/debt-reduction/MISSION.md", selectors.pages.file],
  ["/desk", selectors.pages.desk],
];

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it.each(ROUTES)("renders %s", (path, testId) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(screen.getByTestId(testId)).toBeInTheDocument();
  });
});
