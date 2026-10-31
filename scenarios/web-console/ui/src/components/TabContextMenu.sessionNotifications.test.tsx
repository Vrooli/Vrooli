import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import TabContextMenu from "./TabContextMenu";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import type { NotificationCatalog } from "../api/notifications";

const mocks = vi.hoisted(() => ({
  fetchNotificationCatalog: vi.fn(),
  setSessionNotificationMuted: vi.fn(),
}));

vi.mock("../api/notifications", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/notifications")>();
  return {
    ...actual,
    fetchNotificationCatalog: mocks.fetchNotificationCatalog,
    setSessionNotificationMuted: mocks.setSessionNotificationMuted,
  };
});

const noop = () => {};

// Pre-seed the shared catalog cache so the menu renders the correct label
// synchronously — mirroring the real app, where the catalog is already warm by
// the time the operator right-clicks a tab.
function renderMenu(catalog: NotificationCatalog) {
  const queryClient = createTestQueryClient();
  queryClient.setQueryData(["notification-catalog", "web-console"], catalog);
  mocks.fetchNotificationCatalog.mockResolvedValue(catalog);
  return renderWithProviders(
    <TabContextMenu
      position={{ x: 0, y: 0 }}
      sessionId="sess-1"
      currentGroupId={null}
      isManuallyUnread={false}
      onToggleManuallyUnread={noop}
      onRename={noop}
      onCustomize={noop}
      onRemoveFromGroup={noop}
      onAssignToGroup={noop}
      onClose={noop}
      onDeletePermanently={noop}
      onDismiss={noop}
    />,
    { queryClient },
  );
}

describe("TabContextMenu session notifications", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("offers Mute for a session with no override and writes a muted scope preference", async () => {
    mocks.setSessionNotificationMuted.mockResolvedValue(undefined);
    renderMenu({ sources: [], preferences: [] });

    // A session with no override is not muted, so the control mutes it.
    const item = await screen.findByTestId("tab-ctx-toggle-session-notifications");
    fireEvent.click(item);
    await waitFor(() =>
      expect(mocks.setSessionNotificationMuted).toHaveBeenCalledWith("sess-1", true),
    );
  });

  it("unmutes a session that carries a muted override", async () => {
    mocks.setSessionNotificationMuted.mockResolvedValue(undefined);
    renderMenu({
      sources: [],
      preferences: [
        {
          source_scenario: "web-console",
          category: "",
          enabled: false,
          min_urgency: "",
          channel_override: "",
          scope_key: "sess-1",
        },
      ],
    });

    // The session is muted, so the same control unmutes it — proving the menu
    // reads the per-session override state.
    const item = await screen.findByTestId("tab-ctx-toggle-session-notifications");
    fireEvent.click(item);
    await waitFor(() =>
      expect(mocks.setSessionNotificationMuted).toHaveBeenCalledWith("sess-1", false),
    );
  });
});
