/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";

vi.mock("../api/health", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/health")>()),
  fetchHealth: vi.fn(),
}));
vi.mock("../api/runs", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/runs")>()),
  listTargetStatus: vi.fn(),
}));
vi.mock("../api/targets", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/targets")>()),
  listTargets: vi.fn(),
}));
vi.mock("../api/destinations", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/destinations")>()),
  listDestinations: vi.fn(),
}));
vi.mock("../api/discovery", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/discovery")>()),
  listTargetSuggestions: vi.fn(),
  listDestinationSuggestions: vi.fn(),
}));

import * as healthApi from "../api/health";
import * as runsApi from "../api/runs";
import * as targetsApi from "../api/targets";
import * as destinationsApi from "../api/destinations";
import * as discoveryApi from "../api/discovery";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(healthApi.fetchHealth).mockResolvedValue({
      status: "ok",
      service: "data-backup-manager",
      readiness: "ready",
    } as never);
    vi.mocked(runsApi.listTargetStatus).mockResolvedValue([]);
    vi.mocked(targetsApi.listTargets).mockResolvedValue([]);
    vi.mocked(destinationsApi.listDestinations).mockResolvedValue([]);
    vi.mocked(discoveryApi.listTargetSuggestions).mockResolvedValue([]);
    vi.mocked(discoveryApi.listDestinationSuggestions).mockResolvedValue([]);
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );
    await screen.findByTestId(selectors.overview.setupCta, undefined, { timeout: 5000 });
    // The full shell includes asynchronous posture and health queries. Let the
    // posture query settle before axe scans and cleanup runs, so its completion
    // cannot update the tree during teardown.
    await waitFor(() =>
      expect(screen.getByTestId(selectors.overview.postureHeadline)).not.toHaveTextContent(
        "Checking posture…",
      ),
    );
    await expectNoA11yViolations(container);
  });
});
