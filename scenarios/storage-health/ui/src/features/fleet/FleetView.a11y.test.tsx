import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeFleetEntry, makeScanFleetResponse } from "../storage/mocks/factories";

const { getInventory } = vi.hoisted(() => ({ getInventory: vi.fn() }));

vi.mock("../../api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/storage")>();
  return { ...actual, storageClient: { ...actual.storageClient, getInventory } };
});

import { FleetView } from "./FleetView";

describe("FleetView accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    getInventory.mockResolvedValue(
      makeScanFleetResponse({
        scenarioCount: 2,
        entries: [
          makeFleetEntry({ scenario: "ready" }),
          makeFleetEntry({ scenario: "risky", isolationReady: false, isolationReason: "unwired" }),
        ],
      }),
    );
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the inventory table without axe violations", async () => {
    const { container } = renderWithProviders(<FleetView />, { routerEntries: ["/fleet"] });
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.table)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
