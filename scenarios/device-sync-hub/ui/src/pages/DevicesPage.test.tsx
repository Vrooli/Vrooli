import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders, seedSession } from "../test-utils";

const { listDevices, issuePairingCode } = vi.hoisted(() => ({
  listDevices: vi.fn(),
  issuePairingCode: vi.fn(),
}));

vi.mock("../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/devices")>();
  return { ...actual, devicesClient: { listDevices, issuePairingCode } };
});

import { DevicesPage } from "./DevicesPage";
import { selectors } from "../consts/selectors";

describe("DevicesPage", () => {
  beforeEach(() => {
    listDevices.mockResolvedValue({ devices: [] });
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the owner sign-in prompt when not an owner", () => {
    renderWithProviders(<DevicesPage />);
    expect(screen.getByTestId(selectors.devices.signInPrompt)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.devices.panel)).not.toBeInTheDocument();
  });

  it("renders the management panel (issue + list) once an owner token is present", async () => {
    seedSession({ ownerToken: "owner-jwt" });
    renderWithProviders(<DevicesPage />);

    expect(screen.getByTestId(selectors.devices.panel)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.devices.issuePanel)).toBeInTheDocument();
    await waitFor(() => expect(listDevices).toHaveBeenCalled());
  });
});
