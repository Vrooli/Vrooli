import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeApiError } from "../../api/client";

vi.mock("../../services/usage", () => ({
  getSummary: vi.fn(),
  listRecent: vi.fn(),
}));

import { UsagePage } from "./UsagePage";
import { getSummary, listRecent } from "../../services/usage";

const happySummary = {
  ok: true as const,
  data: {
    since: "2026-05-15T00:00:00Z",
    until: "2026-05-16T00:00:00Z",
    operationsTotal: 100,
    creditsTotal: 25,
    errorCount: 2,
    distribution: [
      { providerTier: "local", providerId: "whisper", count: 60 },
      { providerTier: "byok", providerId: "openai", count: 40 },
    ],
    fallbackReasons: [],
  },
};

const happyRecent = {
  ok: true as const,
  data: [
    {
      operationId: "op-1",
      emittedAt: "2026-05-16T01:00:00Z",
      capability: "stt",
      operation: "transcribe",
      providerTier: "local",
      providerId: "whisper",
      modelId: "v3",
      latencyMs: 42,
      creditsCharged: 1,
      error: "",
      fallbackReason: "",
    },
  ],
};

beforeEach(() => {
  vi.mocked(getSummary).mockResolvedValue(happySummary);
  vi.mocked(listRecent).mockResolvedValue(happyRecent);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("UsagePage", () => {
  it("renders happy data: stats, distribution rows, and recent operations", async () => {
    renderWithProviders(<UsagePage />);
    expect(await screen.findByText("usage.title")).toBeInTheDocument();
    expect(await screen.findByText("local · whisper")).toBeInTheDocument();
    expect(screen.getByText("byok · openai")).toBeInTheDocument();
    expect(screen.getByText("transcribe")).toBeInTheDocument();
  });

  it("renders empty state when summary and recent are both empty", async () => {
    vi.mocked(getSummary).mockResolvedValue({
      ok: true,
      data: { ...happySummary.data, distribution: [], operationsTotal: 0 },
    });
    vi.mocked(listRecent).mockResolvedValue({ ok: true, data: [] });
    renderWithProviders(<UsagePage />);
    expect(await screen.findByText("usage.noUsage")).toBeInTheDocument();
    expect(await screen.findByText("usage.recentEmpty")).toBeInTheDocument();
  });

  it("renders error state when getSummary fails", async () => {
    vi.mocked(getSummary).mockResolvedValue({
      ok: false,
      error: makeApiError("internal", "summary-failed", 500),
    });
    renderWithProviders(<UsagePage />);
    await waitFor(() => expect(screen.getByText(/summary-failed/)).toBeInTheDocument());
  });

  it("calls listRecent exactly once on mount with (86400, 50)", async () => {
    renderWithProviders(<UsagePage />);
    await waitFor(() => {
      expect(vi.mocked(listRecent)).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(listRecent)).toHaveBeenCalledWith(86400, 50);
  });
});
