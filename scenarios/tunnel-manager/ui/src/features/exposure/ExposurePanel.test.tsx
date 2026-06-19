/**
 * ExposurePanel tests — the primary operations surface.
 *
 * Mocks `api/exposure` via the shared builder; asserts the three query states,
 * the core/leased table rendering, and the expose / extend / revoke actions.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeExposureMocks, makeExposure, makeLeasedExposure } from "../../test-utils/mocks/exposure";

vi.mock("../../api/exposure", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/exposure")>();
  return { ...actual, ...makeExposureMocks() };
});

import { ExposurePanel } from "./ExposurePanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

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
    vi.mocked(exposureClient.listExposures).mockResolvedValueOnce({
      exposures: [makeExposure(), makeLeasedExposure()],
    } as never);

    renderWithProviders(<ExposurePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.exposure.table)).toBeInTheDocument();
    });
    expect(screen.getAllByTestId(selectors.exposure.row)).toHaveLength(2);
    expect(screen.getAllByTestId(selectors.exposure.tierBadge)[0]?.textContent).toContain("Core");
    expect(screen.getAllByTestId(selectors.exposure.tierBadge)[1]?.textContent).toContain("Leased");
    // Leased row carries an expiry; core row shows the placeholder.
    const expiries = screen.getAllByTestId(selectors.exposure.leaseExpiry);
    expect(expiries.some((el) => el.textContent.includes("Expires"))).toBe(true);
  });

  it("calls expose with the entered scenario name", async () => {
    const { exposureClient } = await import("../../api/exposure");
    const user = userEvent.setup();
    renderWithProviders(<ExposurePanel />);

    await user.type(screen.getByTestId(selectors.exposure.exposeInput), "image-tools");
    await user.click(screen.getByTestId(selectors.exposure.exposeButton));

    await waitFor(() => {
      expect(exposureClient.expose).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(exposureClient.expose).mock.calls[0]?.[0]).toMatchObject({ scenario: "image-tools" });
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
    await waitFor(() => expect(exposureClient.revokeLease).toHaveBeenCalledWith({ leaseId: "lease-1" }));
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
