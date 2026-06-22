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

import { DashboardView } from "./DashboardView";

describe("DashboardView accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    getInventory.mockResolvedValue(
      makeScanFleetResponse({
        scenarioCount: 2,
        isolationUnreadyCount: 1,
        scannedAt: new Date(Date.now() - 60_000).toISOString(),
        engineDistribution: [{ engine: "sqlite", scenarioCount: 2 } as never],
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

  it("renders the populated dashboard without axe violations", async () => {
    const { container } = renderWithProviders(<DashboardView />);
    await waitFor(() => expect(screen.getByTestId(selectors.dashboard.band)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
