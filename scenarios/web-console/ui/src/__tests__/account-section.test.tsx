import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../test-utils";

vi.mock("../api/monetization", () => ({
  deleteSubscriptionSession: vi.fn(),
  getSubscriptionSession: vi.fn(),
  getSubscriptionSummary: vi.fn(),
  provisionOpenRouterKey: vi.fn(),
  provisionSubscriptionSession: vi.fn(),
  removeOpenRouterKey: vi.fn(),
  testOpenRouterKey: vi.fn(),
}));

vi.mock("../stores/authStore", () => ({
  useAuthStore: vi.fn(() => ({
    error: null,
    initialize: vi.fn().mockResolvedValue(undefined),
    signIn: vi.fn().mockResolvedValue(undefined),
    signOut: vi.fn().mockResolvedValue(undefined),
  })),
}));

import {
  getSubscriptionSession,
  getSubscriptionSummary,
  provisionOpenRouterKey,
  testOpenRouterKey,
} from "../api/monetization";
import AccountSection from "../components/settings/AccountSection";

const sessionMock = vi.mocked(getSubscriptionSession);
const summaryMock = vi.mocked(getSubscriptionSummary);
const saveKeyMock = vi.mocked(provisionOpenRouterKey);
const testKeyMock = vi.mocked(testOpenRouterKey);

describe("AccountSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionMock.mockResolvedValue({ configured: false });
    summaryMock.mockResolvedValue({ configured: false, plan_tier: "free", pending_sync: 0 });
  });

  it("explains the signed-out local-first experience", async () => {
    render(<AccountSection />);

    await waitFor(() => { expect(screen.getByText("Not signed in")).toBeInTheDocument(); });
    expect(screen.getByTestId("account-free-access-card")).toBeInTheDocument();
    expect(screen.getByLabelText("Optional refresh token")).toBeInTheDocument();
    expect(screen.getByText("Local Ollama generation")).toBeInTheDocument();
  });

  it("renders connected plan, credits, and pending sync details", async () => {
    sessionMock.mockResolvedValue({ configured: true });
    summaryMock.mockResolvedValue({
      configured: true,
      status: "active",
      plan_tier: "pro",
      credits: { remaining: 72, limit: 100 },
      pending_sync: 3,
      not_after: "2026-12-31T00:00:00Z",
    });

    render(<AccountSection />);

    await waitFor(() => { expect(screen.getByText("Vrooli account connected")).toBeInTheDocument(); });
    expect(screen.getByText("pro")).toBeInTheDocument();
    expect(screen.getByText("72")).toBeInTheDocument();
    expect(screen.getByTestId("account-credits-meter")).toBeInTheDocument();
    expect(screen.getByText("Usage waiting to sync")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("saves and verifies a write-only OpenRouter key", async () => {
    testKeyMock.mockResolvedValue({ valid: true, source: "openrouter", checked_at: "2026-09-02T12:00:00Z" });
    render(<AccountSection />);

    await waitFor(() => { expect(screen.getByLabelText("OpenRouter API key")).toBeInTheDocument(); });
    fireEvent.change(screen.getByLabelText("OpenRouter API key"), { target: { value: "sk-or-test" } });
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    await waitFor(() => { expect(saveKeyMock).toHaveBeenCalledWith("sk-or-test"); });

    fireEvent.click(screen.getByText("Test key"));
    await waitFor(() => { expect(screen.getByText(/Verified/)).toBeInTheDocument(); });
  });
});
