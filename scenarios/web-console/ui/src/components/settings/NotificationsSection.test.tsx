import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import NotificationsSection from "./NotificationsSection";
import type { NotificationPreference } from "../../api/notifications";
import { renderWithProviders } from "../../test-utils";

const notificationMocks = vi.hoisted(() => ({
  fetchNotificationCatalog: vi.fn(),
  updateNotificationPreference: vi.fn(),
  fetchScenarioLink: vi.fn(),
}));

vi.mock("../../api/notifications", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/notifications")>();
  return {
    ...actual,
    fetchNotificationCatalog: notificationMocks.fetchNotificationCatalog,
    updateNotificationPreference: notificationMocks.updateNotificationPreference,
    fetchScenarioLink: notificationMocks.fetchScenarioLink,
  };
});

const catalog = {
  sources: [
    {
      source_scenario: "web-console",
      display_name: "Web Console",
      icon: "",
      deep_link_base: "",
      declared: true,
      categories: [
        {
          category: "review",
          display_name: "Review requests",
          description: "A PR needs your review.",
          declared: true,
          default_enabled: true,
          default_min_urgency: "normal",
        },
      ],
    },
  ],
  preferences: [],
};

describe("NotificationsSection", () => {
  beforeEach(() => {
    notificationMocks.updateNotificationPreference.mockResolvedValue(undefined);
    notificationMocks.fetchScenarioLink.mockResolvedValue("");
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the catalog query dormant until the section is open", () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue(catalog);
    renderWithProviders(<NotificationsSection open={false} />);
    expect(notificationMocks.fetchNotificationCatalog).not.toHaveBeenCalled();
  });

  it("shows the loading state while the catalog resolves", async () => {
    let resolve!: (value: typeof catalog) => void;
    notificationMocks.fetchNotificationCatalog.mockReturnValue(
      new Promise<typeof catalog>((r) => {
        resolve = r;
      }),
    );

    renderWithProviders(<NotificationsSection open />);

    expect(await screen.findByRole("status")).toHaveTextContent(
      "settings.notificationsSection.loading",
    );
    resolve(catalog);
    await screen.findByTestId("notification-enable-web-console-review");
  });

  it("requests only Web Console's own categories, not every scenario's", async () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue(catalog);

    renderWithProviders(<NotificationsSection open />);

    await screen.findByTestId("notification-enable-web-console-review");
    expect(notificationMocks.fetchNotificationCatalog).toHaveBeenCalledWith("web-console");
  });

  it("renders each source group and its categories", async () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue(catalog);

    renderWithProviders(<NotificationsSection open />);

    expect(await screen.findByText("Web Console")).toBeInTheDocument();
    expect(screen.getByText("Review requests")).toBeInTheDocument();
    expect(
      screen.getByTestId("notification-enable-web-console-review"),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId("notification-urgency-web-console-review"),
    ).toBeInTheDocument();
  });

  it("updates the preference when the enable toggle is flipped", async () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue(catalog);

    renderWithProviders(<NotificationsSection open />);

    const toggle = await screen.findByTestId(
      "notification-enable-web-console-review",
    );
    fireEvent.click(toggle);

    await waitFor(() => {
      expect(notificationMocks.updateNotificationPreference).toHaveBeenCalledTimes(1);
    });
    const sent = notificationMocks.updateNotificationPreference.mock
      .calls[0]![0] as NotificationPreference;
    expect(sent.source_scenario).toBe("web-console");
    expect(sent.category).toBe("review");
    // Default was enabled:true, so the toggle drives it false.
    expect(sent.enabled).toBe(false);
  });

  it("updates the preference when the min-urgency select changes", async () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue(catalog);

    renderWithProviders(<NotificationsSection open />);

    const select = await screen.findByTestId(
      "notification-urgency-web-console-review",
    );
    fireEvent.change(select, { target: { value: "critical" } });

    await waitFor(() => {
      expect(notificationMocks.updateNotificationPreference).toHaveBeenCalledTimes(1);
    });
    const sent = notificationMocks.updateNotificationPreference.mock
      .calls[0]![0] as NotificationPreference;
    expect(sent.min_urgency).toBe("critical");
  });

  it("renders the empty state when no scenario has produced notifications", async () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue({
      sources: [],
      preferences: [],
    });

    renderWithProviders(<NotificationsSection open />);

    expect(await screen.findByTestId("notifications-empty")).toHaveTextContent(
      "settings.notificationsSection.empty",
    );
  });

  it("shows a persistent Open Notification Hub link in the populated state", async () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue(catalog);
    notificationMocks.fetchScenarioLink.mockResolvedValue(
      "https://notification-hub.example.com",
    );

    renderWithProviders(<NotificationsSection open />);

    const link = await screen.findByTestId("notifications-open-hub-link");
    expect(link).toHaveAttribute("href", "https://notification-hub.example.com");
  });

  it("links the empty-state setup hint to the resolved Notification Hub URL", async () => {
    notificationMocks.fetchNotificationCatalog.mockResolvedValue({
      sources: [],
      preferences: [],
    });
    notificationMocks.fetchScenarioLink.mockResolvedValue(
      "https://notification-hub.example.com",
    );

    renderWithProviders(<NotificationsSection open />);

    const link = await screen.findByTestId("notifications-hub-link");
    expect(link).toHaveAttribute("href", "https://notification-hub.example.com");
    expect(notificationMocks.fetchScenarioLink).toHaveBeenCalledWith("notification-hub");
  });

  it("renders the error state with a retry action when the hub is unreachable (503)", async () => {
    const { NotificationsUnavailableError } = await import("../../api/notifications");
    notificationMocks.fetchNotificationCatalog.mockRejectedValue(
      new NotificationsUnavailableError(),
    );

    renderWithProviders(<NotificationsSection open />);

    expect(await screen.findByTestId("notifications-error")).toHaveTextContent(
      "settings.notificationsSection.error",
    );
    expect(screen.getByTestId("notifications-retry")).toBeInTheDocument();
  });
});
