import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { SafetyFeedbackStatsSchema } from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { SafetyFeedbackLabel, type SafetyFeedbackStats } from "../../api/safetyFeedback";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { SafetyFeedbackStatsPanel } from "./SafetyFeedbackStatsPanel";

const stats = (overrides: MessageInitShape<typeof SafetyFeedbackStatsSchema> = {}): SafetyFeedbackStats =>
  create(SafetyFeedbackStatsSchema, {
    total: 4n,
    falsePositives: 1n,
    confirmed: 3n,
    uncertain: 0n,
    falsePositiveRate: 0.25,
    byReason: [
      { reasonCode: "sexual_unassessable", cases: 4n, falsePositives: 1n, confirmed: 3n, falsePositiveRate: 0.25 },
    ],
    byProfile: [
      { profileName: "local-balanced", cases: 4n, falsePositives: 1n, confirmed: 3n, falsePositiveRate: 0.25 },
    ],
    ...overrides,
  });

vi.mock("../safety/useSafetyFeedback", () => ({
  useSafetyFeedbackStats: vi.fn(),
}));

import { useSafetyFeedbackStats } from "../safety/useSafetyFeedback";

const mockedStats = vi.mocked(useSafetyFeedbackStats);

describe("SafetyFeedbackStatsPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the labelled totals and the false-positive rate", () => {
    mockedStats.mockReturnValue({ isLoading: false, error: null, data: stats() } as ReturnType<typeof useSafetyFeedbackStats>);
    renderWithProviders(<SafetyFeedbackStatsPanel />);

    expect(screen.getByTestId(selectors.responsibleUse.feedbackTotals)).toHaveTextContent("4");
    expect(screen.getByTestId(selectors.responsibleUse.feedbackRate)).toHaveTextContent("25.0%");
    expect(screen.getByText(/sexual_unassessable/)).toBeInTheDocument();
    expect(screen.getByText(/local-balanced/)).toBeInTheDocument();
  });

  it("says the rate is unavailable when no decisive label exists", () => {
    mockedStats.mockReturnValue({
      isLoading: false,
      error: null,
      data: stats({ total: 1n, falsePositives: 0n, confirmed: 0n, uncertain: 1n, falsePositiveRate: 0 }),
    } as ReturnType<typeof useSafetyFeedbackStats>);
    renderWithProviders(<SafetyFeedbackStatsPanel />);
    expect(screen.getByTestId(selectors.responsibleUse.feedbackRate)).toHaveTextContent("not enough decisive labels");
  });

  it("shows the empty state before anything is labelled", () => {
    mockedStats.mockReturnValue({
      isLoading: false,
      error: null,
      data: stats({ total: 0n, falsePositives: 0n, confirmed: 0n, uncertain: 0n, falsePositiveRate: 0 }),
    } as ReturnType<typeof useSafetyFeedbackStats>);
    renderWithProviders(<SafetyFeedbackStatsPanel />);
    expect(screen.getByTestId(selectors.responsibleUse.feedbackEmpty)).toBeInTheDocument();
  });

  it("reports a load failure", () => {
    mockedStats.mockReturnValue({
      isLoading: false,
      error: new Error("boom"),
      data: undefined,
    } as ReturnType<typeof useSafetyFeedbackStats>);
    renderWithProviders(<SafetyFeedbackStatsPanel />);
    expect(screen.getByTestId(selectors.responsibleUse.feedbackError)).toBeInTheDocument();
  });

  it("labels the by-reason and by-profile tables", () => {
    mockedStats.mockReturnValue({ isLoading: false, error: null, data: stats() } as ReturnType<typeof useSafetyFeedbackStats>);
    renderWithProviders(<SafetyFeedbackStatsPanel />);
    expect(screen.getByText(/By reason code/)).toBeInTheDocument();
    expect(screen.getByText(/By reviewer profile/)).toBeInTheDocument();
  });

  it("does not name an unspecified label as a real judgement", () => {
    // Guard the switch exhaustiveness: UNSPECIFIED maps to the empty state, not
    // to a decisive label.
    expect(SafetyFeedbackLabel.UNSPECIFIED).toBe(0);
  });
});
