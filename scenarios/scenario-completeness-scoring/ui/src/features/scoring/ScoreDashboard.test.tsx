/**
 * ScoreDashboard tests — focused on the scoring surface only.
 *
 * Renders <ScoreDashboard /> directly so failures point at scoring-feature
 * behaviour, not shell composition. Follows the canonical mock-builder
 * pattern from `@/test-utils`; the `?scenario=` search param is driven via
 * `routerEntries`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeFreshnessBlock,
  makeGetScoreResponse,
  makeListScoresResponse,
  makeMaturityHeadline,
  makeCollectorDegradation,
} from "./mocks/factories";
import { makeScoringMocks } from "./mocks/scoring";

vi.mock("../../api/scoring", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/scoring")>();
  return { ...actual, ...makeScoringMocks() };
});

import { ScoreDashboard } from "./ScoreDashboard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ScoreDashboard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty prompt and fires no scenario score query without a scenario param", async () => {
    const { fetchScore, fetchScoreTrend, fetchScores } = await import("../../api/scoring");

    renderWithProviders(<ScoreDashboard />);

    expect(screen.getByTestId(selectors.scoring.empty)).toBeInTheDocument();
    expect(vi.mocked(fetchScore)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchScoreTrend)).not.toHaveBeenCalled();
    await waitFor(() => {
      expect(vi.mocked(fetchScores)).toHaveBeenCalledWith({ pageToken: "", pageSize: 10 });
    });
  });

  it("renders the full payload for the scenario in the search param", async () => {
    const { fetchScore, fetchScoreTrend, fetchScores } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockResolvedValueOnce(makeGetScoreResponse());

    renderWithProviders(<ScoreDashboard />, { routerEntries: ["/?scenario=web-search"] });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.composite.card)).toBeInTheDocument();
    });
    expect(vi.mocked(fetchScore)).toHaveBeenCalledWith("web-search");
    expect(vi.mocked(fetchScoreTrend)).toHaveBeenCalledWith("web-search");
    expect(vi.mocked(fetchScores)).toHaveBeenCalledWith({ pageToken: "", pageSize: 10 });
    expect(screen.getByTestId(selectors.scoring.composite.score).textContent).toContain("82/100");
    expect(screen.getByTestId(selectors.scoring.maturity.workingRung).textContent).toContain(
      "R1 Safe & standards-clean",
    );
    expect(screen.getByTestId(selectors.scoring.maturity.digest).textContent).toContain("td:abc123");
    expect(
      screen.getByTestId(selectors.scoring.freshnessPhaseRow({ phase: "smoke" })).textContent,
    ).toContain("td:older");
    expect(screen.getByTestId(selectors.scoring.freshness.refreshCommand).textContent).toBe(
      "vrooli scenario test web-search --phases smoke",
    );
    expect(screen.getByTestId(selectors.scoring.importance.score).textContent).toContain("0.8 / 1.0");
    expect(screen.getByTestId(selectors.scoring.importance.signals).textContent).toContain(
      "test-genie",
    );
    expect(screen.getByTestId(selectors.scoring.recommendations.card).textContent).toContain(
      "Fix the 2 standards errors blocking R1.",
    );
    expect(screen.getByTestId(selectors.scoring.actionPlan.projected)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.scoring.trend.delta).textContent).toContain("+7");
    expect(screen.getByTestId(selectors.scoring.trend.series).children).toHaveLength(2);
    expect(screen.getByTestId(selectors.scoring.fleet.table).textContent).toContain("cli-health");
    expect(screen.queryByTestId(selectors.scoring.degradations.card)).not.toBeInTheDocument();
  });

  it("pages the fleet table through ListScores next_page_token", async () => {
    const { fetchScores } = await import("../../api/scoring");
    vi.mocked(fetchScores)
      .mockResolvedValueOnce(makeListScoresResponse({ nextPageToken: "next-page" }))
      .mockResolvedValueOnce(makeListScoresResponse({ nextPageToken: "" }));
    const user = userEvent.setup();

    renderWithProviders(<ScoreDashboard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.fleet.next)).toBeEnabled();
    });
    await user.click(screen.getByTestId(selectors.scoring.fleet.next));

    await waitFor(() => {
      expect(vi.mocked(fetchScores)).toHaveBeenLastCalledWith({ pageToken: "next-page", pageSize: 10 });
    });
  });

  it("submits the typed scenario name and queries it", async () => {
    const { fetchScore } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockResolvedValue(makeGetScoreResponse({ scenario: "cli-health" }));
    const user = userEvent.setup();

    renderWithProviders(<ScoreDashboard />);

    await user.type(screen.getByTestId(selectors.scoring.input), "  cli-health  ");
    await user.click(screen.getByTestId(selectors.scoring.submit));

    await waitFor(() => {
      expect(vi.mocked(fetchScore)).toHaveBeenCalledWith("cli-health");
    });
  });

  it("renders the ladder-clean headline when every rung holds", async () => {
    const { fetchScore } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockResolvedValueOnce(
      makeGetScoreResponse({
        maturity: makeMaturityHeadline({ ladderClean: true, workingRung: "", satisfiedThrough: "" }),
      }),
    );

    renderWithProviders(<ScoreDashboard />, { routerEntries: ["/?scenario=web-search"] });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.maturity.ladderClean)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.scoring.maturity.workingRung)).not.toBeInTheDocument();
  });

  it("labels the digest unavailable when computation failed", async () => {
    const { fetchScore } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockResolvedValueOnce(
      makeGetScoreResponse({
        freshness: makeFreshnessBlock({ currentDigest: "", digestError: "git unavailable" }),
      }),
    );

    renderWithProviders(<ScoreDashboard />, { routerEntries: ["/?scenario=web-search"] });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.maturity.digest).textContent).toContain(
        "git unavailable",
      );
    });
  });

  it("surfaces collector degradations", async () => {
    const { fetchScore } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockResolvedValueOnce(
      makeGetScoreResponse({ degradations: [makeCollectorDegradation()] }),
    );

    renderWithProviders(<ScoreDashboard />, { routerEntries: ["/?scenario=web-search"] });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.degradations.card).textContent).toContain(
        "ui sources unreadable",
      );
    });
  });

  it("renders the error state when the query rejects", async () => {
    const { fetchScore } = await import("../../api/scoring");
    vi.mocked(fetchScore).mockRejectedValueOnce(new Error("scoring unavailable"));

    renderWithProviders(<ScoreDashboard />, { routerEntries: ["/?scenario=nope"] });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.scoring.error)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.scoring.composite.card)).not.toBeInTheDocument();
  });
});
