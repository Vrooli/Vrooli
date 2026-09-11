import { afterEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { CopyIconButton, copyIconButtonStyles } from "@vrooli/react-component-library/CopyIconButton/1.0.1";

describe("CopyIconButton 1.0.1", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("copies its value on press, then shows and announces success", async () => {
    const writeText = vi.fn(() => true);
    renderWithProviders(<CopyIconButton value="pnpm install" aria-label="Copy command" writeText={writeText} />);
    const button = screen.getByRole("button", { name: "Copy command" });
    expect(button).toHaveAttribute("data-rcl-copy-state", "idle");
    expect(button).toHaveAttribute("data-rcl-surface", "ghost");

    fireEvent.click(button);
    expect(writeText).toHaveBeenCalledWith("pnpm install");
    await waitFor(() => { expect(button).toHaveAttribute("data-rcl-copy-state", "copied"); });
    expect(button).toHaveAttribute("title", "Copied");
    expect(screen.getByRole("status")).toHaveTextContent("Copied");
    // The name stays the action: success is a state, not a relabel.
    expect(button).toHaveAccessibleName("Copy command");
  });

  it("reads a function value at the moment of the press", async () => {
    const writeText = vi.fn(() => true);
    let current = "first";
    renderWithProviders(<CopyIconButton value={() => current} aria-label="Copy" writeText={writeText} />);
    current = "second";
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));
    await waitFor(() => { expect(writeText).toHaveBeenCalledWith("second"); });
  });

  it("reports a write that returns false, rejects, or throws as a failure", async () => {
    const cases: Array<() => boolean | Promise<boolean>> = [
      () => false,
      () => Promise.reject(new Error("denied")),
      () => { throw new Error("no clipboard"); },
    ];
    for (const writeText of cases) {
      const { unmount } = renderWithProviders(<CopyIconButton value="x" aria-label="Copy" failedLabel="Copy failed" writeText={writeText} />);
      const button = screen.getByRole("button", { name: "Copy" });
      fireEvent.click(button);
      await waitFor(() => { expect(button).toHaveAttribute("data-rcl-copy-state", "failed"); });
      expect(screen.getByRole("status")).toHaveTextContent("Copy failed");
      unmount();
    }
  });

  it("returns to rest after resetAfterMs", async () => {
    vi.useFakeTimers();
    renderWithProviders(<CopyIconButton value="x" aria-label="Copy" resetAfterMs={1000} writeText={() => true} />);
    const button = screen.getByRole("button", { name: "Copy" });
    fireEvent.click(button);
    await act(async () => { await Promise.resolve(); });
    expect(button).toHaveAttribute("data-rcl-copy-state", "copied");
    act(() => { vi.advanceTimersByTime(999); });
    expect(button).toHaveAttribute("data-rcl-copy-state", "copied");
    act(() => { vi.advanceTimersByTime(1); });
    expect(button).toHaveAttribute("data-rcl-copy-state", "idle");
  });

  it("shows copied when told another surface copied the same text", () => {
    renderWithProviders(<CopyIconButton value="x" aria-label="Copy" copied />);
    expect(screen.getByRole("button", { name: "Copy" })).toHaveAttribute("data-rcl-copy-state", "copied");
  });

  it("colours success and failure through its own sheet, outranking IconButton's surface colour", () => {
    renderWithProviders(<CopyIconButton value="x" aria-label="Copy" writeText={() => true} />);
    const css = [...document.head.querySelectorAll("style")].map((node) => node.textContent ?? "").join("\n");
    expect(css).toContain("var(--color-success)");
    expect(copyIconButtonStyles).toMatch(/\[data-rcl-copy-state="copied"\]:hover:not\(:disabled\)\s*\{\s*color:\s*var\(--color-success\)/);
    expect(copyIconButtonStyles).toMatch(/\[data-rcl-copy-state="failed"\]:hover:not\(:disabled\)\s*\{\s*color:\s*var\(--color-danger\)/);
  });
});
