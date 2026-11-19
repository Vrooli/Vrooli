import type { ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";

import { createTestQueryClient } from "../test-utils";
import { useSessionNotificationMute } from "./useSessionNotificationMute";

const mocks = vi.hoisted(() => ({
  fetchNotificationCatalog: vi.fn(),
  updateNotificationPreference: vi.fn(),
}));

vi.mock("../api/notifications", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/notifications")>();
  return {
    ...actual,
    fetchNotificationCatalog: mocks.fetchNotificationCatalog,
    notificationPreferencesClient: {
      getCatalog: mocks.fetchNotificationCatalog,
      updatePreference: mocks.updateNotificationPreference,
    },
  };
});

function wrapper() {
  const queryClient = createTestQueryClient();
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useSessionNotificationMute", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("reports muted when a session-scoped override disables the session", async () => {
    mocks.fetchNotificationCatalog.mockResolvedValue({
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
    const { result } = renderHook(() => useSessionNotificationMute("sess-1"), {
      wrapper: wrapper(),
    });
    await waitFor(() => expect(result.current.muted).toBe(true));
    expect(result.current.available).toBe(true);
  });

  it("is not muted without an override and toggles to muted", async () => {
    mocks.fetchNotificationCatalog.mockResolvedValue({ sources: [], preferences: [] });
    mocks.updateNotificationPreference.mockResolvedValue({ source_scenario: "web-console", category: "", scope_key: "sess-1", enabled: false, min_urgency: "", channel_override: "" });
    const { result } = renderHook(() => useSessionNotificationMute("sess-1"), {
      wrapper: wrapper(),
    });
    await waitFor(() => expect(result.current.muted).toBe(false));
    result.current.toggle();
    await waitFor(() =>
      expect(mocks.updateNotificationPreference).toHaveBeenCalledWith(expect.objectContaining({ source_scenario: "web-console", category: "", scope_key: "sess-1", enabled: false })),
    );
  });

  it("is unavailable when Notification Hub cannot be reached", async () => {
    const { NotificationsUnavailableError } = await import("../api/notifications");
    mocks.fetchNotificationCatalog.mockRejectedValue(new NotificationsUnavailableError());
    const { result } = renderHook(() => useSessionNotificationMute("sess-1"), {
      wrapper: wrapper(),
    });
    await waitFor(() => expect(result.current.available).toBe(false));
  });
});
