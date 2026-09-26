import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import * as api from "../lib/api";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { EmptyState } from "../components/EmptyState";
import { EventDetail } from "../components/EventDetail";
import { Layout } from "../components/Layout";
import { AppRouter } from "../router";
import { Activity } from "lucide-react";
import { AnalyticsPage } from "./AnalyticsPage";
import { CircuitBreakerPage } from "./CircuitBreakerPage";
import { CompliancePage } from "./CompliancePage";
import { CorrelationTracePage } from "./CorrelationTracePage";
import { EventLogPage } from "./EventLogPage";
import { PoliciesPage } from "./PoliciesPage";
import { PolicyEditorPage } from "./PolicyEditorPage";
import { ScenarioMetricsPage } from "./ScenarioMetricsPage";
import { SettingsPage } from "./SettingsPage";
import { StreamPage } from "./StreamPage";
import { SubscriptionHealthPage } from "./SubscriptionHealthPage";
import { SubscriptionsPage } from "./SubscriptionsPage";

vi.mock("../lib/api", () => ({
  fetchHealth: vi.fn(),
  fetchEvents: vi.fn(),
  subscribeSSE: vi.fn(),
  fetchPolicy: vi.fn(),
  fetchPolicies: vi.fn(),
  createPolicy: vi.fn(),
  updatePolicy: vi.fn(),
  deletePolicy: vi.fn(),
  fetchViolations: vi.fn(),
  overrideCircuitBreaker: vi.fn(),
  fetchSubscription: vi.fn(),
  fetchSubscriptions: vi.fn(),
  fetchSubscriptionHealth: vi.fn(),
  createSubscription: vi.fn(),
  deleteSubscription: vi.fn(),
}));

const health: api.HealthResponse = {
  status: "healthy",
  service: "vrooli-events",
  timestamp: "2026-01-01T12:00:00Z",
  readiness: true,
  subscribers: 2,
  store: { totalEvents: 42, totalPayloadBytes: 2048 },
};

const events: api.EventEnvelope[] = [
  {
    eventId: "evt-a",
    sourceScenario: "alpha",
    targetScenario: "beta",
    eventType: "alpha.completed.v1",
    correlationId: "corr-1",
    metadata: { trace: "yes" },
    payload: { ok: true },
    createdAt: "2026-01-01T12:00:02Z",
  },
  {
    eventId: "evt-b",
    sourceScenario: "alpha",
    targetScenario: "beta",
    eventType: "alpha.error.v1",
    correlationId: "corr-1",
    createdAt: "2026-01-01T12:00:01Z",
  },
  {
    eventId: "evt-c",
    sourceScenario: "gamma",
    eventType: "gamma.started.v1",
    createdAt: "2026-01-01T12:00:03Z",
  },
];

const policies: api.PolicyRule[] = [
  {
    id: 1,
    rule_type: "access",
    source_scenario: "alpha",
    target_scenario: "beta",
    endpoint_pattern: "/api/*",
    effect: "allow",
    priority: 10,
    enabled: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: 2,
    rule_type: "rate_limit",
    source_scenario: "alpha",
    target_scenario: "beta",
    max_requests: 10,
    window_seconds: 60,
    burst_allowance: 2,
    priority: 5,
    enabled: false,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: 3,
    rule_type: "circuit_breaker",
    source_scenario: "alpha",
    target_scenario: "beta",
    failure_threshold: 3,
    cooldown_seconds: 30,
    success_threshold: 2,
    priority: 1,
    enabled: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
];

const subscription: api.SubscriptionData = {
  id: 7,
  name: "alpha-events",
  owner_scenario: "notification-hub",
  event_pattern: "alpha.**",
  source_filter: "alpha",
  delivery_type: "webhook",
  delivery_target: "http://localhost:1234/events",
  enabled: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

function renderPage(element: React.ReactElement, entry = "/") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
  });
  return renderWithProviders(element, {
    withoutI18n: true,
    withoutRouter: true,
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={[entry]}>
          <Routes>
            <Route path="/policies/:id/edit" element={children} />
            <Route path="/subscriptions/:id/health" element={children} />
            <Route path="*" element={children} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    ),
  });
}

function resetApiMocks() {
  vi.mocked(api.fetchHealth).mockResolvedValue(health);
  vi.mocked(api.fetchEvents).mockResolvedValue(events);
  vi.mocked(api.subscribeSSE).mockReturnValue(() => undefined);
  vi.mocked(api.fetchPolicy).mockResolvedValue(policies[0]!);
  vi.mocked(api.fetchPolicies).mockResolvedValue(policies);
  vi.mocked(api.createPolicy).mockResolvedValue({ id: 9 });
  vi.mocked(api.updatePolicy).mockResolvedValue();
  vi.mocked(api.deletePolicy).mockResolvedValue();
  vi.mocked(api.fetchViolations).mockResolvedValue([
    {
      id: 1,
      timestamp: "2026-01-01T12:00:00Z",
      source_scenario: "alpha",
      target_scenario: "beta",
      endpoint: "/api/v1/items",
      rule_id: 1,
      rule_type: "access",
      reason: "denied by policy",
    },
  ]);
  vi.mocked(api.overrideCircuitBreaker).mockResolvedValue();
  vi.mocked(api.fetchSubscription).mockResolvedValue(subscription);
  vi.mocked(api.fetchSubscriptions).mockResolvedValue([subscription]);
  vi.mocked(api.fetchSubscriptionHealth).mockResolvedValue({
    subscription_id: 7,
    total_delivered: 4,
    total_failed: 1,
    consecutive_failures: 0,
    last_delivered_at: "2026-01-01T12:00:00Z",
    last_failed_at: "2026-01-01T11:00:00Z",
    status: "active",
  });
  vi.mocked(api.createSubscription).mockResolvedValue({ id: 8 });
  vi.mocked(api.deleteSubscription).mockResolvedValue();
}

beforeEach(() => {
  vi.clearAllMocks();
  resetApiMocks();
});

describe("rendered page behavior", () => {
  it("renders analytics and settings store data", async () => {
    renderPage(<AnalyticsPage />);
    expect(await screen.findByText("Analytics Overview")).toBeInTheDocument();
    expect(await screen.findByText("42")).toBeInTheDocument();

    cleanup();
    renderPage(<SettingsPage />);
    expect(await screen.findByText("Retention Configuration")).toBeInTheDocument();
    expect(await screen.findByText("2.0 KB")).toBeInTheDocument();
  });

  it("renders a correlation trace from a deep link and accepts a search", async () => {
    renderPage(<CorrelationTracePage />, "/traces?cid=corr-1");
    expect(await screen.findByTestId("trace-timeline")).toBeInTheDocument();
    expect(screen.getAllByTestId(/trace-node-/)).toHaveLength(3);

    const input = screen.getByTestId("trace-correlation-input");
    fireEvent.change(input, { target: { value: "corr-next" } });
    fireEvent.click(screen.getByTestId("trace-search-button"));
    await waitFor(() => expect(vi.mocked(api.fetchEvents)).toHaveBeenCalled());
  });

  it("renders event history, filters, selects detail, and resets", async () => {
    renderPage(<EventLogPage />, "/events?type=alpha.**&source=alpha&cid=corr-1&limit=10");
    expect(await screen.findByText("Event History")).toBeInTheDocument();
    expect(await screen.findByText("alpha.completed.v1")).toBeInTheDocument();
    fireEvent.click(screen.getByText("alpha.completed.v1"));
    expect(screen.getByText("Event Detail")).toBeInTheDocument();
    expect(screen.getByText((text) => text.includes('"trace"'))).toBeInTheDocument();
    fireEvent.click(screen.getByText("Event Detail").parentElement!.querySelector("button")!);
    fireEvent.change(screen.getByPlaceholderText("Source scenario"), { target: { value: "gamma" } });
    fireEvent.change(screen.getByPlaceholderText("Target scenario"), { target: { value: "beta" } });
    fireEvent.change(screen.getByPlaceholderText("Correlation ID"), { target: { value: "corr-2" } });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "25" } });
    fireEvent.click(screen.getByText("Refresh"));
    expect(screen.getByText("Reset")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Reset"));
  });

  it("renders scenario metrics and exercises sorting and navigation", async () => {
    renderPage(<ScenarioMetricsPage />);
    expect(await screen.findByTestId("scenario-metrics-table")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("sort-scenario"));
    fireEvent.click(screen.getByTestId("sort-scenario"));
    fireEvent.click(screen.getByText("alpha"));
  });

  it("renders policy rules, creates, toggles, edits, and deletes", async () => {
    renderPage(<PoliciesPage />);
    expect(await screen.findByTestId("policies-table")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("policies-new-button"));
    fireEvent.change(screen.getAllByRole("combobox")[0]!, { target: { value: "rate_limit" } });
    fireEvent.change(screen.getByPlaceholderText("Source scenario"), { target: { value: "new-source" } });
    fireEvent.change(screen.getByPlaceholderText("Target scenario"), { target: { value: "new-target" } });
    fireEvent.change(screen.getByPlaceholderText("Endpoint pattern"), { target: { value: "/new" } });
    fireEvent.change(screen.getAllByRole("combobox")[1]!, { target: { value: "deny" } });
    fireEvent.change(screen.getByPlaceholderText("Priority"), { target: { value: "2" } });
    fireEvent.click(screen.getByText("Refresh"));
    fireEvent.click(screen.getByText("Create"));
    await waitFor(() => expect(api.createPolicy).toHaveBeenCalled());
    fireEvent.click(screen.getAllByText("On")[0]!);
    fireEvent.click(screen.getAllByText("Edit")[0]!);
    const firstPolicyRow = screen.getByText("access").closest("tr")!;
    fireEvent.click(within(firstPolicyRow).getAllByRole("button")[2]!);
    await waitFor(() => expect(api.deletePolicy).toHaveBeenCalled());
  });

  it("renders every policy editor conditional section and saves", async () => {
    renderPage(<PolicyEditorPage />, "/policies/1/edit");
    expect(await screen.findByText("Edit Policy #1")).toBeInTheDocument();
    expect(await screen.findByText("Effect")).toBeInTheDocument();
    const selects = screen.getAllByRole("combobox");
    fireEvent.change(selects[1]!, { target: { value: "deny" } });
    for (const input of screen.getAllByRole("spinbutton")) {
      fireEvent.change(input, { target: { value: "11" } });
    }
    for (const input of screen.getAllByRole("textbox")) {
      fireEvent.change(input, { target: { value: "changed" } });
    }
    fireEvent.change(selects[0]!, { target: { value: "rate_limit" } });
    expect(screen.getByText("Max Requests")).toBeInTheDocument();
    for (const input of screen.getAllByRole("spinbutton")) {
      fireEvent.change(input, { target: { value: "12" } });
    }
    fireEvent.change(selects[0]!, { target: { value: "circuit_breaker" } });
    expect(screen.getByText("Failure Threshold")).toBeInTheDocument();
    for (const input of screen.getAllByRole("spinbutton")) {
      fireEvent.change(input, { target: { value: "13" } });
    }
    fireEvent.click(screen.getByText("Back"));
    fireEvent.click(screen.getByText("Cancel"));
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => expect(api.updatePolicy).toHaveBeenCalled());
  });

  it("renders circuit breakers and submits an override", async () => {
    renderPage(<CircuitBreakerPage />);
    expect(await screen.findByTestId("circuit-breakers-table")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Refresh"));
    fireEvent.click(screen.getByText("Override"));
    expect(await screen.findByText("Override Breaker #3")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Cancel"));
    fireEvent.click(screen.getByText("Override"));
    const [stateSelect, ttlInput] = [screen.getByRole("combobox"), screen.getByDisplayValue("60")];
    fireEvent.change(stateSelect, { target: { value: "open" } });
    fireEvent.change(ttlInput, { target: { value: "120" } });
    fireEvent.click(screen.getByText("Apply Override"));
    await waitFor(() => expect(api.overrideCircuitBreaker).toHaveBeenCalledWith(3, "open", 120));
  });

  it("renders compliance violations and subscription health", async () => {
    renderPage(<CompliancePage />);
    expect(await screen.findByTestId("compliance-table")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Refresh"));
    expect(screen.getByText("denied by policy")).toBeInTheDocument();

    renderPage(<SubscriptionHealthPage />, "/subscriptions/7/health");
    expect(await screen.findByText("Health: alpha-events")).toBeInTheDocument();
    expect(screen.getByText("80.0%")).toBeInTheDocument();
  });

  it("renders subscriptions, creates a webhook, navigates to health, and deletes", async () => {
    renderPage(<SubscriptionsPage />);
    expect(await screen.findByTestId("subscriptions-table")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("subscriptions-new-button"));
    fireEvent.change(screen.getByPlaceholderText("Name"), { target: { value: "new-sub" } });
    fireEvent.change(screen.getByPlaceholderText("Owner scenario"), { target: { value: "owner" } });
    fireEvent.change(screen.getByPlaceholderText(/Event pattern/), { target: { value: "owner.*" } });
    fireEvent.change(screen.getByPlaceholderText("Delivery target (URL)"), { target: { value: "http://target" } });
    fireEvent.change(screen.getByDisplayValue("SSE"), { target: { value: "webhook" } });
    fireEvent.click(screen.getByText("Create"));
    await waitFor(() => expect(api.createSubscription).toHaveBeenCalled());
    const subscriptionRow = screen.getByText("alpha-events").closest("tr")!;
    fireEvent.click(within(subscriptionRow).getAllByRole("button")[1]!);
    await waitFor(() => expect(api.deleteSubscription).toHaveBeenCalled());
  });

  it("renders the live stream and handles events, pause, filters, and clear", async () => {
    vi.mocked(api.subscribeSSE).mockImplementation((options) => {
      options.onEvent(events[0]!);
      options.onError?.(new Event("error"));
      return () => undefined;
    });
    renderPage(<StreamPage />, "/stream");
    expect(await screen.findByText("Live Event Stream")).toBeInTheDocument();
    expect(screen.getByText("alpha.completed.v1")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Pause"));
    expect(screen.getByText("Resume")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText(/Filter by type/), { target: { value: "alpha.*" } });
    fireEvent.change(screen.getByPlaceholderText(/Filter by source/), { target: { value: "alpha" } });
    expect(screen.getByText("Reset")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Reset"));
    fireEvent.click(screen.getByText("Clear"));
    expect(screen.getByText("0 events captured")).toBeInTheDocument();

    vi.mocked(api.subscribeSSE).mockImplementation((options) => {
      options.onEvent(events[0]!);
      return () => undefined;
    });
    renderPage(<StreamPage />, "/stream");
    expect(await screen.findByText("alpha.completed.v1")).toBeInTheDocument();
    fireEvent.click(screen.getByText("alpha.completed.v1"));
    fireEvent.click(screen.getByText("Event Detail").parentElement!.querySelector("button")!);
  });

  it("renders API failures and exercises retry paths across the dashboards", async () => {
    const failure = () => new Error("dashboard failure");
    const exerciseRetry = async (element: React.ReactElement, entry = "/", message = "Something went wrong") => {
      const view = renderPage(element, entry);
      expect(await screen.findByText(message)).toBeInTheDocument();
      fireEvent.click(screen.getAllByText("Retry")[0]!);
      view.unmount();
    };

    vi.mocked(api.fetchHealth).mockRejectedValue(new Error("Failed to fetch"));
    await exerciseRetry(<AnalyticsPage />, "/", "Cannot reach the server");

    vi.mocked(api.fetchHealth).mockRejectedValue(failure());
    await exerciseRetry(<SettingsPage />);

    vi.mocked(api.fetchEvents).mockRejectedValue(failure());
    await exerciseRetry(<EventLogPage />);

    vi.mocked(api.fetchEvents).mockRejectedValue(failure());
    await exerciseRetry(<CorrelationTracePage />, "/traces?cid=corr-1");

    vi.mocked(api.fetchEvents).mockRejectedValue(failure());
    await exerciseRetry(<ScenarioMetricsPage />);

    vi.mocked(api.fetchPolicies).mockRejectedValue(failure());
    await exerciseRetry(<PoliciesPage />);

    vi.mocked(api.fetchPolicies).mockRejectedValue(failure());
    await exerciseRetry(<CircuitBreakerPage />);

    vi.mocked(api.fetchViolations).mockRejectedValue(failure());
    await exerciseRetry(<CompliancePage />);

    vi.mocked(api.fetchSubscription).mockRejectedValue(failure());
    vi.mocked(api.fetchSubscriptionHealth).mockRejectedValue(failure());
    await exerciseRetry(<SubscriptionHealthPage />, "/subscriptions/7/health");

    vi.mocked(api.fetchSubscriptions).mockRejectedValue(failure());
    await exerciseRetry(<SubscriptionsPage />);
  });

  it("covers the application shell, empty state, and error-boundary recovery", async () => {
    vi.mocked(api.fetchSubscriptions).mockResolvedValue([]);
    renderPage(<SubscriptionsPage />);
    expect(await screen.findByTestId("empty-state")).toBeInTheDocument();
    screen.getByText("No subscriptions yet");

    const layout = renderPage(<Layout />);
    fireEvent.click(screen.getByText("Analytics"));
    layout.unmount();

    vi.mocked(api.fetchHealth).mockResolvedValue({
      status: undefined as never,
      service: "events",
      timestamp: "",
      readiness: false,
      subscribers: 0,
      store: undefined as never,
    });
    const sparseAnalytics = renderPage(<AnalyticsPage />);
    expect(await screen.findByText("unknown")).toBeInTheDocument();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
    sparseAnalytics.unmount();

    const sparseSettings = renderPage(<SettingsPage />);
    expect(await screen.findByText("0 B")).toBeInTheDocument();
    sparseSettings.unmount();

    vi.mocked(api.fetchPolicies).mockResolvedValue([]);
    const emptyCircuit = renderPage(<CircuitBreakerPage />);
    expect(await screen.findByText("No circuit breaker policies defined yet.")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Policies page"));
    emptyCircuit.unmount();

    vi.mocked(api.fetchSubscription).mockResolvedValue(subscription);
    vi.mocked(api.fetchSubscriptionHealth).mockResolvedValue({
      subscription_id: 7,
      total_delivered: 0,
      total_failed: 0,
      consecutive_failures: 0,
      status: "circuit_broken",
    });
    const sparseHealth = renderPage(<SubscriptionHealthPage />, "/subscriptions/7/health");
    expect(await screen.findByText("Delivery suspended")).toBeInTheDocument();
    expect(screen.getByText("N/A")).toBeInTheDocument();
    expect(screen.getAllByText("Never")).toHaveLength(2);
    sparseHealth.unmount();

    const app = renderWithProviders(<AppRouter />, { withoutI18n: true, withoutRouter: true });
    expect(app.container).toBeInTheDocument();
    app.unmount();

    renderPage(
      <EmptyState
        icon={Activity}
        title="Empty with action"
        description="Nothing here"
        action={{ label: "Go somewhere", to: "/stream" }}
      />,
    );
    expect(screen.getByText("Go somewhere")).toBeInTheDocument();

    const sparseView = renderPage(<EventDetail event={{ eventId: "sparse", sourceScenario: "alpha", eventType: "ping", payload: "raw" }} onClose={() => {}} />);
    expect(screen.getByText("Event Detail")).toBeInTheDocument();
    sparseView.unmount();
  });

  it("recovers a render error with ErrorBoundary", () => {
    let shouldThrow = true;
    const ThrowOnce = () => {
      if (shouldThrow) {
        throw new Error("render failure");
      }
      return <span>recovered</span>;
    };
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    renderPage(
      <ErrorBoundary>
        <ThrowOnce />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    shouldThrow = false;
    fireEvent.click(screen.getByText("Try Again"));
    expect(screen.getByText("recovered")).toBeInTheDocument();
    errorSpy.mockRestore();
  });
});
