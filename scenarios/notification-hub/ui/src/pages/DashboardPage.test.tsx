import { en } from "../i18n/catalog";
import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";

import { renderWithProviders } from "../test-utils";
import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { DashboardPage } from "./DashboardPage";

vi.mock("../api/notifications", () => ({
  deliveryClient: { getTimeline: vi.fn() },
  conversationsClient: { listAsks: vi.fn() },
}));

import { conversationsClient, deliveryClient } from "../api/notifications";

const getTimeline = vi.mocked(deliveryClient.getTimeline);
const listAsks = vi.mocked(conversationsClient.listAsks);

describe("Updates page", () => {
  afterEach(() => {
    vi.resetAllMocks();
  });

  it("Given a native notification, when Updates opens, then its safe original source is linked without a Hub approval", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [{ id: "native", title: "Permission", contextUrl: "https://agent-manager.example.test/api/v1/authorizations/native/approve", state: NotificationState.DELIVERED }, { id: "unsafe", title: "Unsafe", contextUrl: "javascript:alert(1)", state: NotificationState.PENDING }] } as never);
    listAsks.mockResolvedValue({ asks: [] } as never);
    renderWithProviders(<DashboardPage />);
    // The destination host is part of the link, so the operator sees where it goes before leaving.
    expect(await screen.findByRole("link", { name: "Open source · agent-manager.example.test" })).toHaveAttribute("href", "https://agent-manager.example.test/api/v1/authorizations/native/approve");
    expect(screen.getAllByRole("link", { name: /^Open source/ })).toHaveLength(1);
  });

  it("says plainly that nothing has arrived yet, and keeps Hub health on the Delivery page", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [] } as never);
    listAsks.mockResolvedValue({ asks: [] } as never);

    renderWithProviders(<DashboardPage />);

    const page = await screen.findByTestId(selectors.pages.dashboard);
    expect(page).toHaveTextContent("No updates yet");
    expect(page).toHaveTextContent("When an agent sends you something, it shows up here.");
    expect(screen.queryByTestId(selectors.health.card)).not.toBeInTheDocument();
    expect(listAsks).toHaveBeenCalledWith({ openOnly: false, limit: 200 });
  });

  it("shows informational content and the seven native states without read or acknowledgement controls", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [
        { id: "delivered", title: "Welcome", body: "Hello", requestedBy: "agent-manager", createdAt: "2026-10-20T14:00:00Z", urgency: "normal", sensitivityLabel: "public", reason: "raw adapter failure sentinel: endpoint=https://private.example/token", state: NotificationState.DELIVERED },
        { id: "pending", title: "Pending item", body: "Later", state: NotificationState.PENDING },
        { id: "held", title: "Held item", body: "Waiting", state: NotificationState.HELD },
        { id: "routed", title: "Routed item", body: "Ready", state: NotificationState.ROUTED },
        { id: "failed", title: "Failed item", body: "Error", state: NotificationState.FAILED },
        { id: "unroutable", title: "Unroutable item", body: "No route", state: NotificationState.UNROUTABLE },
        { id: "suppressed", title: "Suppressed item", body: "Muted", state: NotificationState.SUPPRESSED },
        { id: "unspecified", title: "No context", body: "Default", state: NotificationState.UNSPECIFIED },
        { id: "unknown", title: "Unknown item", body: "Future value", state: 99 as NotificationState },
      ],
    } as never);
    listAsks.mockResolvedValue({
      asks: [{ id: "ask-1", notificationId: "delivered", state: "pending" }],
    } as never);

    renderWithProviders(<DashboardPage />);

    const page = await screen.findByTestId(selectors.pages.dashboard);
    for (const value of ["Welcome", "Hello", "agent-manager", "Pending", "Held", "Routed", "Delivered", "Failed", "Unroutable", "Suppressed", "Unknown status", "A delivery adapter accepted the payload"]) {
      expect(page).toHaveTextContent(value);
    }
    expect(page).not.toHaveTextContent("raw adapter failure sentinel");
    expect(page).toHaveTextContent(/Unknown status.*Unknown status/s);
    const decisionLink = screen.getByRole("link", { name: "Open decision" });
    expect(decisionLink).toHaveAttribute("href", "/asks/ask-1");
    expect(decisionLink).toHaveClass("min-h-11", "inline-flex");
    expect(screen.getAllByRole("link", { name: "Delivery details" })).toHaveLength(9);
    expect(screen.queryByRole("link", { name: /Open decision/ })).toHaveAttribute("href", "/asks/ask-1");
    expect(screen.queryByRole("button", { name: /read|acknowledge/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/timeline/i)).not.toBeInTheDocument();
  });

  it("does not invent an Open decision link when no returned ask matches the notification", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [{ id: "notification-1", title: "Informational update", body: "Details", state: NotificationState.PENDING }],
    } as never);
    listAsks.mockResolvedValue({ asks: [{ id: "unrelated", notificationId: "different-notification", state: "pending" }] } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByTestId(selectors.pages.dashboard)).toHaveTextContent("Informational update");
    expect(screen.queryByRole("link", { name: "Open decision" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Delivery details" })).toHaveAttribute("href", "/delivery/notification-1");
  });

  it("finds the exact ask identity throughout the bounded returned lookup", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [{ id: "notification-1", title: "Older linked update", state: NotificationState.PENDING }],
    } as never);
    listAsks.mockResolvedValue({
      asks: [
        ...Array.from({ length: 199 }, (_, index) => ({ id: `unrelated-${index}`, notificationId: `other-${index}`, state: "pending" })),
        { id: "matched-ask", notificationId: "notification-1", state: "pending" },
      ],
    } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByRole("link", { name: "Open decision" })).toHaveAttribute("href", "/asks/matched-ask");
  });

  it("links the E4 history notification to its exact returned ask and preserves routed-test context", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [{
        id: "e4-test-history",
        title: "E4 answered history fixture",
        body: "Disposable closed ask",
        state: NotificationState.DELIVERED,
      }],
    } as never);
    listAsks.mockResolvedValue({
      asks: [{ id: "e4-test-answered-ask", notificationId: "e4-test-history", state: "answered" }],
    } as never);

    renderWithProviders(<DashboardPage />, {
      routerEntries: ["/updates?vrooli_test_mode=1&e4_fixture=empty-open-asks"],
    });

    expect(await screen.findByText(/E4 answered history fixture/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open decision" }))
      .toHaveAttribute("href", "/asks/e4-test-answered-ask?vrooli_test_mode=1");
  });

  it("keeps the feed visible while checking decision links", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [{ id: "notification-1", title: "Informational update", state: NotificationState.PENDING }] } as never);
    listAsks.mockImplementation(() => new Promise(() => {}) as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText(/Informational update/)).toBeInTheDocument();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open decision" })).not.toBeInTheDocument();
  });

  it("keeps updates visible when linked-ask lookup fails and does not offer an unverified decision link", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [{ id: "notification-1", title: "Informational update", body: "Details", state: NotificationState.PENDING }],
    } as never);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 0 } } });
    queryClient.setQueryData(["updates", "linked-asks"], {
      asks: [{ id: "stale-ask", notificationId: "notification-1", state: "pending" }],
    });
    listAsks.mockRejectedValue(new Error("offline"));

    renderWithProviders(<DashboardPage />, { queryClient });

    const page = await screen.findByTestId(selectors.pages.dashboard);
    expect(page).toHaveTextContent("Informational update");
    expect(page).toHaveTextContent("Couldn’t check which updates have a decision, so decision links are hidden.");
    expect(screen.queryByRole("link", { name: "Open decision" })).not.toBeInTheDocument();
    queryClient.clear();
  });

  it("reports an unavailable feed without naming producer health", async () => {
    await setLocale("en");
    getTimeline.mockRejectedValue(new Error("offline"));
    listAsks.mockResolvedValue({ asks: [] } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByTestId(selectors.pages.dashboard)).toHaveTextContent("Couldn’t load updates.");
  });

  it("announces initial feed loading while keeping the Updates region available", async () => {
    await setLocale("en");
    getTimeline.mockImplementation(() => new Promise(() => {}) as never);
    listAsks.mockResolvedValue({ asks: [] } as never);

    renderWithProviders(<DashboardPage />);

    const page = screen.getByTestId(selectors.pages.dashboard);
    expect(page).toHaveAccessibleName("Updates");
    expect(page).toHaveTextContent(en.common.loading);
    expect(page).not.toHaveTextContent("No updates yet");
  });

  it("retains returned feed content with a stale marker when refresh fails", async () => {
    await setLocale("en");
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 0 } } });
    queryClient.setQueryData(["updates"], {
      notifications: [{ id: "cached-notification", title: "Previously loaded update", body: "Known content", state: NotificationState.PENDING }],
    });
    getTimeline.mockRejectedValue(new Error("offline"));
    listAsks.mockResolvedValue({ asks: [] } as never);

    renderWithProviders(<DashboardPage />, { queryClient });

    const page = await screen.findByTestId(selectors.pages.dashboard);
    expect(page).toHaveTextContent("Previously loaded update");
    expect(page).toHaveTextContent("Showing what loaded earlier; the refresh failed.");
    expect(page).not.toHaveTextContent("No updates yet");
    queryClient.clear();
  });
});
