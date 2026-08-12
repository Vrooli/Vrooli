import { act, cleanup, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AsyncBoundary } from "./ui/DataTable/versions/1.3.0/AsyncBoundary";
import { renderWithProviders } from "../test-utils";

describe("AsyncBoundary", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("delays the first-load skeleton and announces pending content", async () => {
    vi.useFakeTimers();
    renderWithProviders(<AsyncBoundary status="pending" loadingDelay={25} />);

    const boundary = document.querySelector<HTMLElement>("[data-rcl-async-boundary]");
    expect(boundary).not.toBeNull();
    expect(within(boundary!).getByText("Loading content")).toBeInTheDocument();
    expect(document.querySelector("[data-rcl-async-skeleton]")).not.toBeInTheDocument();
    await act(async () => {
      vi.advanceTimersByTime(25);
    });
    expect(document.querySelector("[data-rcl-async-boundary]")).toHaveAttribute(
      "data-rcl-async-status",
      "pending",
    );
    expect(within(boundary!).getByText("This may take a moment.")).toBeInTheDocument();
  });

  it.each([
    ["refreshing", "Refreshing"],
    ["stale", "Showing saved content"],
    ["partial-error", "Some information needs attention"],
  ] as const)("preserves children in %s state", (status, heading) => {
    renderWithProviders(
      <AsyncBoundary status={status}>
        <div>saved content</div>
      </AsyncBoundary>,
    );
    const boundary = document.querySelector<HTMLElement>("[data-rcl-async-boundary]");
    expect(within(boundary!).getByText("saved content")).toBeInTheDocument();
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
    renderWithProviders(<AsyncBoundary status="error" error="Unavailable" retry={retry} />);

    const button = screen.getByRole("button", { name: "Try again" });
    act(() => button.click());
    expect(retry).toHaveBeenCalledOnce();
    expect(button).toBeDisabled();
    await act(async () => resolveRetry());
    await waitFor(() => expect(button).not.toBeDisabled());
  });

  it("honors explicit offline state and can render an alternate pending surface", () => {
    renderWithProviders(
      <AsyncBoundary status="success" offline pending={<div>custom pending</div>}>
        <div>saved content</div>
      </AsyncBoundary>,
    );
    const boundary = document.querySelector<HTMLElement>("[data-rcl-async-boundary]");
    expect(boundary).toHaveAttribute("data-rcl-async-status", "offline");
    expect(within(boundary!).getByText("You’re offline")).toBeInTheDocument();
    expect(
      within(boundary!).getAllByText("Showing the latest content saved on this device."),
    ).toHaveLength(2);
  });

  it("renders the empty and failure surfaces without optional content", () => {
    const { rerender } = renderWithProviders(<AsyncBoundary status="idle" />);
    expect(screen.getByText("No content is available yet.")).toBeInTheDocument();
    expect(document.querySelector("[data-rcl-async-boundary]")).not.toHaveAttribute("aria-busy");

    rerender(<AsyncBoundary status="error" errorTitle="Could not load" error="Try later" />);
    expect(screen.getByText("Could not load")).toBeInTheDocument();
    expect(screen.getByText("Try later")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("keeps the supplied content hidden when preservation is disabled", () => {
    renderWithProviders(
      <AsyncBoundary status="refreshing" preserveContent={false} detectOffline={false}>
        <div>previous content</div>
      </AsyncBoundary>,
    );
    expect(screen.queryByText("previous content")).not.toBeInTheDocument();
    expect(screen.getByText("No content is available yet.")).toBeInTheDocument();
  });

  it("clamps a negative pending delay and accepts a custom loading surface", () => {
    vi.useFakeTimers();
    renderWithProviders(
      <AsyncBoundary status="pending" loadingDelay={-50} pending={<div>loading card</div>} />,
    );
    expect(screen.getByText("loading card")).toBeInTheDocument();
    expect(screen.queryByTestId("missing")).not.toBeInTheDocument();
    act(() => vi.runOnlyPendingTimers());
  });

  it("describes success and offline fallback states without duplicate announcements", () => {
    const { rerender } = renderWithProviders(<AsyncBoundary status="success" />);
    const boundary = document.querySelector<HTMLElement>("[data-rcl-async-boundary]");
    expect(boundary).toHaveAttribute("data-rcl-async-status", "success");
    expect(screen.getByText("No content is available yet.")).toBeInTheDocument();
    rerender(<AsyncBoundary status="success" />);

    rerender(<AsyncBoundary status="offline" detectOffline={false} />);
    expect(screen.getByText("Reconnect to refresh this view.")).toBeInTheDocument();
  });
});
