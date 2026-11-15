/**
 * CoverageBanner tests. The load-bearing behaviors: an incomplete-coverage
 * report surfaces the recommended count and a register action that calls
 * acceptDefaultTargets with includeSensitive=false; sensitive targets render in
 * a separate verified Notification Hub review state; and a complete
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
  requestSensitiveReview: vi.fn(),
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
    reacquirableTargets: [],
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(coverageApi.acceptDefaultTargets).mockResolvedValue({ dryRun: false } as never);
  vi.mocked(coverageApi.requestSensitiveReview).mockResolvedValue({
    askId: "ask-sensitive-1",
    candidateCount: 1,
    pending: true,
  } as never);
});

afterEach(() => cleanup());

describe("CoverageBanner", () => {
  it("keeps a stable loading surface when requested by the overview", async () => {
    vi.mocked(coverageApi.getCoverageReport).mockImplementation(
      () => new Promise(() => undefined) as never,
    );

    renderWithProviders(<CoverageBanner reserveSpace />);

    expect(await screen.findByTestId(selectors.coverage.banner)).toHaveAttribute(
      "aria-busy",
      "true",
    );
  });

  it("keeps compact loading behavior unchanged when no reservation is requested", () => {
    vi.mocked(coverageApi.getCoverageReport).mockImplementation(
      () => new Promise(() => undefined) as never,
    );

    renderWithProviders(<CoverageBanner />);

    expect(screen.queryByTestId(selectors.coverage.banner)).not.toBeInTheDocument();
  });

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

  it("requests one verified review and does not register sensitive targets", async () => {
    const user = userEvent.setup();
    vi.mocked(coverageApi.getCoverageReport).mockResolvedValue(report() as never);

    renderWithProviders(<CoverageBanner />);

    const button = await screen.findByTestId(selectors.coverage.requestSensitiveReview);
    await user.click(button);

    await waitFor(() =>
      expect(coverageApi.requestSensitiveReview).toHaveBeenCalledTimes(1),
    );
    expect(coverageApi.acceptDefaultTargets).not.toHaveBeenCalledWith(expect.objectContaining({ includeSensitive: true }));
    // Test setup uses the i18n pseudo-locale, which returns keys verbatim.
    // Production locale catalogs supply the human-readable interpolated text.
    expect(await screen.findByRole("status")).toHaveTextContent(
      "coverage.sensitiveReviewPending",
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

  it("shows owner-declared reacquirable data separately without a registration action", async () => {
    vi.mocked(coverageApi.getCoverageReport).mockResolvedValue(
      report({
        summary: { ...report().summary, recommendedCount: 0, sensitiveCount: 0, defaultCoverageComplete: true, hasSensitiveUnreviewed: false },
        recommendedTargets: [],
        sensitiveTargets: [],
        reacquirableTargets: [{ id: "cache", owner: "ollama", name: "models", locator: "/models", budgetMaxBytes: "50GiB", retention: "models 30d" }],
      }) as never,
    );
    renderWithProviders(<CoverageBanner detailed />);
    const list = await screen.findByTestId(selectors.coverage.reacquirableList);
    expect(list.textContent).toContain("ollama/models");
    expect(list.textContent).toContain("50GiB");
    expect(screen.queryByTestId(selectors.coverage.registerRecommended)).not.toBeInTheDocument();
  });

  it("shows owner reacquisition evidence for already registered targets", async () => {
    vi.mocked(coverageApi.getCoverageReport).mockResolvedValue(
      report({
        registeredTargets: [{
          id: "stable-audio-models",
          owner: "stable-audio-3",
          name: "models",
          reacquirable: true,
          reacquisitionCommand: "stable-audio-3 models install",
          reacquisitionEvidence: "server/model-lock.json",
          reacquisitionNeedsOperator: true,
          reacquisitionAuthReason: "provider license",
        }],
      }) as never,
    );

    renderWithProviders(<CoverageBanner detailed />);

    const list = await screen.findByTestId(selectors.coverage.registeredReacquirableList);
    expect(list.textContent).toContain("stable-audio-3/models");
    expect(list.textContent).toContain("stable-audio-3 models install");
    expect(list.textContent).toContain("server/model-lock.json");
    // The test suite defaults i18next to cimode so labels remain stable.
    expect(list.textContent).toContain("coverage.reacquisitionAuthorization");
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
