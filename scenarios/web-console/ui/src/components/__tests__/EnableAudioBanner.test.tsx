import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import EnableAudioBanner from "../EnableAudioBanner";
import { strings } from "../../consts/strings";

describe("EnableAudioBanner", () => {
  it("renders enable and dismiss buttons with copy explaining the block", () => {
    render(<EnableAudioBanner onEnable={vi.fn().mockResolvedValue(true)} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("enable-audio-banner")).toBeInTheDocument();
    expect(screen.getByTestId("enable-audio-banner-enable")).toBeInTheDocument();
    expect(screen.getByTestId("enable-audio-banner-dismiss")).toBeInTheDocument();
    expect(screen.getByText(strings.enableAudioBanner.title)).toBeInTheDocument();
  });

  it("clicking enable invokes onEnable exactly once", async () => {
    const onEnable = vi.fn().mockResolvedValue(true);
    render(<EnableAudioBanner onEnable={onEnable} onDismiss={vi.fn()} />);
    fireEvent.click(screen.getByTestId("enable-audio-banner-enable"));
    await waitFor(() => expect(onEnable).toHaveBeenCalledTimes(1));
  });

  it("clicking dismiss invokes onDismiss exactly once", () => {
    const onDismiss = vi.fn();
    render(<EnableAudioBanner onEnable={vi.fn().mockResolvedValue(true)} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId("enable-audio-banner-dismiss"));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("shows Enabling… and disables both buttons while enable is in flight", async () => {
    // Defer resolution so the enabling state is observable.
    let resolveEnable: ((v: boolean) => void) | undefined;
    const onEnable = vi.fn(() => new Promise<boolean>((r) => { resolveEnable = r; }));
    const onDismiss = vi.fn();
    render(<EnableAudioBanner onEnable={onEnable} onDismiss={onDismiss} />);

    fireEvent.click(screen.getByTestId("enable-audio-banner-enable"));
    await waitFor(() => expect(screen.getByText(strings.enableAudioBanner.enabling)).toBeInTheDocument());
    expect(screen.getByTestId("enable-audio-banner-enable")).toBeDisabled();
    expect(screen.getByTestId("enable-audio-banner-dismiss")).toBeDisabled();

    // Dismiss is ignored while enabling.
    fireEvent.click(screen.getByTestId("enable-audio-banner-dismiss"));
    expect(onDismiss).not.toHaveBeenCalled();

    resolveEnable?.(true);
    await waitFor(() => expect(screen.getByText(strings.enableAudioBanner.enable)).toBeInTheDocument());
  });

  it("role=status for assistive tech", () => {
    render(<EnableAudioBanner onEnable={vi.fn().mockResolvedValue(true)} onDismiss={vi.fn()} />);
    expect(screen.getByTestId("enable-audio-banner").getAttribute("role")).toBe("status");
  });

  it("applies horizontal safe-area padding", () => {
    render(<EnableAudioBanner onEnable={vi.fn().mockResolvedValue(true)} onDismiss={vi.fn()} />);
    const className = screen.getByTestId("enable-audio-banner").className;
    expect(className).toContain("--wc-safe-left");
    expect(className).toContain("--wc-safe-right");
  });
});
