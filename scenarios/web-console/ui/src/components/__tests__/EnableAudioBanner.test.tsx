import { renderWithProviders as render } from "../../test-utils";
import { describe, it, expect, vi } from "vitest";
import { useState } from "react";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { useTranslation } from "react-i18next";
import BannerRegion from "../banners/BannerRegion";
import { enableAudioBanner } from "../banners/descriptors";
import { INSTANT_DAMPING } from "../banners/damping";
import { strings } from "../../consts/strings";

const ENABLE = "enable-audio-banner-enable";
const DISMISS = "enable-audio-banner-dismiss";

/**
 * Mirrors Workspace: the in-flight guard lives in the host, because a
 * descriptor is data and cannot hold state across the await.
 */
function Host({
  onEnable,
  onDismiss,
}: {
  onEnable: () => Promise<boolean>;
  onDismiss: () => void;
}) {
  const { t } = useTranslation();
  const [enabling, setEnabling] = useState(false);
  return (
    <BannerRegion
      damping={INSTANT_DAMPING}
      banners={[
        enableAudioBanner(t, {
          enabling,
          onEnable: () => {
            if (enabling) return;
            setEnabling(true);
            void onEnable().finally(() => { setEnabling(false); });
          },
          onDismiss,
        }),
      ]}
    />
  );
}

describe("enable audio banner", () => {
  it("renders enable and dismiss buttons with copy explaining the block", () => {
    render(<Host onEnable={vi.fn().mockResolvedValue(true)} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("enable-audio-banner")).toBeInTheDocument();
    expect(screen.getByTestId(ENABLE)).toBeInTheDocument();
    expect(screen.getByTestId(DISMISS)).toBeInTheDocument();
    expect(screen.getByText(strings.enableAudioBanner.title)).toBeInTheDocument();
  });

  it("clicking enable invokes onEnable exactly once", async () => {
    const onEnable = vi.fn().mockResolvedValue(true);
    render(<Host onEnable={onEnable} onDismiss={vi.fn()} />);
    fireEvent.click(screen.getByTestId(ENABLE));
    await waitFor(() => { expect(onEnable).toHaveBeenCalledTimes(1); });
  });

  it("clicking dismiss invokes onDismiss exactly once", () => {
    const onDismiss = vi.fn();
    render(<Host onEnable={vi.fn().mockResolvedValue(true)} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId(DISMISS));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("shows Enabling… and blocks dismiss while enable is in flight", async () => {
    // Defer resolution so the enabling state is observable.
    let resolveEnable: ((v: boolean) => void) | undefined;
    const onEnable = vi.fn(() => new Promise<boolean>((r) => { resolveEnable = r; }));
    const onDismiss = vi.fn();
    render(<Host onEnable={onEnable} onDismiss={onDismiss} />);

    fireEvent.click(screen.getByTestId(ENABLE));
    await waitFor(() => { expect(screen.getByText(strings.enableAudioBanner.enabling)).toBeInTheDocument(); });
    expect(screen.getByTestId(ENABLE)).toBeDisabled();
    // Dismissing mid-unlock would strand the pending TTS event. The button
    // stays put and goes disabled — withdrawing it would resize the banner
    // while the reader is looking at it.
    expect(screen.getByTestId(DISMISS)).toBeDisabled();
    fireEvent.click(screen.getByTestId(DISMISS));
    expect(onDismiss).not.toHaveBeenCalled();

    resolveEnable?.(true);
    await waitFor(() => { expect(screen.getByText(strings.enableAudioBanner.enable)).toBeInTheDocument(); });
  });

  it("role=status for assistive tech", () => {
    render(<Host onEnable={vi.fn().mockResolvedValue(true)} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("enable-audio-banner").getAttribute("role")).toBe("status");
  });

  it("renders through the shared banner base", () => {
    render(<Host onEnable={vi.fn().mockResolvedValue(true)} onDismiss={vi.fn()} />);
    const banner = screen.getByTestId("enable-audio-banner");
    expect(banner).toHaveAttribute("data-wc-banner");
    expect(banner).toHaveAttribute("data-tone", "info");
  });
});
