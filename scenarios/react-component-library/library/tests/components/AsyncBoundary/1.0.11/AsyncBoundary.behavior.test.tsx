import { act, cleanup, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

// Copy owned by the published AsyncBoundary release, not this app's catalog.
// Each string is named once here so the assertions survive a wording change
// in one place instead of eleven.
const copy = {
  loading: "Loading content",
  loadingHint: "This may take a moment.",
  offline: "You’re offline",
  offlineHint: "Showing the latest content saved on this device.",
  empty: "No content is available yet.",
  reconnect: "Reconnect to refresh this view.",
} as const;

describe("AsyncBoundary", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("delays the first-load skeleton and announces pending content", () => {
    vi.useFakeTimers();
    renderWithProviders(<AsyncBoundary status="pending" loadingDelay={25} />);

    const boundary = document.querySelector<HTMLElement>(
      "[data-rcl-async-boundary]",
    );
    expect(boundary).not.toBeNull();
    expect(within(boundary!).getByText(copy.loading)).toBeInTheDocument();
    expect(
      document.querySelector("[data-rcl-async-skeleton]"),
    ).not.toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(25);
    });
    expect(document.querySelector("[data-rcl-async-boundary]")).toHaveAttribute(
      "data-rcl-async-status",
      "pending",
    );
    expect(within(boundary!).getByText(copy.loadingHint)).toBeInTheDocument();
  });

  const preservedContent = "saved content";

  it.each([
    ["refreshing", "Refreshing"],
    ["stale", "Showing saved content"],
    ["partial-error", "Some information needs attention"],
  ] as const)("preserves children in %s state", (status, heading) => {
    renderWithProviders(
      <AsyncBoundary status={status}>
        <div>{preservedContent}</div>
      </AsyncBoundary>,
    );
    const boundary = document.querySelector<HTMLElement>(
      "[data-rcl-async-boundary]",
    );
    expect(within(boundary!).getByText(preservedContent)).toBeInTheDocument();
    expect(within(boundary!).getByText(heading)).toBeInTheDocument();
  });

  it("renders an error retry action and waits for the async retry", async () => {
    let resolveRetry!: () => void;
    const retry = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveRetry = resolve;
        }),
    );
    renderWithProviders(
      <AsyncBoundary status="error" error="Unavailable" retry={retry} />,
    );

    const button = screen.getByRole("button", { name: "Try again" });
    act(() => button.click());
    expect(retry).toHaveBeenCalledOnce();
    expect(button).toBeDisabled();
    await act(async () => {
      resolveRetry();
      await Promise.resolve();
    });
    await waitFor(() => expect(button).not.toBeDisabled());
  });

  it("honors explicit offline state and can render an alternate pending surface", () => {
    renderWithProviders(
      <AsyncBoundary
        status="success"
        offline
        pending={<div>custom pending</div>}
      >
        <div>{preservedContent}</div>
      </AsyncBoundary>,
    );
    const boundary = document.querySelector<HTMLElement>(
      "[data-rcl-async-boundary]",
    );
    expect(boundary).toHaveAttribute("data-rcl-async-status", "offline");
    expect(within(boundary!).getByText(copy.offline)).toBeInTheDocument();
    expect(within(boundary!).getAllByText(copy.offlineHint)).toHaveLength(2);
  });

  it("renders the empty and failure surfaces without optional content", () => {
    const errorTitle = "Could not load";
    const errorDetail = "Try later";
    const { rerender } = renderWithProviders(<AsyncBoundary status="idle" />);
    expect(screen.getByText(copy.empty)).toBeInTheDocument();
    expect(
      document.querySelector("[data-rcl-async-boundary]"),
    ).not.toHaveAttribute("aria-busy");

    rerender(
      <AsyncBoundary
        status="error"
        errorTitle={errorTitle}
        error={errorDetail}
      />,
    );
    expect(screen.getByText(errorTitle)).toBeInTheDocument();
    expect(screen.getByText(errorDetail)).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("keeps the supplied content hidden when preservation is disabled", () => {
    const previousContent = "previous content";
    renderWithProviders(
      <AsyncBoundary
        status="refreshing"
        preserveContent={false}
        detectOffline={false}
      >
        <div>{previousContent}</div>
      </AsyncBoundary>,
    );
    expect(screen.queryByText(previousContent)).not.toBeInTheDocument();
    expect(screen.getByText(copy.empty)).toBeInTheDocument();
  });

  it("clamps a negative pending delay and accepts a custom loading surface", () => {
    const customPending = "loading card";
    vi.useFakeTimers();
    renderWithProviders(
      <AsyncBoundary
        status="pending"
        loadingDelay={-50}
        pending={<div>{customPending}</div>}
      />,
    );
    expect(screen.getByText(customPending)).toBeInTheDocument();
    expect(screen.queryByTestId("missing")).not.toBeInTheDocument();
    act(() => {
      vi.runOnlyPendingTimers();
    });
  });

  it("describes success and offline fallback states without duplicate announcements", () => {
    const { rerender } = renderWithProviders(
      <AsyncBoundary status="success" />,
    );
    const boundary = document.querySelector<HTMLElement>(
      "[data-rcl-async-boundary]",
    );
    expect(boundary).toHaveAttribute("data-rcl-async-status", "success");
    expect(screen.getByText(copy.empty)).toBeInTheDocument();
    rerender(<AsyncBoundary status="success" />);

    rerender(<AsyncBoundary status="offline" detectOffline={false} />);
    expect(screen.getByText(copy.reconnect)).toBeInTheDocument();
  });
});
