/**
 * CoverageBanner tests. The load-bearing behaviors: an incomplete-coverage
 * report surfaces the recommended count and a register action that calls
 * acceptDefaultTargets with includeSensitive=false; sensitive targets render in
 * a separate review state with their own deliberate opt-in; and a complete
 * report with nothing sensitive renders nothing in the compact form.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { SourceKind } from "../../api/targets";

vi.mock("../../api/coverage", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/coverage")>()),
  getCoverageReport: vi.fn(),
  acceptDefaultTargets: vi.fn(),
}));

import * as coverageApi from "../../api/coverage";
import { CoverageBanner } from "./CoverageBanner";

const recommended = {
  id: "s1",
  owner: "vrooli",
  name: "plans",
  sourceKind: SourceKind.FILESYSTEM,
  locator: "/home/u/.vrooli/plans",
  rationale: "Your plans.",
  approxBytes: 4096n,
  sensitive: false,
  warning: "",
};

const sensitive = {
  id: "s2",
  owner: "codex",
  name: "auth",
  sourceKind: SourceKind.FILESYSTEM,
  locator: "/home/u/.codex/auth.json",
  rationale: "Codex OAuth tokens.",
  approxBytes: 471n,
  sensitive: true,
  warning: "Includes credentials.",
};

function report(overrides: Record<string, unknown> = {}) {
  return {
    summary: {
      registeredCount: 1,
      recommendedCount: 1,
      sensitiveCount: 1,
      plannedCount: 0,
      backedUpCount: 0,
      verifiedCount: 0,
      defaultCoverageComplete: false,
      hasSensitiveUnreviewed: true,
      hasUnplannedRegisteredTargets: false,
      hasUnverifiedTargets: true,
    },
    registeredTargets: [],
    recommendedTargets: [recommended],
    sensitiveTargets: [sensitive],
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(coverageApi.acceptDefaultTargets).mockResolvedValue({ dryRun: false } as never);
});

afterEach(() => cleanup());

describe("CoverageBanner", () => {
  it("registers recommended (non-sensitive) defaults on click", async () => {
    const user = userEvent.setup();
    vi.mocked(coverageApi.getCoverageReport).mockResolvedValue(report() as never);

    renderWithProviders(<CoverageBanner />);

    const button = await screen.findByTestId(selectors.coverage.registerRecommended);
    await user.click(button);

    await waitFor(() =>
      expect(coverageApi.acceptDefaultTargets).toHaveBeenCalledWith({
        includeSensitive: false,
        dryRun: false,
      }),
    );
  });

  it("registers sensitive targets only via the explicit opt-in control", async () => {
    const user = userEvent.setup();
    vi.mocked(coverageApi.getCoverageReport).mockResolvedValue(report() as never);

    renderWithProviders(<CoverageBanner />);

    const button = await screen.findByTestId(selectors.coverage.registerSensitive);
    await user.click(button);

    await waitFor(() =>
      expect(coverageApi.acceptDefaultTargets).toHaveBeenCalledWith({
        includeSensitive: true,
        dryRun: false,
      }),
    );
  });

  it("lists recommended and sensitive targets in detailed mode", async () => {
    vi.mocked(coverageApi.getCoverageReport).mockResolvedValue(report() as never);

    renderWithProviders(<CoverageBanner detailed />);

    const recList = await screen.findByTestId(selectors.coverage.recommendedList);
    const sensList = screen.getByTestId(selectors.coverage.sensitiveList);
    // Owner/name is data, not copy; assert via the list containers' content.
    expect(recList.textContent).toContain("vrooli/plans");
    expect(sensList.textContent).toContain("codex/auth");
  });

  it("renders nothing compact when coverage is complete and nothing is sensitive", async () => {
    vi.mocked(coverageApi.getCoverageReport).mockResolvedValue(
      report({
        summary: {
          registeredCount: 2,
          recommendedCount: 0,
          sensitiveCount: 0,
          plannedCount: 2,
          backedUpCount: 2,
          verifiedCount: 2,
          defaultCoverageComplete: true,
          hasSensitiveUnreviewed: false,
          hasUnplannedRegisteredTargets: false,
          hasUnverifiedTargets: false,
        },
        recommendedTargets: [],
        sensitiveTargets: [],
      }) as never,
    );

    const { container } = renderWithProviders(<CoverageBanner />);
    // The query resolves async; assert the banner never appears.
    await waitFor(() => {
      expect(screen.queryByTestId(selectors.coverage.banner)).not.toBeInTheDocument();
    });
    expect(container).toBeTruthy();
  });
});
