import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeDevice, seedSession } from "../../test-utils/session";
import { TrustState } from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";

const { redeemPairingCode, requestPairing, approvePairing } = vi.hoisted(() => ({
  redeemPairingCode: vi.fn(),
  requestPairing: vi.fn(),
  approvePairing: vi.fn(),
}));

vi.mock("../../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/devices")>();
  return { ...actual, devicesClient: { redeemPairingCode, requestPairing, approvePairing } };
});

import { JoinHubForm } from "./JoinHubForm";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { loadSession } from "../session/store";

describe("JoinHubForm", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("validates an empty code before calling redeem", async () => {
    const user = userEvent.setup();
    renderWithProviders(<JoinHubForm onBack={vi.fn()} />);
    await user.click(screen.getByTestId(selectors.join.redeemButton));
    expect(redeemPairingCode).not.toHaveBeenCalled();
    expect(screen.getByTestId(selectors.join.error)).toHaveTextContent(strings.join.missingCode);
  });

  it("redeems a code and persists the returned device credentials", async () => {
    const user = userEvent.setup();
    redeemPairingCode.mockResolvedValueOnce({
      deviceToken: "dt-redeem",
      device: makeDevice({ id: "dev-redeem" }),
    });
    renderWithProviders(<JoinHubForm onBack={vi.fn()} />);

    await user.type(screen.getByTestId(selectors.join.codeInput), "CODE-42");
    await user.click(screen.getByTestId(selectors.join.redeemButton));

    await waitFor(() => {
      expect(loadSession().deviceToken).toBe("dt-redeem");
    });
    expect(redeemPairingCode).toHaveBeenCalledWith(expect.objectContaining({ code: "CODE-42" }));
  });

  it("shows the waiting state after requesting approval", async () => {
    const user = userEvent.setup();
    requestPairing.mockResolvedValueOnce({
      deviceToken: "dt-pending",
      device: makeDevice({ id: "dev-pending" }),
    });
    renderWithProviders(<JoinHubForm onBack={vi.fn()} />);

    await user.type(screen.getByTestId(selectors.join.deviceNameInput), "My Phone");
    await user.click(screen.getByTestId(selectors.join.requestButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.join.waiting)).toBeInTheDocument();
    });
  });

  it("lets a signed-in owner approve this browser's pending pairing request", async () => {
    const user = userEvent.setup();
    const pending = makeDevice({ id: "dev-pending", trustState: TrustState.PENDING });
    seedSession({ deviceToken: "dt-pending", device: pending, ownerToken: "owner-jwt" });
    approvePairing.mockResolvedValueOnce({
      device: makeDevice({ id: "dev-pending", trustState: TrustState.TRUSTED }),
    });
    renderWithProviders(<JoinHubForm onBack={vi.fn()} />);

    await user.click(screen.getByTestId(selectors.join.approveThisDevice));

    await waitFor(() => {
      expect(approvePairing).toHaveBeenCalledWith({ deviceId: "dev-pending" });
      expect(loadSession().device?.trustState).toBe(TrustState.TRUSTED);
    });
  });

  it("calls onBack from the back button", async () => {
    const user = userEvent.setup();
    const onBack = vi.fn();
    renderWithProviders(<JoinHubForm onBack={onBack} />);
    await user.click(screen.getByTestId(selectors.onboarding.back));
    expect(onBack).toHaveBeenCalledOnce();
  });
});
