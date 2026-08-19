import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import BannerRegion from "../banners/BannerRegion";
import { useSessionRecoveryBanner } from "../banners/useRecoveryBanners";
import { INSTANT_DAMPING } from "../banners/damping";

const listSessionsWithRecovery = vi.fn<() => Promise<unknown>>();
vi.mock("../../api/sessions", () => ({
  listSessionsWithRecovery: () => listSessionsWithRecovery(),
  listRecoverableSessions: () => Promise.resolve([]),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key} ${JSON.stringify(vars)}` : key,
  }),
}));

function Host() {
  return <BannerRegion banners={[useSessionRecoveryBanner()]} damping={INSTANT_DAMPING} />;
}

function snapshot(over: Partial<{ in_progress: boolean; total: number; recovered: number; awaiting_recovery: number; adopted: number }>) {
  return {
    sessions: [],
    recovery: { in_progress: false, total: 0, recovered: 0, awaiting_recovery: 0, adopted: 0, ...over },
  };
}

describe("session recovery banner", () => {
  beforeEach(() => { listSessionsWithRecovery.mockReset(); });
  afterEach(() => { vi.restoreAllMocks(); });

  it("renders nothing when recovery was never in progress", async () => {
    listSessionsWithRecovery.mockResolvedValue(snapshot({ in_progress: false }));
    const { container } = render(<Host />);
    // Give the initial poll a tick to resolve.
    await waitFor(() => { expect(listSessionsWithRecovery).toHaveBeenCalled(); });
    expect(container.querySelector('[data-testid="session-recovery-banner"]')).toBeNull();
  });

  it("shows recovering progress while in progress", async () => {
    listSessionsWithRecovery.mockResolvedValue(snapshot({ in_progress: true, total: 36, recovered: 4 }));
    render(<Host />);
    const banner = await screen.findByTestId("session-recovery-banner");
    expect(banner.textContent).toContain("sessionRecovery.recovering");
    expect(banner.textContent).toContain('"recovered":4');
    expect(banner.textContent).toContain('"total":36');
    // No "View" reload action while still recovering.
    expect(screen.queryByTestId("session-recovery-banner-view")).toBeNull();
  });

  it("shows a completed state with a reload action after recovery finishes", async () => {
    // First poll: in progress. Second poll: done with recovered sessions.
    listSessionsWithRecovery
      .mockResolvedValueOnce(snapshot({ in_progress: true, total: 2, recovered: 0 }))
      .mockResolvedValue(snapshot({ in_progress: false, total: 2, recovered: 2 }));

    vi.useFakeTimers();
    try {
      render(<Host />);
      // Resolve the first poll (in-progress) and let the 1500ms re-poll fire.
      await vi.waitFor(() => { expect(listSessionsWithRecovery).toHaveBeenCalledTimes(1); });
      await vi.advanceTimersByTimeAsync(1600);
      await vi.waitFor(() => { expect(screen.queryByTestId("session-recovery-banner-view")).not.toBeNull(); });
    } finally {
      vi.useRealTimers();
    }
    expect(screen.getByTestId("session-recovery-banner").textContent).toContain("sessionRecovery.recovered");
  });
});
