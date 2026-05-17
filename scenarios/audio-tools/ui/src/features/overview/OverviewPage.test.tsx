import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders, makeHealthResponse } from "../../test-utils";
import { makeApiError } from "../../api/client";
import { strings } from "../../consts/strings";

vi.mock("../../api/health", () => ({
  fetchHealth: vi.fn(),
}));

vi.mock("../../services/settings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../services/settings")>();
  return { ...actual, getProviderConfig: vi.fn() };
});

vi.mock("../../services/usage", () => ({
  listRecent: vi.fn(),
}));

import { OverviewPage } from "./OverviewPage";
import { fetchHealth } from "../../api/health";
import { getProviderConfig } from "../../services/settings";
import { listRecent } from "../../services/usage";

const happyProvider = {
  ok: true as const,
  data: { byokEnabled: true, vrooliEnabled: true, localEnabled: false },
};

beforeEach(() => {
  vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse({ status: "ok" }));
  vi.mocked(getProviderConfig).mockResolvedValue(happyProvider);
  vi.mocked(listRecent).mockResolvedValue({
    ok: true,
    data: [
      {
        operationId: "op-1",
        emittedAt: "2026-05-16T00:00:00Z",
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
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function render() {
  return renderWithProviders(
    <MemoryRouter>
      <OverviewPage />
    </MemoryRouter>,
  );
}

describe("OverviewPage", () => {
  it("renders happy data: header, summary cards, and the recent operation row", async () => {
    render();
    expect(await screen.findByText(strings.app.title)).toBeInTheDocument();
    expect(await screen.findByText(/transcribe/)).toBeInTheDocument();
  });

  it("renders empty state when there are no recent operations", async () => {
    vi.mocked(listRecent).mockResolvedValue({ ok: true, data: [] });
    render();
    expect(await screen.findByText(strings.overview.noOperations)).toBeInTheDocument();
  });

  it("renders error state when listRecent fails", async () => {
    vi.mocked(listRecent).mockResolvedValue({
      ok: false,
      error: makeApiError("internal", "recent-failed", 500),
    });
    render();
    await waitFor(() => expect(screen.getByText(/recent-failed/)).toBeInTheDocument());
  });

  it("fires listRecent exactly once on mount with the documented (3600s, 5) shape", async () => {
    render();
    await waitFor(() => {
      expect(vi.mocked(listRecent)).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(listRecent)).toHaveBeenCalledWith(60 * 60, 5);
  });
});
