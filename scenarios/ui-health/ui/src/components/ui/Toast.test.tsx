import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ToastHost, ToastProvider, useToast } from "./Toast";

const DISMISS = "dismiss-x";

function Pusher({ onReady }: { onReady: (push: ReturnType<typeof useToast>["push"]) => void }) {
  const { push } = useToast();
  onReady(push);
  return null;
}

describe("Toast", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("pushes a toast and auto-dismisses after the duration", () => {
    let push: ReturnType<typeof useToast>["push"] | null = null;
    render(
      <ToastProvider>
        <Pusher onReady={(p) => { push = p; }} />
        <ToastHost dismissLabel={DISMISS} data-testid="toast-host" />
      </ToastProvider>,
    );
    act(() => {
      push!({ tone: "info", title: "title-x", durationMs: 1000 });
    });
    expect(screen.getByTestId("toast-host")).toHaveTextContent("title-x");
    act(() => {
      vi.advanceTimersByTime(1100);
    });
    expect(screen.getByTestId("toast-host")).not.toHaveTextContent("title-x");
  });

  it("dismisses on user click", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    let push: ReturnType<typeof useToast>["push"] | null = null;
    render(
      <ToastProvider>
        <Pusher onReady={(p) => { push = p; }} />
        <ToastHost dismissLabel={DISMISS} data-testid="toast-host" />
      </ToastProvider>,
    );
    act(() => {
      push!({ tone: "error", title: "bad-x", durationMs: 0 });
    });
    await user.click(screen.getByRole("button", { name: DISMISS }));
    expect(screen.getByTestId("toast-host")).not.toHaveTextContent("bad-x");
  });
});
