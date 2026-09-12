import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { PairingRequestSchema } from "@vrooli/proto-types/device-sync-hub/v1/realtime/realtime_pb";

import { renderWithProviders } from "../../test-utils";

const { approvePairing } = vi.hoisted(() => ({ approvePairing: vi.fn() }));

vi.mock("../../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/devices")>();
  return { ...actual, devicesClient: { approvePairing } };
});

import { PendingPairingBanner } from "./PendingPairingBanner";
import { selectors } from "../../consts/selectors";

const pending = create(PairingRequestSchema, { deviceId: "dev-9", name: "New Phone", kind: "phone" });

describe("PendingPairingBanner", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("approves the pending device then dismisses on success", async () => {
    const user = userEvent.setup();
    approvePairing.mockResolvedValue({});
    const onDismiss = vi.fn();
    renderWithProviders(<PendingPairingBanner pending={pending} onDismiss={onDismiss} />);

    expect(screen.getByTestId(selectors.devices.pendingBanner)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: strings("approve") }));

    await waitFor(() => expect(approvePairing).toHaveBeenCalledWith({ deviceId: "dev-9" }));
    await waitFor(() => expect(onDismiss).toHaveBeenCalled());
  });

  it("dismisses locally without approving", async () => {
    const user = userEvent.setup();
    const onDismiss = vi.fn();
    renderWithProviders(<PendingPairingBanner pending={pending} onDismiss={onDismiss} />);

    await user.click(screen.getByRole("button", { name: strings("dismiss") }));

    expect(approvePairing).not.toHaveBeenCalled();
    expect(onDismiss).toHaveBeenCalled();
  });
});

// In cimode i18n renders the translation key, so buttons are labelled by key.
function strings(which: "approve" | "dismiss"): string {
  return which === "approve"
    ? "devices.pendingBanner.approve"
    : "devices.pendingBanner.dismiss";
}
