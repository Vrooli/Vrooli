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
  deleteSubscriptionSession,
  getSubscriptionSession,
  getSubscriptionSummary,
  provisionOpenRouterKey,
  provisionSubscriptionSession,
  removeOpenRouterKey,
  testOpenRouterKey,
} from "../api/monetization";
import { useAuthStore } from "../stores/authStore";
import AccountSection from "../components/settings/AccountSection";

const sessionMock = vi.mocked(getSubscriptionSession);
const summaryMock = vi.mocked(getSubscriptionSummary);
const saveKeyMock = vi.mocked(provisionOpenRouterKey);
const testKeyMock = vi.mocked(testOpenRouterKey);
const removeKeyMock = vi.mocked(removeOpenRouterKey);
const deleteSessionMock = vi.mocked(deleteSubscriptionSession);
const provisionSessionMock = vi.mocked(provisionSubscriptionSession);
const authStoreMock = vi.mocked(useAuthStore);

describe("AccountSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionMock.mockResolvedValue({ configured: false });
    summaryMock.mockResolvedValue({ configured: false, plan_tier: "free", pending_sync: 0 });
    authStoreMock.mockReturnValue({
      error: null,
      initialize: vi.fn().mockResolvedValue(undefined),
      signIn: vi.fn().mockResolvedValue(undefined),
      signOut: vi.fn().mockResolvedValue(undefined),
    } as never);
  });

  it("explains the signed-out local-first experience", async () => {
    render(<AccountSection />);

    await waitFor(() => { expect(screen.getByText("Not signed in")).toBeInTheDocument(); });
    expect(screen.getByTestId("account-free-access-card")).toBeInTheDocument();
    expect(screen.getByLabelText("Optional refresh token")).toBeInTheDocument();
    expect(screen.getByText("Local Ollama generation")).toBeInTheDocument();
    expect(screen.getByText(/Aquila remains fully useful without an account/)).toBeInTheDocument();
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

  it("removes a stored OpenRouter key and reports it", async () => {
    removeKeyMock.mockResolvedValue(undefined as never);
    render(<AccountSection />);
    await waitFor(() => { expect(screen.getByText("Remove key")).toBeInTheDocument(); });
    fireEvent.click(screen.getByText("Remove key"));
    await waitFor(() => { expect(removeKeyMock).toHaveBeenCalledTimes(1); });
    await waitFor(() => { expect(screen.getByText(/OpenRouter key removed/)).toBeInTheDocument(); });
  });

  it("opens the hosted billing and pricing pages in a new tab", async () => {
    const openSpy = vi.fn();
    vi.stubGlobal("open", openSpy);
    try {
      render(<AccountSection />);
      await waitFor(() => { expect(screen.getByText("Manage billing")).toBeInTheDocument(); });
      fireEvent.click(screen.getByText("Manage billing"));
      fireEvent.click(screen.getByText("Compare plans"));
      expect(openSpy).toHaveBeenCalledTimes(2);
      expect(String(openSpy.mock.calls[0]![0])).toContain("/account");
      expect(String(openSpy.mock.calls[1]![0])).toContain("/pricing");
      expect(openSpy.mock.calls[0]![1]).toBe("_blank");
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("provisions a session from a pasted refresh token", async () => {
    provisionSessionMock.mockResolvedValue(undefined as never);
    render(<AccountSection />);
    await waitFor(() => { expect(screen.getByLabelText("Optional refresh token")).toBeInTheDocument(); });
    fireEvent.change(screen.getByLabelText("Optional refresh token"), { target: { value: "  refresh-xyz  " } });
    fireEvent.click(screen.getByText("Sign in"));
    await waitFor(() => { expect(provisionSessionMock).toHaveBeenCalledWith("refresh-xyz"); });
    await waitFor(() => { expect(screen.getByText(/Subscription connected securely/)).toBeInTheDocument(); });
  });

  it("signs out of a connected account and clears the subscription session", async () => {
    const authSignOut = vi.fn().mockResolvedValue(undefined);
    authStoreMock.mockReturnValue({
      error: null,
      initialize: vi.fn().mockResolvedValue(undefined),
      signIn: vi.fn().mockResolvedValue(undefined),
      signOut: authSignOut,
    } as never);
    sessionMock.mockResolvedValue({ configured: true });
    summaryMock.mockResolvedValue({ configured: true, status: "active", plan_tier: "pro", pending_sync: 0 });
    deleteSessionMock.mockResolvedValue(undefined as never);

    render(<AccountSection />);
    await waitFor(() => { expect(screen.getByText("Sign out")).toBeInTheDocument(); });
    fireEvent.click(screen.getByText("Sign out"));
    await waitFor(() => { expect(authSignOut).toHaveBeenCalledTimes(1); });
    expect(deleteSessionMock).toHaveBeenCalledTimes(1);
    await waitFor(() => { expect(screen.getByText(/Signed out/)).toBeInTheDocument(); });
  });
});
