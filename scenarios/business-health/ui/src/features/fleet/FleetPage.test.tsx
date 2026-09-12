/**
 * FleetPage tests — auto-scan on mount → worst-first table → filter/text narrow
 * → row-click deep-link into the matrix. The connect client and the router's
 * `useNavigate` are mocked at the module boundary so tests assert component
 * behavior against fixture proto messages, not the network or real navigation.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { setLocale } from "../../i18n";

const { navigateMock } = vi.hoisted(() => ({ navigateMock: vi.fn() }));

vi.mock("../../api/fleet", () => ({
  fleetClient: { scanFleet: vi.fn() },
}));

vi.mock("react-router-dom", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router-dom")>()),
  useNavigate: () => navigateMock,
}));

import { FleetPage } from "./FleetPage";
import { fleetClient } from "../../api/fleet";
import { makeScanFleetResponse, makeFleetEntry, makeFleetScanError } from "./mocks/factories";

const populated = () =>
  makeScanFleetResponse({
    scenarioCount: 3,
    passingCount: 1,
    starterRegistryCount: 1,
    templateLaggardCount: 1,
    entries: [
      makeFleetEntry({ scenario: "beta", debtScore: 30, passed: false, templateLaggard: true }),
      makeFleetEntry({ scenario: "gamma", debtScore: 10, unprovenClaims: 3 }),
      makeFleetEntry({ scenario: "alpha", debtScore: 50, passed: false, starterRegistry: true }),
    ],
  });

describe("FleetPage", () => {
  beforeEach(() => {
    vi.mocked(fleetClient.scanFleet).mockReset();
    navigateMock.mockReset();
  });
  afterEach(async () => {
    cleanup();
    await setLocale("en");
  });

  it("shows a loading state while the scan is in flight", async () => {
    vi.mocked(fleetClient.scanFleet).mockReturnValue(new Promise(() => {}) as never);
    renderWithProviders(<FleetPage />);
    expect(await screen.findByTestId(selectors.fleet.loading)).toBeInTheDocument();
  });

  it("renders an error alert when the scan fails", async () => {
    vi.mocked(fleetClient.scanFleet).mockRejectedValue(new Error("boom"));
    renderWithProviders(<FleetPage />);
    expect(await screen.findByTestId(selectors.fleet.error)).toBeInTheDocument();
  });

  it("renders the empty state when no scenarios were discovered", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(
      makeScanFleetResponse({ scenarioCount: 0, passingCount: 0, entries: [] }),
    );
    renderWithProviders(<FleetPage />);
    expect(await screen.findByTestId(selectors.fleet.empty)).toBeInTheDocument();
  });

  it("renders an unstamped template version and toggles sort direction", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(
      makeScanFleetResponse({
        scenarioCount: 1,
        passingCount: 0,
        entries: [
          makeFleetEntry({ scenario: "solo", templateVersion: "", orphanedTargets: 2, debtScore: 7 }),
        ],
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<FleetPage />);

    const row = await screen.findByTestId(selectors.fleet.row({ scenario: "solo" }));
    // Unstamped template renders the em-dash placeholder (no letters -> literal ok).
    expect(within(row).getByText("—")).toBeInTheDocument();

    // Toggle the default-active debt column from descending to ascending.
    await user.click(screen.getByRole("button", { name: strings.fleet.column.debt }));
    expect(screen.getByTestId(selectors.fleet.row({ scenario: "solo" }))).toBeInTheDocument();
  });

  it("renders the summary tiles and table worst-first by debt", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(populated());
    renderWithProviders(<FleetPage />);

    expect(await screen.findByTestId(selectors.fleet.table)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.fleet.tiles)).toBeInTheDocument();

    const worst = screen.getByTestId(selectors.fleet.row({ scenario: "alpha" }));
    const middle = screen.getByTestId(selectors.fleet.row({ scenario: "beta" }));
    const best = screen.getByTestId(selectors.fleet.row({ scenario: "gamma" }));
    // Worst-first: alpha(50) precedes beta(30) precedes gamma(10) in the DOM.
    expect(worst.compareDocumentPosition(middle) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(middle.compareDocumentPosition(best) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("narrows rows when a filter toggle is enabled", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(populated());
    const user = userEvent.setup();
    renderWithProviders(<FleetPage />);

    await screen.findByTestId(selectors.fleet.table);
    await user.click(screen.getByTestId(selectors.fleet.filterLaggard));

    expect(screen.getByTestId(selectors.fleet.row({ scenario: "beta" }))).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.fleet.row({ scenario: "alpha" }))).not.toBeInTheDocument();
  });

  it("filters by scenario substring from the text box", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(populated());
    const user = userEvent.setup();
    renderWithProviders(<FleetPage />);

    await screen.findByTestId(selectors.fleet.table);
    await user.type(screen.getByTestId(selectors.fleet.filterText), "gam");

    expect(screen.getByTestId(selectors.fleet.row({ scenario: "gamma" }))).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.fleet.row({ scenario: "alpha" }))).not.toBeInTheDocument();
  });

  it("lists ungradeable scenarios", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(
      makeScanFleetResponse({
        scenarioCount: 1,
        entries: [makeFleetEntry()],
        errors: [makeFleetScanError({ scenario: "broken", reason: "no service.json" })],
      }),
    );
    renderWithProviders(<FleetPage />);

    const errors = await screen.findByTestId(selectors.fleet.errors);
    expect(within(errors).getByText(strings.fleet.errorsHeading)).toBeInTheDocument();
  });

  it("deep-links into the matrix when a row is clicked", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(populated());
    const user = userEvent.setup();
    renderWithProviders(<FleetPage />);

    const row = await screen.findByTestId(selectors.fleet.row({ scenario: "alpha" }));
    await user.click(within(row).getByRole("button"));

    expect(navigateMock).toHaveBeenCalledWith("/matrix?scenario=alpha");
  });

  it("renders localized copy under a real locale", async () => {
    await setLocale("ja");
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(populated());
    renderWithProviders(<FleetPage />);
    expect(await screen.findByTestId(selectors.fleet.table)).toBeInTheDocument();
  });
});
