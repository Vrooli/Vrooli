/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { makeScanFleetResponse } from "../features/storage/mocks/factories";

// The index route is the data-backed dashboard; stub the client so the shell
// a11y scan renders deterministically without a live transport.
vi.mock("../api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/storage")>();
  return {
    ...actual,
    storageClient: {
      ...actual.storageClient,
      getInventory: vi.fn().mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 })),
      scanFleet: vi.fn().mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 })),
    },
  };
});

import { TestAppRouter } from "../app/routes";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );
    // Let the dashboard's inventory query settle so the scan sees the final
    // (empty-CTA) tree rather than the loading skeleton.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.dashboard.empty)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
