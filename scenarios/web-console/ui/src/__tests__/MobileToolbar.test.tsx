import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, act, fireEvent, screen } from "@testing-library/react";
import MobileToolbar from "../components/MobileToolbar";

// Draft persistence wants a stable sessionId — pass a fixed one through props.
function renderToolbar(overrides: Partial<Parameters<typeof MobileToolbar>[0]> = {}) {
  const onInput = overrides.onInput ?? vi.fn(() => true);
  const settledSubs = new Set<(seq: number, ok: boolean) => void>();
  const fireSettled = (seq: number, ok: boolean) => {
    for (const cb of settledSubs) cb(seq, ok);
  };
  const subscribeInputSettled =
    overrides.subscribeInputSettled ??
    vi.fn((cb: (seq: number, ok: boolean) => void) => {
      settledSubs.add(cb);
      return () => settledSubs.delete(cb);
    });

  const pendingSubs = new Set<() => void>();
  let snapshot: readonly { data: string; addedAt: number }[] = [];
  const setSnapshot = (next: readonly { data: string; addedAt: number }[]) => {
    snapshot = next;
    for (const cb of pendingSubs) cb();
  };
  const subscribePendingInput =
    overrides.subscribePendingInput ??
    vi.fn((cb: () => void) => {
      pendingSubs.add(cb);
      return () => pendingSubs.delete(cb);
    });
  const getPendingInputSnapshot =
    overrides.getPendingInputSnapshot ?? vi.fn(() => snapshot);

  const utils = render(
    <MobileToolbar
      onInput={onInput}
      subscribeInputSettled={subscribeInputSettled}
      subscribePendingInput={subscribePendingInput}
      getPendingInputSnapshot={getPendingInputSnapshot}
      activeSessionId={overrides.activeSessionId ?? "sess-1"}
      onFocusTerminal={overrides.onFocusTerminal ?? vi.fn()}
      {...overrides}
    />,
  );

  return { ...utils, onInput, fireSettled, setSnapshot };
}

describe("MobileToolbar — send/ack flow", () => {
  beforeEach(() => {
    // Reset draft persistence across tests so inputs start empty.
    try {
      window.localStorage.clear();
    } catch {
      /* no-op */
    }
  });

  it("preserves draft during sending and clears on ok=true", () => {
    const { onInput, fireSettled } = renderToolbar({ onInput: vi.fn(() => true) });

    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "echo hi" } });
    expect(textarea.value).toBe("echo hi");

    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    expect(onInput).toHaveBeenCalledWith("echo hi");
    // Draft is kept visible during sending.
    expect(textarea.value).toBe("echo hi");
    expect(screen.getByTestId("send-status-sending")).toBeTruthy();

    act(() => fireSettled(1, true));
    // On success, draft clears and status switches to "sent".
    expect(textarea.value).toBe("");
  });

  it("restores editable draft and shows Send failed on ok=false", () => {
    const { fireSettled } = renderToolbar({ onInput: vi.fn(() => true) });

    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "long payload" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    expect(textarea.value).toBe("long payload");

    act(() => fireSettled(1, false));
    // Draft is still in the box so the user can retry.
    expect(textarea.value).toBe("long payload");
    expect(screen.getByTestId("send-status-failed")).toBeTruthy();
  });

  it("keeps queued status when onInput returns false", () => {
    renderToolbar({ onInput: vi.fn(() => false) });

    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "queued-cmd" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    // Draft preserved (user still needs to retry).
    expect(textarea.value).toBe("queued-cmd");
    expect(screen.getByTestId("send-status-queued")).toBeTruthy();
  });

  it("renders N unsent pill when queue non-empty and hides when drained", () => {
    const { setSnapshot } = renderToolbar({ onInput: vi.fn(() => true) });

    expect(screen.queryByTestId("pending-input-pill")).toBeNull();

    act(() => setSnapshot([
      { data: "ls", addedAt: Date.now() - 3000 },
      { data: "pwd", addedAt: Date.now() },
    ]));
    const pill = screen.getByTestId("pending-input-pill");
    expect(pill.textContent).toMatch(/2 unsent/);

    act(() => setSnapshot([]));
    expect(screen.queryByTestId("pending-input-pill")).toBeNull();
  });
});
