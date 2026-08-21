/**
 * ExposurePanel tests — the primary operations surface.
 *
 * Mocks `api/exposure` via the shared builder; asserts the three query states,
 * the core/leased table rendering, and the expose / extend / revoke actions.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeExposureMocks, makeExposure, makeLeasedExposure } from "../../test-utils/mocks/exposure";
import { makeProbesMocks, makeRouteClassification } from "../../test-utils/mocks/probes";
import { makeRoute, makeRoutesMocks } from "../../test-utils/mocks/routes";

vi.mock("../../api/exposure", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/exposure")>();
  return { ...actual, ...makeExposureMocks() };
});

vi.mock("../../api/probes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/probes")>();
  return { ...actual, ...makeProbesMocks() };
});

vi.mock("../../api/routes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/routes")>();
  return { ...actual, ...makeRoutesMocks() };
});

import { ExposurePanel } from "./ExposurePanel";
import { selectors } from "../../consts/selectors";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import { PublicExposure } from "../../api/routes";

describe("ExposurePanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when no scenarios are exposed", async () => {
    renderWithProviders(<ExposurePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.exposure.table)).not.toBeInTheDocument();
  });

  it("renders core and leased rows with tier badges and lease expiry", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.listExposures).mockResolvedValue({
      exposures: [makeExposure(), makeLeasedExposure()],
    } as never);

    renderWithProviders(<ExposurePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.exposure.table)).toBeInTheDocument();
    });
    expect(screen.getAllByTestId(selectors.exposure.row)).toHaveLength(2);
    expect(screen.getAllByTestId(selectors.exposure.tierBadge)[0]?.textContent).toContain("Core");
    expect(screen.getAllByTestId(selectors.exposure.tierBadge)[1]?.textContent).toContain("Leased");
    expect(screen.getByTestId(selectors.exposure.coreCount)).toHaveTextContent("1");
    // Leased row carries an expiry; core row shows the placeholder.
    const expiries = screen.getAllByTestId(selectors.exposure.leaseExpiry);
    expect(expiries.some((el) => el.textContent.includes("Expires"))).toBe(true);
  });

  it("filters exposures by search text and tier", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.listExposures).mockResolvedValue({
      exposures: [makeExposure(), makeLeasedExposure()],
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);
    await waitFor(() => expect(screen.getAllByTestId(selectors.exposure.row)).toHaveLength(2));

    await user.type(screen.getByTestId(selectors.exposure.searchInput), "image");
    const rows = screen.getAllByTestId(selectors.exposure.row);
    expect(rows).toHaveLength(1);
    expect(rows[0]).toHaveTextContent("image-tools");

    await user.selectOptions(screen.getByTestId(selectors.exposure.tierFilter), "core");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.empty)).toHaveTextContent("No exposures match");
    });
  });

  it("shows classification badges from probe classifications", async () => {
    const { exposureClient } = await import("../../api/exposure");
    const { probesClient } = await import("../../api/probes");
    vi.mocked(exposureClient.listExposures).mockResolvedValue({
      exposures: [makeExposure()],
    } as never);
    vi.mocked(probesClient.classify).mockResolvedValue({
      classifications: [makeRouteClassification({ classification: 3, assessment: "local port is down" })],
    } as never);

    renderWithProviders(<ExposurePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.exposure.healthBadge)).toHaveTextContent("Scenario down");
    });
    expect(screen.getByTestId(selectors.exposure.unhealthyCount)).toHaveTextContent("1");
  });

  it("filters the collection by typed policy and readiness state", async () => {
    const { exposureClient } = await import("../../api/exposure");
    const { probesClient } = await import("../../api/probes");
    const { routesClient } = await import("../../api/routes");
    vi.mocked(exposureClient.listExposures).mockResolvedValue({
      exposures: [makeExposure(), makeLeasedExposure()],
    } as never);
    vi.mocked(routesClient.listRoutes).mockResolvedValue({
      routes: [
        makeRoute({ publicExposure: PublicExposure.ENABLED }),
        makeRoute({ id: "route-image", scenario: "image-tools", subdomain: "image-tools", publicUrl: "https://image-tools.itsagitime.com" }),
      ],
    } as never);
    vi.mocked(probesClient.classify).mockResolvedValue({
      classifications: [
        makeRouteClassification(),
        makeRouteClassification({ subdomain: "image-tools", classification: 3 }),
      ],
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);
    await waitFor(() => expect(screen.getAllByTestId(selectors.exposure.row)).toHaveLength(2));

    await user.selectOptions(screen.getByTestId(selectors.exposure.policyFilter), "public");
    expect(screen.getAllByTestId(selectors.exposure.row)).toHaveLength(1);
    expect(screen.getByTestId(selectors.exposure.row)).toHaveTextContent("agent-manager");

    await user.selectOptions(screen.getByTestId(selectors.exposure.policyFilter), "all");
    await user.selectOptions(screen.getByTestId(selectors.exposure.readinessFilter), "attention");
    expect(screen.getAllByTestId(selectors.exposure.row)).toHaveLength(1);
    expect(screen.getByTestId(selectors.exposure.row)).toHaveTextContent("image-tools");
  });

  it("requires selecting a known manifest route before exposing", async () => {
    const { exposureClient } = await import("../../api/exposure");
    const { probesClient } = await import("../../api/probes");
    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);

    expect(screen.getByTestId(selectors.exposure.exposeButton)).toBeDisabled();
    await waitFor(() => expect(screen.getByRole("option", { name: /agent-manager/ })).toBeInTheDocument());
    await user.selectOptions(screen.getByTestId(selectors.exposure.exposeInput), "agent-manager");
    await user.click(screen.getByTestId(selectors.exposure.exposeButton));
    expect(screen.getByTestId(selectors.exposure.reviewDialog)).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.config.publicExposureBody))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.exposure.confirmExposeButton)).toBeDisabled();
    await user.click(screen.getByTestId(selectors.exposure.policyAcknowledgement));
    await user.click(screen.getByTestId(selectors.exposure.confirmExposeButton));

    await waitFor(() => {
      expect(exposureClient.expose).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(exposureClient.expose).mock.calls[0]?.[0]).toMatchObject({
      scenario: "agent-manager",
      ttlSeconds: 604800n,
    });
    const classifyCallsBeforeVerification = vi.mocked(probesClient.classify).mock.calls.length;
    await user.click(screen.getByTestId(selectors.exposure.verifyResultButton));
    await waitFor(() => expect(vi.mocked(probesClient.classify).mock.calls.length).toBeGreaterThan(classifyCallsBeforeVerification));
    expect(screen.getAllByText(i18n.t(strings.exposure.verificationPolicy)).length).toBeGreaterThan(0);
  });

  it("opens an existing route detail when linked from the constellation", async () => {
    renderWithProviders(<ExposurePanel />, { initialEntries: ["/exposure?scenario=agent-manager"] });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.exposure.detailDialog)).toBeInTheDocument();
    });
    expect(screen.getByRole("heading", { name: "agent-manager" })).toBeInTheDocument();
    await userEvent.setup().click(screen.getByTestId(selectors.exposure.detailCloseButton));
    await waitFor(() => expect(screen.queryByTestId(selectors.exposure.detailDialog)).not.toBeInTheDocument());
  });

  it("closes layered exposure dialogs with Escape and restores body scrolling", async () => {
    const user = userEvent.setup();
    const originalOverflow = document.body.style.overflow;
    renderWithProviders(<ExposurePanel />, { initialEntries: ["/exposure?scenario=agent-manager"] });

    await waitFor(() => expect(screen.getByTestId(selectors.exposure.detailDialog)).toBeInTheDocument());
    expect(document.body.style.overflow).toBe("hidden");
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByTestId(selectors.exposure.detailDialog)).not.toBeInTheDocument());
    expect(document.body.style.overflow).toBe(originalOverflow);
  });

  it("opens expired lease detail and starts a re-expose review", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.listLeases).mockResolvedValueOnce({
      leases: [{
        id: "expired-lease",
        scenario: "agent-manager",
        requestedBy: "operator",
        expiresAt: { seconds: BigInt(1), nanos: 0 },
        status: 2,
      }],
    } as never);
    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.exposure.expiredPanel)).toBeInTheDocument());
    await user.click(within(screen.getByTestId(selectors.exposure.expiredPanel)).getByRole("button", { name: "agent-manager" }));
    expect(screen.getByTestId(selectors.exposure.detailDialog)).toBeInTheDocument();
    expect(screen.getByText(new URL(makeRoute().publicUrl).hostname)).toBeInTheDocument();
    await user.click(within(screen.getByTestId(selectors.exposure.detailDialog)).getByTestId(selectors.exposure.reExposeButton));
    expect(screen.getByTestId(selectors.exposure.reviewDialog)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "agent-manager" })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.exposure.confirmExposeButton)).toBeDisabled();
  });

  it("keeps the review open with classified retry guidance when provisioning fails", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.expose).mockRejectedValueOnce(new Error("port unavailable"));
    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);

    await waitFor(() => expect(screen.getByRole("option", { name: /agent-manager/ })).toBeInTheDocument());
    await user.selectOptions(screen.getByTestId(selectors.exposure.exposeInput), "agent-manager");
    await user.click(screen.getByTestId(selectors.exposure.exposeButton));
    await user.click(screen.getByTestId(selectors.exposure.policyAcknowledgement));
    await user.click(screen.getByTestId(selectors.exposure.confirmExposeButton));

    await waitFor(() => expect(screen.getByTestId(selectors.exposure.exposeError)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.exposure.reviewDialog)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.exposure.confirmExposeButton)).toHaveTextContent("Retry exposure");
  });

  it("supports a bounded custom duration and shows the projected expiry", async () => {
    const { exposureClient } = await import("../../api/exposure");
    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);

    await waitFor(() => expect(screen.getByRole("option", { name: /agent-manager/ })).toBeInTheDocument());
    await user.selectOptions(screen.getByTestId(selectors.exposure.exposeInput), "agent-manager");
    await user.click(screen.getByTestId(selectors.exposure.exposeButton));
    await user.selectOptions(screen.getByTestId(selectors.exposure.durationSelect), "custom");
    await user.clear(screen.getByTestId(selectors.exposure.customDurationInput));
    await user.type(screen.getByTestId(selectors.exposure.customDurationInput), "2");
    expect(screen.getByText(i18n.t(strings.exposure.projectedExpiry))).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.exposure.policyAcknowledgement));
    await user.click(screen.getByTestId(selectors.exposure.confirmExposeButton));

    await waitFor(() => expect(exposureClient.expose).toHaveBeenCalledWith({ scenario: "agent-manager", ttlSeconds: 7200n }));
  });

  it("disables the expose button when the input is empty", () => {
    renderWithProviders(<ExposurePanel />);
    expect(screen.getByTestId(selectors.exposure.exposeButton)).toBeDisabled();
  });

  it("extends and revokes a lease through the row actions", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.listExposures).mockResolvedValue({
      exposures: [makeLeasedExposure()],
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.exposure.extendButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.exposure.extendButton));
    await waitFor(() => expect(exposureClient.extendLease).toHaveBeenCalledWith({ leaseId: "lease-1" }));

    await user.click(screen.getByTestId(selectors.exposure.revokeButton));
    await user.click(screen.getByTestId(selectors.exposure.revokeConfirmButton));
    await waitFor(() => expect(exposureClient.revokeLease).toHaveBeenCalledWith({ leaseId: "lease-1" }));
  });

  it("keeps lease actions available from route detail", async () => {
    const { exposureClient } = await import("../../api/exposure");
    const { probesClient } = await import("../../api/probes");
    vi.mocked(exposureClient.listExposures).mockResolvedValue({ exposures: [makeLeasedExposure()] } as never);
    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);

    await waitFor(() => expect(screen.getAllByTestId(selectors.exposure.row)).toHaveLength(1));
    const scenarioButton = screen.getByTestId(selectors.exposure.row).querySelector("button");
    expect(scenarioButton).not.toBeNull();
    await user.click(scenarioButton!);
    expect(screen.getByTestId(selectors.exposure.detailDialog)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.exposure.detailProbeButton));
    await waitFor(() => expect(probesClient.runProbes).toHaveBeenCalledTimes(1));
    await user.click(screen.getByTestId(selectors.exposure.detailExtendButton));
    await waitFor(() => expect(exposureClient.extendLease).toHaveBeenCalledWith({ leaseId: "lease-1" }));
    await user.click(screen.getByTestId(selectors.exposure.detailRevokeButton));
    await user.click(screen.getByTestId(selectors.exposure.revokeConfirmButton));
    await waitFor(() => expect(exposureClient.revokeLease).toHaveBeenCalledWith({ leaseId: "lease-1" }));
  });

  it("runs reconcile and renders the operator result", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.reconcile).mockResolvedValueOnce({
      coreEnsured: 2,
      leasesReaped: 1,
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);
    await user.click(screen.getByTestId(selectors.exposure.reconcileButton));

    await waitFor(() => {
      expect(exposureClient.reconcile).toHaveBeenCalledTimes(1);
    });
    expect(screen.getByTestId(selectors.exposure.reconcileResult)).toHaveTextContent(
      "Core ensured: 2 · leases reaped: 1",
    );
  });

  it("surfaces the error state when listExposures rejects", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.listExposures).mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<ExposurePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.error)).toBeInTheDocument();
    });
  });
});
