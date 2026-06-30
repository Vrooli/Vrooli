import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import SessionRecoveryBanner from "../SessionRecoveryBanner";

const listSessionsWithRecovery = vi.fn<() => Promise<unknown>>();
vi.mock("../../api/sessions", () => ({
  listSessionsWithRecovery: () => listSessionsWithRecovery(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key} ${JSON.stringify(vars)}` : key,
  }),
}));

function snapshot(over: Partial<{ in_progress: boolean; total: number; recovered: number; awaiting_recovery: number; adopted: number }>) {
  return {
    sessions: [],
    recovery: { in_progress: false, total: 0, recovered: 0, awaiting_recovery: 0, adopted: 0, ...over },
  };
}

describe("SessionRecoveryBanner", () => {
  beforeEach(() => { listSessionsWithRecovery.mockReset(); });
  afterEach(() => { vi.restoreAllMocks(); });

  it("renders nothing when recovery was never in progress", async () => {
    listSessionsWithRecovery.mockResolvedValue(snapshot({ in_progress: false }));
    const { container } = render(<SessionRecoveryBanner />);
    // Give the initial poll a tick to resolve.
    await waitFor(() => expect(listSessionsWithRecovery).toHaveBeenCalled());
    expect(container.querySelector('[data-testid="session-recovery-banner"]')).toBeNull();
  });

  it("shows recovering progress while in progress", async () => {
    listSessionsWithRecovery.mockResolvedValue(snapshot({ in_progress: true, total: 36, recovered: 4 }));
    render(<SessionRecoveryBanner />);
    const banner = await screen.findByTestId("session-recovery-banner");
    expect(banner.textContent).toContain("sessionRecovery.recovering");
    expect(banner.textContent).toContain('"recovered":4');
    expect(banner.textContent).toContain('"total":36');
    // No "View" reload button while still recovering.
    expect(screen.queryByTestId("session-recovery-view")).toBeNull();
  });

  it("shows a completed state with a reload action after recovery finishes", async () => {
    // First poll: in progress. Second poll: done with recovered sessions.
    listSessionsWithRecovery
      .mockResolvedValueOnce(snapshot({ in_progress: true, total: 2, recovered: 0 }))
      .mockResolvedValue(snapshot({ in_progress: false, total: 2, recovered: 2 }));

    vi.useFakeTimers();
    try {
      render(<SessionRecoveryBanner />);
      // Resolve the first poll (in-progress) and let the 1500ms re-poll fire.
      await vi.waitFor(() => expect(listSessionsWithRecovery).toHaveBeenCalledTimes(1));
      await vi.advanceTimersByTimeAsync(1600);
      await vi.waitFor(() => expect(screen.queryByTestId("session-recovery-view")).not.toBeNull());
    } finally {
      vi.useRealTimers();
    }
    expect(screen.getByTestId("session-recovery-banner").textContent).toContain("sessionRecovery.recovered");
  });
});
