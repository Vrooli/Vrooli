import { act, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  AppNotificationSettings,
  type AppNotificationClient,
  type AppNotificationState,
} from "@vrooli/react-component-library/AppNotificationSettings/1";

function state(displayName: string): AppNotificationState {
  return {
    scenario: displayName.toLowerCase().replaceAll(" ", "-"),
    displayName,
    vapidPublicKey: "",
    eligible: true,
    ineligibleReason: "",
    publicUrl: "https://example.test",
    subscription: null,
    devices: [],
  };
}

function client(displayName: string): AppNotificationClient {
  return {
    support: () => "supported",
    getState: vi.fn(() => Promise.resolve(state(displayName))),
    prepareSubscription: vi.fn(),
    enable: vi.fn(),
    disable: vi.fn(),
    confirm: vi.fn(),
    fetchCategories: vi.fn(() => Promise.resolve([])),
    saveCategory: vi.fn(),
  };
}

describe("AppNotificationSettings client ownership", () => {
  it("uses a replacement client after the settings host changes it", async () => {
    const first = client("First App");
    const second = client("Second App");
    const view = render(<AppNotificationSettings client={first} showCategories={false} />);

    expect(await screen.findByText(/First App notifications/)).toBeInTheDocument();
    view.rerender(<AppNotificationSettings client={second} showCategories={false} />);

    expect(await screen.findByText(/Second App notifications/)).toBeInTheDocument();
    expect(first.getState).toHaveBeenCalledTimes(1);
    expect(second.getState).toHaveBeenCalledTimes(1);
  });

  it("ignores an older app-state response after the client changes", async () => {
    let resolveFirst!: (value: AppNotificationState) => void;
    const first = client("First App");
    first.getState = vi.fn(() => new Promise((resolve) => {
      resolveFirst = resolve;
    }));
    const second = client("Second App");
    const view = render(<AppNotificationSettings client={first} showCategories={false} />);
    view.rerender(<AppNotificationSettings client={second} showCategories={false} />);

    expect(await screen.findByText(/Second App notifications/)).toBeInTheDocument();
    await act(async () => {
      resolveFirst(state("First App"));
    });
    await waitFor(() => expect(screen.getByText(/Second App notifications/)).toBeInTheDocument());
    expect(screen.queryByText(/First App notifications/)).not.toBeInTheDocument();
  });

  it("clears a stale category-load error when a new client loads categories", async () => {
    const first = client("First App");
    first.fetchCategories = vi.fn(() => Promise.reject(new Error("unavailable")));
    const second = client("Second App");
    const view = render(<AppNotificationSettings client={first} />);
    expect(await screen.findByText("Notification types could not be loaded.")).toBeInTheDocument();

    view.rerender(<AppNotificationSettings client={second} />);
    await waitFor(() => {
      expect(screen.queryByText("Notification types could not be loaded.")).not.toBeInTheDocument();
    });
    expect(second.fetchCategories).toHaveBeenCalledTimes(1);
  });
});
