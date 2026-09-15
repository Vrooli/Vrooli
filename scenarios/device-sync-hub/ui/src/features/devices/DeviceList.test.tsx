import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders, makeDevice } from "../../test-utils";
import { TrustState } from "../../api/devices";
import { strings } from "../../consts/strings";

const { listDevices, renameDevice, revokeDevice, approvePairing } = vi.hoisted(() => ({
  listDevices: vi.fn(),
  renameDevice: vi.fn(),
  revokeDevice: vi.fn(),
  approvePairing: vi.fn(),
}));

vi.mock("../../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/devices")>();
  return {
    ...actual,
    devicesClient: { listDevices, renameDevice, revokeDevice, approvePairing },
  };
});

import { DeviceList } from "./DeviceList";
import { selectors } from "../../consts/selectors";

describe("DeviceList", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists devices and exposes approve only for pending ones", async () => {
    listDevices.mockResolvedValue({
      devices: [
        makeDevice({ id: "trusted-1", trustState: TrustState.TRUSTED }),
        makeDevice({ id: "pending-1", trustState: TrustState.PENDING }),
      ],
    });
    renderWithProviders(<DeviceList />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.devices.row({ id: "trusted-1" }))).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.devices.approve({ id: "pending-1" }))).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.devices.approve({ id: "trusted-1" }))).not.toBeInTheDocument();
  });

  it("revokes a device only after confirmation", async () => {
    const user = userEvent.setup();
    listDevices.mockResolvedValue({ devices: [makeDevice({ id: "d1", name: "Old Phone" })] });
    revokeDevice.mockResolvedValue({});
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderWithProviders(<DeviceList />);

    await waitFor(() => expect(screen.getByTestId(selectors.devices.row({ id: "d1" }))).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.devices.revoke({ id: "d1" })));

    expect(confirmSpy).toHaveBeenCalled();
    await waitFor(() => expect(revokeDevice).toHaveBeenCalledWith({ deviceId: "d1" }));
    confirmSpy.mockRestore();
  });

  it("does not revoke when the confirm is dismissed", async () => {
    const user = userEvent.setup();
    listDevices.mockResolvedValue({ devices: [makeDevice({ id: "d1" })] });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderWithProviders(<DeviceList />);

    await waitFor(() => expect(screen.getByTestId(selectors.devices.row({ id: "d1" }))).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.devices.revoke({ id: "d1" })));

    expect(revokeDevice).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("renames a device when the draft is non-empty", async () => {
    const user = userEvent.setup();
    listDevices.mockResolvedValue({ devices: [makeDevice({ id: "d1", name: "Old" })] });
    renameDevice.mockResolvedValue({});
    renderWithProviders(<DeviceList />);

    await waitFor(() => expect(screen.getByTestId(selectors.devices.row({ id: "d1" }))).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.devices.rename({ id: "d1" })));

    const input = screen.getByLabelText(strings.devices.renameLabel);
    await user.clear(input);
    await user.type(input, "New Name");
    await user.click(screen.getByText(strings.devices.renameSave));

    await waitFor(() =>
      expect(renameDevice).toHaveBeenCalledWith({ deviceId: "d1", name: "New Name" }),
    );
  });

  it("discards a rename when the draft is blank", async () => {
    const user = userEvent.setup();
    listDevices.mockResolvedValue({ devices: [makeDevice({ id: "d1", name: "Old" })] });
    renderWithProviders(<DeviceList />);

    await waitFor(() => expect(screen.getByTestId(selectors.devices.row({ id: "d1" }))).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.devices.rename({ id: "d1" })));
    await user.clear(screen.getByLabelText(strings.devices.renameLabel));
    await user.click(screen.getByText(strings.devices.renameSave));

    expect(renameDevice).not.toHaveBeenCalled();
    // Editing closes regardless: the rename pencil is back.
    expect(screen.getByTestId(selectors.devices.rename({ id: "d1" }))).toBeInTheDocument();
  });

  it("approves a pending device", async () => {
    const user = userEvent.setup();
    listDevices.mockResolvedValue({ devices: [makeDevice({ id: "p1", trustState: TrustState.PENDING })] });
    approvePairing.mockResolvedValue({});
    renderWithProviders(<DeviceList />);

    await waitFor(() => expect(screen.getByTestId(selectors.devices.approve({ id: "p1" }))).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.devices.approve({ id: "p1" })));
    await waitFor(() => expect(approvePairing).toHaveBeenCalledWith({ deviceId: "p1" }));
  });

  it("hides the revoke action for an already-revoked device", async () => {
    listDevices.mockResolvedValue({ devices: [makeDevice({ id: "r1", trustState: TrustState.REVOKED })] });
    renderWithProviders(<DeviceList />);

    await waitFor(() => expect(screen.getByTestId(selectors.devices.row({ id: "r1" }))).toBeInTheDocument());
    expect(screen.queryByTestId(selectors.devices.revoke({ id: "r1" }))).not.toBeInTheDocument();
  });

  it("renders presence + last-seen metadata for an offline device", async () => {
    listDevices.mockResolvedValue({
      devices: [
        makeDevice({
          id: "off1",
          online: false,
          lastSeenAt: timestampFromDate(new Date("2024-01-02T03:04:00Z")),
        }),
      ],
    });
    renderWithProviders(<DeviceList />);

    await waitFor(() => expect(screen.getByTestId(selectors.devices.row({ id: "off1" }))).toBeInTheDocument());
    expect(screen.getByLabelText(strings.devices.offlineLabel)).toBeInTheDocument();
  });

  it("renders the empty state when there are no devices", async () => {
    listDevices.mockResolvedValue({ devices: [] });
    renderWithProviders(<DeviceList />);
    await waitFor(() => expect(screen.getByTestId(selectors.devices.empty)).toBeInTheDocument());
  });

  it("renders an error message when the list query fails", async () => {
    listDevices.mockRejectedValue(new ConnectError("denied", Code.Unavailable));
    renderWithProviders(<DeviceList />);
    await waitFor(() =>
      expect(screen.getByText(strings.errors.unavailable)).toBeInTheDocument(),
    );
  });
});
