import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, act, fireEvent, screen } from "@testing-library/react";
import MobileToolbar from "../components/MobileToolbar";

// Draft persistence wants a stable sessionId — pass a fixed one through props.
function renderToolbar(overrides: Partial<Parameters<typeof MobileToolbar>[0]> = {}) {
  const onInput = overrides.onInput ?? vi.fn(() => ({ status: "sent" as const, seq: 1 }));
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
    const { onInput, fireSettled } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent" as const, seq: 1 })) });

    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "echo hi" } });
    expect(textarea.value).toBe("echo hi");

    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    expect(onInput).toHaveBeenCalledWith("echo hi", "toolbar-submit");
    // Draft is kept visible during sending.
    expect(textarea.value).toBe("echo hi");
    expect(screen.getByTestId("send-status-sending")).toBeTruthy();

    act(() => fireSettled(1, true));
    // On success, draft clears and status switches to "sent".
    expect(textarea.value).toBe("");
  });

  it("restores editable draft and shows Send failed on ok=false", () => {
    const { fireSettled } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent" as const, seq: 1 })) });

    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "long payload" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    expect(textarea.value).toBe("long payload");

    act(() => fireSettled(1, false));
    // Draft is still in the box so the user can retry.
    expect(textarea.value).toBe("long payload");
    expect(screen.getByTestId("send-status-failed")).toBeTruthy();
  });

  it("keeps queued status when onInput returns queued", () => {
    renderToolbar({ onInput: vi.fn(() => ({ status: "queued" as const, reason: "not-ready" as const })) });

    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "queued-cmd" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    // Draft preserved (user still needs to retry).
    expect(textarea.value).toBe("queued-cmd");
    expect(screen.getByTestId("send-status-queued")).toBeTruthy();
  });

  it("renders N unsent pill when queue non-empty and hides when drained", () => {
    const { setSnapshot } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent" as const, seq: 1 })) });

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

describe("MobileToolbar — arrow hold-to-repeat", () => {
  beforeEach(() => {
    try {
      window.localStorage.clear();
    } catch {
      /* no-op */
    }
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // CSI escapes mirrored from toolbar-keys.ts so assertions stay legible.
  const ARROW_UP_INPUT = "\x1b[A";
  const ARROW_LEFT_INPUT = "\x1b[D";

  // Arrow labels render as Unicode glyphs that slugify() strips, so we
  // query by visible button text instead of by data-testid.
  const getArrow = (glyph: "↑" | "↓" | "←" | "→") =>
    screen.getByRole("button", { name: glyph });

  it("arrow fires once on pointerdown (no release needed)", () => {
    const { onInput } = renderToolbar();
    const up = getArrow("↑");

    act(() => {
      fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
    });

    expect(onInput).toHaveBeenCalledWith(ARROW_UP_INPUT, "toolbar-key");
    expect(onInput).toHaveBeenCalledTimes(1);
  });

  it("arrow repeats while held after the initial delay", () => {
    const { onInput } = renderToolbar();
    const up = getArrow("↑");

    act(() => {
      fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
    });
    expect(onInput).toHaveBeenCalledTimes(1);

    // Advance past the initial delay plus three repeat intervals.
    act(() => {
      vi.advanceTimersByTime(400 + 40 * 3);
    });

    expect(onInput).toHaveBeenCalledTimes(4);
    const mock = (onInput as unknown as import("vitest").Mock);
    for (const call of mock.mock.calls) {
      expect(call).toEqual([ARROW_UP_INPUT, "toolbar-key"]);
    }
  });

  it("pointerup stops the repeat stream", () => {
    const { onInput } = renderToolbar();
    const left = getArrow("←");
    const mock = (onInput as unknown as import("vitest").Mock);

    act(() => {
      fireEvent.pointerDown(left, { pointerType: "touch", button: 0 });
    });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    const countAtRelease = mock.mock.calls.length;

    act(() => {
      fireEvent.pointerUp(left);
    });
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(onInput).toHaveBeenCalledTimes(countAtRelease);
    expect(mock.mock.calls[0]).toEqual([ARROW_LEFT_INPUT, "toolbar-key"]);
  });

  it("pointerleave (finger dragged off) stops repeats", () => {
    const { onInput } = renderToolbar();
    const up = getArrow("↑");
    const mock = (onInput as unknown as import("vitest").Mock);

    act(() => {
      fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
    });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    const countAtLeave = mock.mock.calls.length;

    act(() => {
      fireEvent.pointerLeave(up);
    });
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(onInput).toHaveBeenCalledTimes(countAtLeave);
  });

  it("quick tap on arrow fires exactly once (no phantom repeat after release)", () => {
    const { onInput } = renderToolbar();
    const up = getArrow("↑");

    act(() => {
      fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
    });
    act(() => {
      vi.advanceTimersByTime(50);
    });
    act(() => {
      fireEvent.pointerUp(up);
    });
    // A synthetic click may still follow on some browsers — verify it's ignored.
    act(() => {
      fireEvent.click(up);
    });
    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onInput).toHaveBeenCalledTimes(1);
  });

  it("non-arrow toolbar keys keep click semantics (no pointerdown fire)", () => {
    const { onInput } = renderToolbar();
    const esc = screen.getByTestId("toolbar-key-esc");

    act(() => {
      fireEvent.pointerDown(esc);
    });
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(onInput).not.toHaveBeenCalled();

    act(() => {
      fireEvent.click(esc);
    });
    expect(onInput).toHaveBeenCalledTimes(1);
    expect(onInput).toHaveBeenCalledWith("\x1b", "toolbar-key");
  });
});
