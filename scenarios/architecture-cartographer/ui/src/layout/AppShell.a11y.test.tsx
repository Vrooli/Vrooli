/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";

vi.mock("../api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn().mockResolvedValue({ snapshots: [], nextPageToken: "" }),
    extractGraph: vi.fn().mockResolvedValue({ snapshot: undefined, fromCache: false }),
  },
}));
vi.mock("../api/health", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "ok",
    service: "architecture-cartographer-api",
    timestamp: new Date(0).toISOString(),
  }),
}));

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
    // Load the overview's live snapshot panel explicitly, then wait for the
    // query to settle so axe scans a stable DOM. Without this, useQuery's
    // resolution races the test teardown and surfaces as an unwrapped-act
    // warning.
    fireEvent.click(screen.getByRole("button", { name: "Load snapshots" }));
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.features.targets.activeSnapshots.empty),
      ).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
