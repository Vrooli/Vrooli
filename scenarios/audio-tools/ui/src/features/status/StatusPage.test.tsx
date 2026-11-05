import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

vi.mock("../../api/healthStatus", () => ({
  getProviderHealth: vi.fn(),
}));
vi.mock("../../api/providerLifecycle", () => ({
  listLocalProviders: vi.fn(),
  startProvider: vi.fn(),
  stopProvider: vi.fn(),
  restartProvider: vi.fn(),
  pullModel: vi.fn(),
  streamProviderLogs: vi.fn(),
}));

import { StatusPage } from "./StatusPage";
import { getProviderHealth } from "../../api/healthStatus";
import {
  listLocalProviders,
  pullModel,
  restartProvider,
  startProvider,
  stopProvider,
  streamProviderLogs,
} from "../../api/providerLifecycle";
import { strings } from "../../consts/strings";

// Hand-shaped values for the proto enums we need. Importing the enum
// from proto-types at top-level would require the mocked modules above
// to also export it; keeping numeric values here matches the generated
// enum order (UNSPECIFIED=0, AVAILABLE=1, UNAVAILABLE=2, UNKNOWN=3 for
// State; START=1, STOP=2, RESTART=3, PULL_MODEL=4, VIEW_LOGS=5).
const STATE_AVAILABLE = 1;
const STATE_UNAVAILABLE = 2;
const TIER_LOCAL = 1;
const CAP_STT = 1;
const CAP_SUMMARIZE = 3;
const PROCESS_RUNNING = 1;
const PROCESS_STOPPED = 2;

function sampleHealth() {
  return {
    generatedAt: "2026-05-17T00:00:00Z",
    cacheTtlSeconds: 30,
    capabilities: [
      {
        capability: CAP_STT,
        effectiveState: STATE_UNAVAILABLE,
        providers: [
          { capability: CAP_STT, tier: TIER_LOCAL, providerId: "whisper-stt", state: STATE_UNAVAILABLE, lastCheckedAt: "2026-05-17T00:00:00Z", errorMessage: "whisper down" },
        ],
      },
      {
        capability: CAP_SUMMARIZE,
        effectiveState: STATE_AVAILABLE,
        providers: [
          { capability: CAP_SUMMARIZE, tier: TIER_LOCAL, providerId: "ollama", state: STATE_AVAILABLE, lastCheckedAt: "2026-05-17T00:00:00Z" },
        ],
      },
    ],
  };
}

function sampleProviders() {
  return {
    providers: [
      { providerId: "whisper-stt", displayName: "Whisper STT", resourceSlug: "whisper", processState: PROCESS_STOPPED, supportedActions: [1, 2, 3, 5] },
      { providerId: "ollama", displayName: "Ollama", resourceSlug: "ollama", processState: PROCESS_RUNNING, supportedActions: [1, 2, 3, 4, 5] },
    ],
  };
}

beforeEach(() => {
  vi.mocked(getProviderHealth).mockResolvedValue(sampleHealth() as never);
  vi.mocked(listLocalProviders).mockResolvedValue(sampleProviders() as never);
  vi.mocked(startProvider).mockResolvedValue({ providerId: "whisper-stt", dryRun: false, message: "started" } as never);
  vi.mocked(stopProvider).mockResolvedValue({ providerId: "whisper-stt", dryRun: false, message: "stopped" } as never);
  vi.mocked(restartProvider).mockResolvedValue({ providerId: "whisper-stt", dryRun: false, message: "restarted" } as never);
  vi.mocked(pullModel).mockResolvedValue({ providerId: "ollama", modelName: "phi3", dryRun: false, message: "pulled" } as never);
  vi.mocked(streamProviderLogs).mockReturnValue((async function* () {
    await Promise.resolve();
    yield { line: "line-1", tsUnixMs: BigInt(0), stream: 1 };
    yield { line: "line-2", tsUnixMs: BigInt(0), stream: 1 };
  })() as never);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("StatusPage", () => {
  it("renders capability rows with action buttons for local providers", async () => {
    renderWithProviders(<StatusPage />);
    expect(await screen.findByText(/whisper-stt/)).toBeInTheDocument();
    expect(await screen.findByText(/ollama/)).toBeInTheDocument();
    // PULL_MODEL must appear on ollama row only.
    await waitFor(() => {
      expect(screen.getByRole("button", { name: strings.status.actionPullModel })).toBeInTheDocument();
    });
    // Restart appears for at least one provider.
    expect(screen.getAllByRole("button", { name: strings.status.actionRestart }).length).toBeGreaterThan(0);
  });

  it("Restart button invokes restartProvider mutation and refetches", async () => {
    renderWithProviders(<StatusPage />);
    const user = userEvent.setup();
    const restartBtns = await screen.findAllByRole("button", { name: strings.status.actionRestart });
    const restartBtn = restartBtns[0];
    if (!restartBtn) throw new Error("Restart button not rendered");
    await user.click(restartBtn);
    await waitFor(() => {
      expect(vi.mocked(restartProvider)).toHaveBeenCalledTimes(1);
    });
    // Refetch fires after settle (>=2 health calls — initial + invalidate).
    await waitFor(() => {
      expect(vi.mocked(getProviderHealth).mock.calls.length).toBeGreaterThanOrEqual(2);
    });
  });

  it("PullModel modal posts the typed model name", async () => {
    renderWithProviders(<StatusPage />);
    const user = userEvent.setup();
    const pullBtn = await screen.findByRole("button", { name: strings.status.actionPullModel });
    await user.click(pullBtn);
    const input = await screen.findByLabelText(strings.status.pullFieldLabel);
    await user.type(input, "phi3:mini");
    const confirm = screen.getByRole("button", { name: strings.status.pullConfirm });
    await user.click(confirm);
    await waitFor(() => {
      expect(vi.mocked(pullModel)).toHaveBeenCalledWith("phi3:mini");
    });
  });

  it("LogsDrawer renders streamed log lines after opening", async () => {
    renderWithProviders(<StatusPage />);
    const user = userEvent.setup();
    const viewLogs = await screen.findAllByRole("button", { name: strings.status.actionViewLogs });
    const firstLogs = viewLogs[0];
    if (!firstLogs) throw new Error("View logs button not rendered");
    await user.click(firstLogs);
    await waitFor(() => {
      const stream = screen.getByTestId("logs-stream");
      expect(stream.textContent).toMatch(/line-1/);
      expect(stream.textContent).toMatch(/line-2/);
    });
  });
});
