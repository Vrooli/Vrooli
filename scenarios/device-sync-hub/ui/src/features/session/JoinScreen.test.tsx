import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeDevice } from "../../test-utils/session";

const { redeemPairingCode, requestPairing } = vi.hoisted(() => ({
  redeemPairingCode: vi.fn(),
  requestPairing: vi.fn(),
}));

vi.mock("../../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/devices")>();
  return { ...actual, devicesClient: { redeemPairingCode, requestPairing } };
});

import { JoinScreen } from "./JoinScreen";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { loadSession } from "./store";

describe("JoinScreen", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("validates an empty code before calling redeem", async () => {
    const user = userEvent.setup();
    renderWithProviders(<JoinScreen />);
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
    renderWithProviders(<JoinScreen />);

    await user.type(screen.getByTestId(selectors.join.codeInput), "CODE-42");
    await user.click(screen.getByTestId(selectors.join.redeemButton));

    await waitFor(() => {
      expect(loadSession().deviceToken).toBe("dt-redeem");
    });
    expect(redeemPairingCode).toHaveBeenCalledWith(
      expect.objectContaining({ code: "CODE-42" }),
    );
  });

  it("shows the waiting state after requesting approval", async () => {
    const user = userEvent.setup();
    requestPairing.mockResolvedValueOnce({
      deviceToken: "dt-pending",
      device: makeDevice({ id: "dev-pending" }),
    });
    renderWithProviders(<JoinScreen />);

    await user.type(screen.getByTestId(selectors.join.deviceNameInput), "My Phone");
    await user.click(screen.getByTestId(selectors.join.requestButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.join.waiting)).toBeInTheDocument();
    });
  });
});
