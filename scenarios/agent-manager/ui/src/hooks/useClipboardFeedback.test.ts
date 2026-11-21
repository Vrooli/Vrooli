import assert from "node:assert/strict";
import { act, fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { renderWithProviders } from "../test-utils/index.js";
import { useClipboardFeedback } from "./useClipboardFeedback.js";

const renderHookHarness = (resetAfterMs?: number) => renderWithProviders(
  createElement(ClipboardHarness, resetAfterMs === undefined ? {} : { resetAfterMs }),
  { withoutQueryClient: true, withoutRouter: true, withoutI18n: true },
);

function ClipboardHarness({ resetAfterMs = 2000 }: { resetAfterMs?: number }) {
  const { status, copyText } = useClipboardFeedback(resetAfterMs);
  return createElement("button", {
    type: "button",
    "data-status": status,
    onClick: () => { void copyText("clipboard payload"); },
  }, status);
}

function setClipboard(writeText: ReturnType<typeof vi.fn>) {
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
}

test("clipboard feedback resets repeated copies from the latest success and clears its timer on unmount", async () => {
  vi.useFakeTimers();
  try {
    setClipboard(vi.fn(async () => undefined));
    const view = renderHookHarness(2000);
    const button = screen.getByRole("button");
    const setTimeoutSpy = vi.spyOn(globalThis, "setTimeout");
    const clearTimeoutSpy = vi.spyOn(globalThis, "clearTimeout");
    const feedbackTimers = () => setTimeoutSpy.mock.calls.flatMap(([, delay], index) =>
      delay === 2000 ? [setTimeoutSpy.mock.results[index]?.value] : []);
    const wasCleared = (timer: unknown) => clearTimeoutSpy.mock.calls.some(([cleared]) => cleared === timer);

    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
    });
    const [firstTimer] = feedbackTimers();
    assert.ok(firstTimer !== undefined);
    assert.equal(button.getAttribute("data-status"), "copied");

    await act(async () => { await vi.advanceTimersByTimeAsync(1500); });
    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
    });
    const [, repeatedTimer] = feedbackTimers();
    assert.ok(repeatedTimer !== undefined);
    assert.equal(wasCleared(firstTimer), true);

    await act(async () => { await vi.advanceTimersByTimeAsync(500); });
    assert.equal(button.getAttribute("data-status"), "copied", "the first copy's timeout must not clear repeated feedback");
    await act(async () => { await vi.advanceTimersByTimeAsync(1500); });
    assert.equal(button.getAttribute("data-status"), "idle", "repeated feedback should reset after its full duration");
    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
    });
    const [, , outstandingTimer] = feedbackTimers();
    assert.ok(outstandingTimer !== undefined);
    view.unmount();
    assert.equal(wasCleared(outstandingTimer), true, "unmount must clear the active feedback timer");
  } finally {
    vi.restoreAllMocks();
    vi.useRealTimers();
  }
});

test("clipboard errors retain error feedback and reset it after the configured duration", async () => {
  vi.useFakeTimers();
  try {
    setClipboard(vi.fn(async () => { throw new Error("clipboard denied"); }));
    const view = renderHookHarness(1500);
    const button = screen.getByRole("button");
    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
    });
    assert.equal(button.getAttribute("data-status"), "error");
    await act(async () => { await vi.advanceTimersByTimeAsync(1500); });
    assert.equal(button.getAttribute("data-status"), "idle");
    view.unmount();
  } finally {
    vi.restoreAllMocks();
    vi.useRealTimers();
  }
});

test("successful feedback uses the caller's requested duration", async () => {
  vi.useFakeTimers();
  try {
    setClipboard(vi.fn(async () => undefined));
    const view = renderHookHarness(1500);
    const button = screen.getByRole("button");
    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
    });
    assert.equal(button.getAttribute("data-status"), "copied");
    await act(async () => { await vi.advanceTimersByTimeAsync(1499); });
    assert.equal(button.getAttribute("data-status"), "copied");
    await act(async () => { await vi.advanceTimersByTimeAsync(1); });
    assert.equal(button.getAttribute("data-status"), "idle");
    view.unmount();
  } finally {
    vi.restoreAllMocks();
    vi.useRealTimers();
  }
});

test("clipboard results that settle after unmount schedule no status work", async () => {
  vi.useFakeTimers();
  try {
    const pending: Array<{ resolve: () => void; reject: (error: Error) => void }> = [];
    const writeText = vi.fn(() => new Promise<void>((resolve, reject) => {
      pending.push({ resolve: () => resolve(), reject });
    }));
    setClipboard(writeText);
    const view = renderHookHarness();
    const setTimeoutSpy = vi.spyOn(globalThis, "setTimeout");
    const copyTimerCount = () => setTimeoutSpy.mock.calls.filter(([, delay]) => delay === 2000).length;

    fireEvent.click(screen.getByRole("button"));
    fireEvent.click(screen.getByRole("button"));
    assert.equal(pending.length, 2);
    view.unmount();

    await act(async () => {
      pending[0]!.resolve();
      pending[1]!.reject(new Error("late clipboard denial"));
      await Promise.resolve();
    });
    assert.equal(copyTimerCount(), 0, "late success or rejection must not create feedback timers");
  } finally {
    vi.restoreAllMocks();
    vi.useRealTimers();
  }
});
