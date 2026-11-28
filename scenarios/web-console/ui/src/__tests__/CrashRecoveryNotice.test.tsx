import { renderWithProviders as render } from "../test-utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import BannerRegion from "../components/banners/BannerRegion";
import { useCrashRecoveryBanner } from "../components/banners/useRecoveryBanners";
import { INSTANT_DAMPING } from "../components/banners/damping";

const listMock = vi.hoisted(() => vi.fn());
vi.mock("../api/sessions", () => ({ listRecoverableSessions: listMock }));

function Host({ onOpenArchive }: { onOpenArchive: () => void }) {
  return <BannerRegion banners={[useCrashRecoveryBanner(onOpenArchive)]} damping={INSTANT_DAMPING} />;
}

describe("crash recovery banner", () => {
  beforeEach(() => {
    listMock.mockReset();
  });

  it("renders nothing when no crash orphans exist", async () => {
    listMock.mockResolvedValue([]);
    const { container } = render(<Host onOpenArchive={vi.fn()} />);
    await waitFor(() => { expect(listMock).toHaveBeenCalled(); });
    expect(container.querySelector("[data-testid='crash-recovery-notice']")).toBeNull();
  });

  it("reports the count compactly and opens the archive", async () => {
    listMock.mockResolvedValue([
      { id: "one", agent_type: "codex", recoverable: true },
      { id: "two", agent_type: "grok", recoverable: false },
    ]);
    const onOpenArchive = vi.fn();
    render(<Host onOpenArchive={onOpenArchive} />);
    const notice = await screen.findByTestId("crash-recovery-notice");
    expect(notice).toHaveTextContent("recoverableSessions.heading");
    // The top safe-area inset is owned by TopSafeArea around the whole region
    // now, not bolted onto this one notice with a `topSafe` prop.
    expect(notice).toHaveAttribute("data-rcl-banner");
    fireEvent.click(screen.getByTestId("crash-recovery-notice-view-archive"));
    expect(onOpenArchive).toHaveBeenCalledOnce();
  });
});
