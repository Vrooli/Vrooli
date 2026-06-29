/**
 * AccessPanel tests — the read-only /public Access-bypass status surface.
 * Renders the global switch + configured badges, the dry-run plan, and the
 * per-host table; asserts it issues GetAccessStatus and never a mutation.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeConfigMocks, makeAccessStatusResponse } from "../../test-utils/mocks/config";

vi.mock("../../api/config", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/config")>();
  return { ...actual, ...makeConfigMocks() };
});

import { AccessPanel } from "./AccessPanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("AccessPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the disabled/unconfigured summary and per-host row from GetAccessStatus", async () => {
    const { configClient } = await import("../../api/config");
    renderWithProviders(<AccessPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.access.table)).toBeInTheDocument();
    });
    expect(configClient.getAccessStatus).toHaveBeenCalled();
    expect(screen.getByTestId(selectors.access.globalBadge)).toHaveTextContent("disabled");
    expect(screen.getByTestId(selectors.access.configuredBadge)).toHaveTextContent("not configured");
    expect(screen.getByTestId(selectors.access.hostName)).toHaveTextContent("web-console.itsagitime.com");
    expect(screen.getByTestId(selectors.access.bypassBadge)).toHaveTextContent("Off");
    expect(screen.getByTestId(selectors.access.note)).toHaveTextContent("read-only");
  });

  it("renders the dry-run plan when reconcile would create or remove bypass apps", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getAccessStatus).mockResolvedValueOnce(
      makeAccessStatusResponse({
        status: {
          enabled: true,
          configured: true,
          hosts: [
            {
              host: "web-console.itsagitime.com",
              override: "enabled",
              effectiveBypass: true,
              managed: false,
              appId: "",
            },
          ],
          toCreate: ["web-console.itsagitime.com"],
          toRemove: ["stale.itsagitime.com"],
        },
      }),
    );

    renderWithProviders(<AccessPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.access.planCreate)).toHaveTextContent("web-console.itsagitime.com");
    });
    expect(screen.getByTestId(selectors.access.globalBadge)).toHaveTextContent("enabled");
    expect(screen.getByTestId(selectors.access.configuredBadge)).toHaveTextContent("configured");
    expect(screen.getByTestId(selectors.access.planRemove)).toHaveTextContent("stale.itsagitime.com");
    expect(screen.getByTestId(selectors.access.bypassBadge)).toHaveTextContent("On");
    expect(screen.getByTestId(selectors.access.overrideBadge)).toHaveTextContent("Enabled");
  });

  it("shows the empty state when no hosts are reported", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getAccessStatus).mockResolvedValueOnce(
      makeAccessStatusResponse({ status: { enabled: false, configured: false, hosts: [], toCreate: [], toRemove: [] } }),
    );

    renderWithProviders(<AccessPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.empty)).toHaveTextContent("No exposed hosts");
    });
  });

  it("refetches status on refresh without issuing any mutation", async () => {
    const { configClient } = await import("../../api/config");
    const user = userEvent.setup();
    renderWithProviders(<AccessPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.access.table)).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.access.refreshButton));

    await waitFor(() => {
      expect(vi.mocked(configClient.getAccessStatus).mock.calls.length).toBeGreaterThanOrEqual(2);
    });
    expect(configClient.setPublicExposure).not.toHaveBeenCalled();
  });
});
