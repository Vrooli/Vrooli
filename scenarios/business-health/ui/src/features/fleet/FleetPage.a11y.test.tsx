/**
 * FleetPage accessibility regression — the loaded worst-first table (tiles,
 * filters, sortable headers, clickable rows) must be axe-clean under a real
 * locale.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/fleet", () => ({
  fleetClient: { scanFleet: vi.fn() },
}));

import { FleetPage } from "./FleetPage";
import { fleetClient } from "../../api/fleet";
import { makeScanFleetResponse, makeFleetEntry, makeFleetScanError } from "./mocks/factories";

describe("FleetPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(
      makeScanFleetResponse({
        scenarioCount: 2,
        passingCount: 1,
        templateLaggardCount: 1,
        entries: [
          makeFleetEntry({ scenario: "alpha", debtScore: 40, passed: false, templateLaggard: true }),
          makeFleetEntry({ scenario: "beta", debtScore: 5 }),
        ],
        errors: [makeFleetScanError()],
      }),
    );
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("has no violations for the loaded fleet table", async () => {
    const { container } = renderWithProviders(<FleetPage />);
    await screen.findByTestId(selectors.fleet.table);
    await expectNoA11yViolations(container);
  });
});
