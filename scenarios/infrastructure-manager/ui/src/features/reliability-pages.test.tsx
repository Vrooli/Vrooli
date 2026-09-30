import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings.generated";

const mocks = vi.hoisted(() => ({
  fetchCells: vi.fn(),
  fetchCondition: vi.fn(),
  fetchCoverage: vi.fn(),
  fetchFocus: vi.fn(),
  fetchTrust: vi.fn(),
}));

vi.mock("../api/reliability", () => mocks);

import { ConditionPage } from "./condition/ConditionPage";
import { CoveragePage } from "./coverage/CoveragePage";
import { FocusPage } from "./focus/FocusPage";

/**
 * These tests assert the reliability pages' HONESTY RULES, not their markup.
 *
 * Copy is asserted through `strings.*`: tests run in i18next `cimode`, where
 * `t()` returns the key, so a key assertion proves the right phrase is wired
 * without pinning any wording in any locale. Data that came off the wire (a
 * cell ref, a finding title, a reason) is asserted with a regex or through a
 * role query, because it is data rather than copy.
 */

function renderPage(ui: Parameters<typeof renderWithProviders>[0]) {
  return renderWithProviders(ui, {
    queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  });
}

/** A projection that answered: one cell missing, an authored denominator. */
const readProjection = {
  projection: 1,
  available: true,
  ratio: { value: 0.5 },
  confidence: { level: 1, rationale: "owner-authored" },
  nowCount: 1,
  inReachCount: 1,
  missingCount: 1,
};

/**
 * A projection that did NOT answer. Every counter is `0` on the wire — that is
 * a protobuf default, not a measurement, and the page must never print it.
 */
const unreadProjection = {
  projection: 2,
  available: false,
  ratio: undefined,
  confidence: { level: 2, rationale: "" },
  nowCount: 0,
  inReachCount: 0,
  missingCount: 0,
  unavailableReason: "source unavailable",
};

describe("reliability detail pages", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => cleanup());

  it("names loading state on each detail page", () => {
    const pending = () => new Promise<never>(() => undefined);
    mocks.fetchCondition.mockImplementation(pending);
    mocks.fetchTrust.mockImplementation(pending);
    mocks.fetchCoverage.mockImplementation(pending);
    mocks.fetchCells.mockImplementation(pending);
    mocks.fetchFocus.mockImplementation(pending);

    const { unmount } = renderPage(<ConditionPage />);
    expect(screen.getAllByText(strings.pages.condition.reading).length).toBeGreaterThan(0);
    unmount();
    renderPage(<CoveragePage />);
    expect(screen.getAllByText(strings.pages.coverage.reading).length).toBeGreaterThan(0);
    expect(screen.getAllByText(strings.pages.coverage.readingGaps).length).toBeGreaterThan(0);
    cleanup();
    renderPage(<FocusPage />);
    expect(screen.getAllByText(strings.pages.focus.reading).length).toBeGreaterThan(0);
  });

  it("renders condition readings, trust, history, and unavailable sources", async () => {
    mocks.fetchCondition.mockResolvedValue({
      readings: [{ id: "r-1", cellRef: "api/latency", value: 42, unit: "ms", source: "autoheal", trustVerdict: 1, bandVerdict: 1 }],
      sources: [{ source: "autoheal", available: false, reason: "provider paused" }],
    });
    mocks.fetchTrust.mockResolvedValue({ trust: { distribution: [{ verdict: 1, count: 1 }], checkedDenominator: 1, total: 2 } });

    renderPage(<ConditionPage />);

    expect(await screen.findByRole("rowheader", { name: /api\/latency/ })).toBeInTheDocument();
    // The reason travels verbatim on the instrument chrome, in the lamp's
    // accessible name, so a screen reader hears why the source is dark.
    expect(
      within(screen.getByTestId("condition-availability")).getByRole("img", { name: /provider paused/ }),
    ).toBeInTheDocument();
    expect(screen.getByText(strings.pages.condition.historyEmptyTitle)).toBeInTheDocument();
    // The trust triple stays one unit: the distribution and both denominators.
    const triple = screen.getByRole("region", { name: strings.instrument.trustDistributionLabel });
    expect(triple).toHaveTextContent(strings.instrument.trust.valid);
    expect(triple).toHaveTextContent(strings.instrument.checkedOf);
  });

  it("withholds the band verdict from a reading that is not trusted", async () => {
    mocks.fetchCondition.mockResolvedValue({
      readings: [
        { id: "r-1", cellRef: "api/latency", value: 42, unit: "ms", source: "autoheal", trustVerdict: 1, bandVerdict: 1 },
        // GHOST trust, and the wire still carries IN_BAND. It must not render.
        { id: "r-2", cellRef: "api/ghost", value: 7, unit: "ms", source: "autoheal", trustVerdict: 2, bandVerdict: 1 },
      ],
      sources: [],
    });
    mocks.fetchTrust.mockResolvedValue({ trust: undefined });

    renderPage(<ConditionPage />);

    const ghostRow = (await screen.findByRole("rowheader", { name: /api\/ghost/ })).closest("tr");
    expect(ghostRow).not.toBeNull();
    expect(ghostRow).toHaveTextContent(strings.instrument.band.notEvaluated);
    expect(ghostRow).toHaveTextContent(strings.pages.condition.untrustedNotBanded);
    expect(ghostRow).not.toHaveTextContent(strings.instrument.band.inBand);
    // The trusted row keeps its band verdict.
    const trustedRow = screen.getByRole("rowheader", { name: /api\/latency/ }).closest("tr");
    expect(trustedRow).toHaveTextContent(strings.instrument.band.inBand);
  });

  it("prints no value for a reading whose source could not answer", async () => {
    mocks.fetchCondition.mockResolvedValue({
      // UNAVAILABLE trust: `value: 0` is the wire default, not a reading.
      readings: [{ id: "r-1", cellRef: "api/latency", value: 0, unit: "ms", source: "autoheal", trustVerdict: 6, bandVerdict: 0, unavailableReason: "sensor not reachable" }],
      sources: [],
    });
    mocks.fetchTrust.mockResolvedValue({ trust: undefined });

    renderPage(<ConditionPage />);

    const row = (await screen.findByRole("rowheader", { name: /api\/latency/ })).closest("tr");
    expect(row).toHaveTextContent("—");
    expect(row).not.toHaveTextContent(/\b0\b/);
    expect(row).toHaveTextContent(/sensor not reachable/);
    expect(row).toHaveTextContent(strings.instrument.trust.unavailable);
  });

  it("reports a condition source error without inferring health", async () => {
    mocks.fetchCondition.mockRejectedValue(new Error("condition down"));
    mocks.fetchTrust.mockRejectedValue(new Error("trust down"));
    renderPage(<ConditionPage />);
    expect(await screen.findByText(strings.pages.condition.unavailableTitle)).toBeInTheDocument();
    expect(screen.getAllByText(strings.pages.condition.unavailableBody).length).toBeGreaterThan(0);
  });

  it("renders coverage ratios, dated open loops, and integrity findings", async () => {
    mocks.fetchCoverage.mockResolvedValue({
      projections: [readProjection],
      integrityFindings: [{ code: "MISSING_TARGET", location: "api/latency", message: "target is absent" }],
    });
    mocks.fetchCells.mockResolvedValue({ cells: [{ projection: 1, id: "api/latency", status: 3, question: "Can latency be read?", gapOpenedOn: "2026-08-01", gapOpenDays: 19 }] });

    renderPage(<CoveragePage />);

    // The space is named by its own identifier, not a translated label.
    expect(await screen.findByRole("heading", { name: /supervision/ })).toBeInTheDocument();
    const openLoop = screen.getByTestId("coverage-open-loop");
    expect(within(openLoop).getByRole("cell", { name: /Can latency be read\?/ })).toBeInTheDocument();
    expect(within(screen.getByTestId("coverage-integrity")).getByText(/MISSING_TARGET/)).toBeInTheDocument();
  });

  it("keeps every ratio adjacent to its denominator confidence and rationale", async () => {
    mocks.fetchCoverage.mockResolvedValue({ projections: [readProjection], integrityFindings: [] });
    mocks.fetchCells.mockResolvedValue({ cells: [] });

    renderPage(<CoveragePage />);

    const ratio = await screen.findByRole("region", { name: strings.instrument.ratioLabel });
    expect(ratio).toHaveTextContent("50%");
    expect(ratio).toHaveTextContent(strings.instrument.confidence.authoritative);
    expect(ratio).toHaveTextContent(/owner-authored/);
  });

  it("never renders a fabricated zero for a space that could not be read", async () => {
    mocks.fetchCoverage.mockResolvedValue({ projections: [unreadProjection], integrityFindings: [] });
    mocks.fetchCells.mockResolvedValue({ cells: [] });

    renderPage(<CoveragePage />);

    await screen.findByRole("heading", { name: /availability/ });
    const grid = screen.getByTestId("coverage-grid");
    // Every counter the wire zeroed renders as an em dash instead.
    expect(within(grid).getAllByLabelText(strings.instrument.notAvailable).length).toBeGreaterThanOrEqual(3);
    expect(within(grid).queryByText(/^0$/)).toBeNull();
    // The ratio is withheld, not printed as 0%, and it still carries its
    // confidence and the reason the space could not be read.
    const ratio = within(grid).getByRole("region", { name: strings.instrument.ratioUncomputedLabel });
    expect(ratio).not.toHaveTextContent("%");
    // The reason the space could not be read travels with its lamp.
    expect(within(grid).getByRole("img", { name: /source unavailable/ })).toBeInTheDocument();
    expect(within(grid).getByText(strings.pages.coverage.projectionUnavailable)).toBeInTheDocument();
    // The unread space is named on the instrument chrome as well, so the
    // outage cannot be mistaken for a coverage collapse.
    expect(screen.getAllByRole("img", { name: /availability/ }).length).toBeGreaterThan(0);
  });

  it("dates every open-loop cell and never ages an undated gap", async () => {
    mocks.fetchCoverage.mockResolvedValue({ projections: [readProjection], integrityFindings: [] });
    mocks.fetchCells.mockResolvedValue({
      cells: [
        { projection: 1, id: "api/latency", status: 3, question: "Can latency be read?", gapOpenedOn: "2026-08-01", gapOpenDays: 19 },
        // No date, so `gap_open_days: 0` is a default rather than an age.
        { projection: 1, id: "api/undated", status: 3, question: "Is anything watching?", gapOpenedOn: "", gapOpenDays: 0 },
      ],
    });

    renderPage(<CoveragePage />);

    await screen.findByRole("rowheader", { name: /api\/undated/ });
    const openLoop = screen.getByTestId("coverage-open-loop");
    const datedRow = within(openLoop).getByRole("rowheader", { name: /api\/latency/ }).closest("tr");
    expect(datedRow).toHaveTextContent(/2026-08-01/);
    expect(datedRow).toHaveTextContent(strings.pages.coverage.ageDays);

    const undatedRow = within(openLoop).getByRole("rowheader", { name: /api\/undated/ }).closest("tr");
    expect(undatedRow).toHaveTextContent(strings.pages.coverage.undated);
    expect(undatedRow).not.toHaveTextContent(strings.pages.coverage.ageDays);
    expect(undatedRow).toHaveTextContent("—");

    // The undated gap leads: a gap nobody can put a clock on is the thing this
    // instrument exists to surface, not to bury under the dated ones.
    const rowHeaders = within(openLoop).getAllByRole("rowheader");
    expect(rowHeaders[0]).toHaveTextContent(/api\/undated/);
  });

  it("reports a coverage source error", async () => {
    mocks.fetchCoverage.mockRejectedValue(new Error("coverage down"));
    mocks.fetchCells.mockRejectedValue(new Error("cells down"));
    renderPage(<CoveragePage />);
    expect(await screen.findAllByText(strings.pages.coverage.unavailableTitle)).not.toHaveLength(0);
    expect(screen.getAllByText(strings.pages.coverage.unavailableBody).length).toBeGreaterThan(0);
    expect(screen.getByText(strings.pages.coverage.openLoopUnavailable)).toBeInTheDocument();
  });

  it("renders ranked focus findings and source health", async () => {
    mocks.fetchFocus.mockResolvedValue({
      allSourcesUnavailable: false,
      noFindings: false,
      sources: [{ id: "condition", label: "Condition", available: true, findingCount: 1 }],
      findings: [{ id: "f-1", title: "Read latency", source: "condition", message: "Verify the sensor", rationale: { rank: 1, cascadeStage: "condition", explanation: "first failing stage" } }],
    });
    renderPage(<FocusPage />);
    expect(await screen.findByRole("heading", { name: /Read latency/ })).toBeInTheDocument();
    expect(within(screen.getByTestId("focus-sources")).getByText(/Condition/)).toBeInTheDocument();
    // The cascade stage that ranked the finding is stated on the finding,
    // beside the explanation of why that stage put it there.
    expect(
      within(screen.getByTestId("focus-surface")).getByText(/condition · first failing stage/),
    ).toBeInTheDocument();
    expect(within(screen.getByTestId("focus-surface")).getByText(strings.pages.focus.rankLabel)).toBeInTheDocument();
  });

  it("reports a focus source error", async () => {
    mocks.fetchFocus.mockRejectedValue(new Error("focus down"));
    renderPage(<FocusPage />);
    expect(await screen.findByText(strings.pages.focus.unavailableTitle)).toBeInTheDocument();
    expect(screen.getAllByText(strings.pages.focus.unavailableBody).length).toBeGreaterThan(0);
  });

  it("names empty and unavailable reliability data explicitly", async () => {
    mocks.fetchCondition.mockResolvedValue({ readings: [], sources: [] });
    mocks.fetchTrust.mockResolvedValue({ trust: undefined });
    renderPage(<ConditionPage />);
    expect(await screen.findByText(strings.pages.condition.emptyTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.condition.noSourceReportTitle)).toBeInTheDocument();

    cleanup();
    mocks.fetchCoverage.mockResolvedValue({ projections: [unreadProjection], integrityFindings: [] });
    mocks.fetchCells.mockResolvedValue({ cells: [] });
    renderPage(<CoveragePage />);
    expect(await screen.findByText(strings.pages.coverage.openLoopEmptyTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.coverage.integrityEmptyTitle)).toBeInTheDocument();

    cleanup();
    mocks.fetchFocus.mockResolvedValue({
      allSourcesUnavailable: true,
      noFindings: false,
      sources: [{ id: "condition", label: "Condition", available: false, findingCount: 0, reason: "offline" }],
      findings: [],
    });
    renderPage(<FocusPage />);
    // Unread is not empty: the page says nothing could be READ.
    expect(await screen.findByText(strings.pages.focus.nothingReadTitle)).toBeInTheDocument();
    expect(within(screen.getByTestId("focus-sources")).getByRole("img", { name: /offline/ })).toBeInTheDocument();

    cleanup();
    mocks.fetchFocus.mockResolvedValue({
      allSourcesUnavailable: false,
      noFindings: true,
      sources: [{ id: "condition", label: "Condition", available: true, findingCount: 0 }],
      findings: [],
    });
    renderPage(<FocusPage />);
    // Empty IS empty, and it states how much of the surface was read.
    expect(await screen.findByText(strings.pages.focus.emptyTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.focus.emptyBody)).toBeInTheDocument();
  });
});
