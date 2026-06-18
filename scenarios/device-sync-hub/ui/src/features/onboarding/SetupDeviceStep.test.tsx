import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { makeDevice } from "../../test-utils/session";
import { strings } from "../../consts/strings";

const { setupOwnerDevice } = vi.hoisted(() => ({ setupOwnerDevice: vi.fn() }));

vi.mock("../../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/devices")>();
  return { ...actual, devicesClient: { setupOwnerDevice } };
});

import { SetupDeviceStep } from "./SetupDeviceStep";
import { selectors } from "../../consts/selectors";
import { saveSession, loadSession, clearSession } from "../session/store";

describe("SetupDeviceStep", () => {
  beforeEach(() => {
    // Owner signed in, but this browser is not yet paired.
    saveSession({ deviceToken: null, device: null, ownerToken: "owner-jwt", ownerEmail: "owner@example.com" });
  });
  afterEach(() => {
    cleanup();
    clearSession();
    vi.clearAllMocks();
  });

  // [REQ:REQ-P0-003] Owner bootstrap admits this browser to the trust group.
  it("calls SetupOwnerDevice and persists the returned device token", async () => {
    const user = userEvent.setup();
    setupOwnerDevice.mockResolvedValueOnce({
      deviceToken: "dt-owner",
      device: makeDevice({ id: "owner-dev", name: "Workstation" }),
    });
    renderWithProviders(<SetupDeviceStep onJoinInstead={vi.fn()} onSignOut={vi.fn()} />);

    await user.type(screen.getByTestId(selectors.setupDevice.nameInput), "Workstation");
    await user.click(screen.getByTestId(selectors.setupDevice.submit));

    await waitFor(() => {
      expect(loadSession().deviceToken).toBe("dt-owner");
    });
    expect(setupOwnerDevice).toHaveBeenCalledWith(
      expect.objectContaining({ profile: expect.objectContaining({ deviceName: "Workstation" }) }),
    );
  });

  it("signs out (clears owner token) and notifies the parent", async () => {
    const user = userEvent.setup();
    const onSignOut = vi.fn();
    renderWithProviders(<SetupDeviceStep onJoinInstead={vi.fn()} onSignOut={onSignOut} />);

    await user.click(screen.getByTestId(selectors.setupDevice.signOut));
    await waitFor(() => {
      expect(loadSession().ownerToken).toBeNull();
    });
    expect(onSignOut).toHaveBeenCalledOnce();
  });

  it("offers a join-instead escape hatch", async () => {
    const user = userEvent.setup();
    const onJoinInstead = vi.fn();
    renderWithProviders(<SetupDeviceStep onJoinInstead={onJoinInstead} onSignOut={vi.fn()} />);

    await user.click(screen.getByTestId(selectors.setupDevice.joinInstead));
    expect(onJoinInstead).toHaveBeenCalledOnce();
  });

  it("explains when the hub already belongs to another owner", async () => {
    const user = userEvent.setup();
    setupOwnerDevice.mockRejectedValueOnce(new ConnectError("not hub owner", Code.PermissionDenied));
    renderWithProviders(<SetupDeviceStep onJoinInstead={vi.fn()} onSignOut={vi.fn()} />);

    await user.click(screen.getByTestId(selectors.setupDevice.submit));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.setupDevice.error)).toHaveTextContent(
        strings.setupDevice.alreadyOwned,
      ),
    );
  });

  it("explains when the owner sign-in needs refreshing", async () => {
    const user = userEvent.setup();
    setupOwnerDevice.mockRejectedValueOnce(new ConnectError("missing owner", Code.Unauthenticated));
    renderWithProviders(<SetupDeviceStep onJoinInstead={vi.fn()} onSignOut={vi.fn()} />);

    await user.click(screen.getByTestId(selectors.setupDevice.submit));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.setupDevice.error)).toHaveTextContent(
        strings.setupDevice.signInExpired,
      ),
    );
  });
});
