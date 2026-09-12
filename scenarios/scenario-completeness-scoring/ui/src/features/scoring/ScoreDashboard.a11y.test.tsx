/**
 * ScoreDashboard accessibility regression tests.
 *
 * The scoring feature owns its query states, so the a11y waits and mocks
 * live with the feature instead of leaking into `App.a11y.test.tsx`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeGetScoreResponse } from "./mocks/factories";
import { makeScoringMocks } from "./mocks/scoring";

vi.mock("../../api/scoring", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/scoring")>();
  return { ...actual, ...makeScoringMocks() };
});

import { ScoreDashboard } from "./ScoreDashboard";

describe("ScoreDashboard accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state without axe violations", async () => {
    const { container } = renderWithProviders(<ScoreDashboard />);

    expect(screen.getByTestId(selectors.scoring.empty)).toBeInTheDocument();

    await expectNoA11yViolations(container);
  });

  it("renders the full payload without axe violations", async () => {
    const { fetchScore } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockResolvedValueOnce(makeGetScoreResponse());

    const { container } = renderWithProviders(<ScoreDashboard />, {
      routerEntries: ["/?scenario=web-search"],
    });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.composite.card)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the error state without axe violations", async () => {
    const { fetchScore } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockRejectedValueOnce(new Error("scoring unavailable"));

    const { container } = renderWithProviders(<ScoreDashboard />, {
      routerEntries: ["/?scenario=nope"],
    });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.error)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
