import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { appDeliveryClient } from "../../api/notifications";
import { DeviceAppsPanel } from "./DeviceAppsPanel";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/notifications", () => ({ appDeliveryClient: {
  listDeviceApps: vi.fn(),
  removeDeviceApp: vi.fn().mockResolvedValue({}),
} }));

const devices = {
  devices: [
    {
      group: "iphone", label: "iPhone", lastUsed: "2026-10-28T10:00:00Z",
      apps: [
        { subscriptionId: "c1", scenario: "", displayName: "Chimesail", healthy: true, status: "Receives every app not listed here", confirmedAt: "" },
        { subscriptionId: "a1", scenario: "web-console", displayName: "Web Console", healthy: true, status: "Shows Web Console notifications here instead of Chimesail", confirmedAt: "2026-10-28T10:00:00Z" },
      ],
    },
    { group: "mac", label: "Mac · Chrome", lastUsed: "2026-10-20T10:00:00Z", apps: [{ subscriptionId: "c2", scenario: "", displayName: "Chimesail", healthy: true, status: "Receives every app not listed here", confirmedAt: "" }] },
  ],
};

describe("DeviceAppsPanel", () => {
  it("lists each device with its apps and hands a device back to Chimesail", async () => {
    const user = userEvent.setup();
    vi.mocked(appDeliveryClient.listDeviceApps).mockResolvedValue(devices as never);
    renderWithProviders(<DeviceAppsPanel />);

    const root = await screen.findByTestId(selectors.deviceApps.root);
    await vi.waitFor(() => {
      expect(root).toHaveTextContent("iPhone");
    });
    expect(root).toHaveTextContent("Mac · Chrome");
    expect(root).toHaveTextContent("Shows Web Console notifications here instead of Chimesail");
    const buttons = screen.getAllByTestId(selectors.deviceApps.removeButton);
    expect(buttons).toHaveLength(1);

    await user.click(buttons[0]!);
    expect(vi.mocked(appDeliveryClient.removeDeviceApp)).toHaveBeenCalledWith({ subscriptionId: "a1" });
    expect(await screen.findByTestId(selectors.deviceApps.message)).toHaveTextContent("Web Console notifications on iPhone now arrive in Chimesail.");
  });

  it("says so when no device has notifications turned on", async () => {
    vi.mocked(appDeliveryClient.listDeviceApps).mockResolvedValue({ devices: [] } as never);
    renderWithProviders(<DeviceAppsPanel />);
    const root = await screen.findByTestId(selectors.deviceApps.root);
    await vi.waitFor(() => {
      expect(root).toHaveTextContent("No device has notifications turned on yet.");
    });
  });
});
