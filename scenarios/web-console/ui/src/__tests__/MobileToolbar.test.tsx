import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";
import { createRef, type RefObject } from "react";
import MobileToolbar, { type MobileToolbarHandle } from "../components/MobileToolbar";
import { i18n } from "../i18n";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { InputSettlementCallback } from "../hooks/terminal/useStdinStream";
import type { PendingInputSnapshot } from "../components/MobileToolbar";
import { toolbarPrefsFromPreset, type ToolbarPrefs } from "../lib/toolbarLayout";

/**
 * Put the toolbar in a known composition. Every shipped preset leaves AI
 * suggest off, so a test that exercises it has to ask for it — which is the
 * point of the setting.
 */
function setToolbarPrefs(overrides: Partial<ToolbarPrefs> = {}) {
  const named = overrides.preset && overrides.preset !== "custom" ? overrides.preset : "balanced";
  const base = toolbarPrefsFromPreset(named);
  useWorkspaceStore.setState({
    toolbarPrefs: {
      ...base,
      ...overrides,
      enabled: { ...base.enabled, ...(overrides.enabled ?? {}) },
    },
  });
}

// Draft persistence wants a stable sessionId — pass a fixed one through props.
function renderToolbar(overrides: Partial<Parameters<typeof MobileToolbar>[0]> = {}) {
  const onInput = overrides.onInput ?? vi.fn(() => ({ status: "sent" as const, offset: 1 }));
  const settledSubs = new Set<(offset: number, ok: boolean) => void>();
  const fireSettled = (offset: number, ok: boolean) => {
    for (const cb of settledSubs) cb(offset, ok);
  };
  const subscribeInputSettled =
    overrides.subscribeInputSettled ??
    vi.fn((cb: (offset: number, ok: boolean) => void) => {
      settledSubs.add(cb);
      return () => settledSubs.delete(cb);
    });
  const awaitOffset = overrides.awaitOffset ?? vi.fn((_seq: number, cb: InputSettlementCallback) => {
    const listener = (offset: number, ok: boolean) => cb(ok);
    settledSubs.add(listener);
    return () => settledSubs.delete(listener);
  });

  const pendingSubs = new Set<() => void>();
  let snapshot: readonly PendingInputSnapshot[] = [];
  const setSnapshot = (next: readonly PendingInputSnapshot[]) => {
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
      awaitOffset={awaitOffset}
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

  it("keeps the mobile command target at the minimum touch height", () => {
    renderToolbar();

    expect(screen.getByTestId("mobile-command-input")).toHaveClass("min-h-11");
  });

  it("preserves draft during sending and clears on ok=true", () => {
    const { onInput, fireSettled } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent" as const, offset: 1 })) });

    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "echo hi" } });
    expect(textarea.value).toBe("echo hi");

    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    expect(onInput).toHaveBeenCalledWith("echo hi", "bulk_text");
    // Draft is kept visible during sending.
    expect(textarea.value).toBe("echo hi");
    expect(screen.getByTestId("send-status-sending")).toBeTruthy();

    act(() => fireSettled(1, true));
    // On success, draft clears and status switches to "sent".
    expect(textarea.value).toBe("");
  });

  it("restores editable draft and shows Send failed on ok=false", () => {
    const { fireSettled } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent" as const, offset: 1 })) });

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

  it("clears a queued status before a later send enters the ack flow", () => {
    const onInput = vi.fn()
      .mockReturnValueOnce({ status: "queued" as const, reason: "ws-closed" as const })
      .mockReturnValueOnce({ status: "sent" as const, offset: 4 });
    const { fireSettled } = renderToolbar({ onInput });
    const textarea = screen.getByTestId("mobile-command-input");

    fireEvent.change(textarea, { target: { value: "first" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    expect(screen.getByTestId("send-status-queued")).toBeInTheDocument();

    fireEvent.change(textarea, { target: { value: "second" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    expect(screen.getByTestId("send-status-sending")).toBeInTheDocument();
    act(() => fireSettled(4, true));
    expect(textarea).toHaveValue("");
  });

  it("treats an empty command as Enter and restores terminal focus", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    const onFocusTerminal = vi.fn();
    renderToolbar({ onInput, onFocusTerminal });

    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(onInput).toHaveBeenCalledWith("\r", "typing");
    expect(onFocusTerminal).toHaveBeenCalledOnce();
  });

  it("submits whitespace verbatim without treating it as Enter", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 2 }));
    renderToolbar({ onInput });
    const textarea = screen.getByTestId("mobile-command-input");

    fireEvent.change(textarea, { target: { value: "  " } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(onInput).toHaveBeenCalledWith("  ", "bulk_text");
  });

  it("applies active modifiers to command text and clears them", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 3 }));
    renderToolbar({ onInput });
    const textarea = screen.getByTestId("mobile-command-input");

    fireEvent.click(screen.getByTestId("toolbar-mod-ctrl"));
    fireEvent.change(textarea, { target: { value: "c" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(onInput).toHaveBeenCalledWith("\x03", "bulk_text");
    expect(useWorkspaceStore.getState().modifiers).toEqual({ ctrl: false, alt: false, shift: false });
  });

  it("keeps the draft when the input gate rejects the command", () => {
    const onInput = vi.fn(() => ({ status: "rejected" as const, reason: "disposed" as const }));
    renderToolbar({ onInput });
    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;

    fireEvent.change(textarea, { target: { value: "retry me" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(textarea.value).toBe("retry me");
    expect(screen.queryByTestId("send-status-sending")).toBeNull();
  });

  it("renders N unsent pill when queue non-empty and hides when drained", async () => {
    // Opt into the real `en` locale so the `{{count}}` interpolation in
    // the unsent-pill heading renders an actual digit — cimode otherwise
    // returns the raw key path with the token unsubstituted.
    await i18n.changeLanguage("en");
    const { setSnapshot } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent" as const, offset: 1 })) });

    expect(screen.queryByTestId("pending-input-pill")).toBeNull();

    act(() => setSnapshot([
      { data: "ls", addedAt: Date.now() - 3000, intent: "typing" },
      { data: "pwd", addedAt: Date.now(), intent: "typing" },
    ]));
    const pill = screen.getByTestId("pending-input-pill");
    expect(pill.textContent).toMatch(/2 unsent/);
    const pillButton = pill.querySelector("button");
    if (!pillButton) throw new Error("pending input pill button missing");
    fireEvent.pointerDown(pillButton);
    fireEvent.click(pillButton);
    expect(screen.getByTestId("pending-input-disclosure")).toBeInTheDocument();

    act(() => setSnapshot([]));
    expect(screen.queryByTestId("pending-input-pill")).toBeNull();
  });

  it("shows truncated and newline-safe entries in the pending disclosure", async () => {
    await i18n.changeLanguage("en");
    const long = "x".repeat(61) + "\nnext";
    const { setSnapshot } = renderToolbar();

    act(() => setSnapshot([
      { data: long, addedAt: Date.now() - 1000, intent: "bulk_text" },
      { data: "line1\nline2", addedAt: Date.now(), intent: "bulk_text" },
    ]));
    fireEvent.click(screen.getByTestId("pending-input-pill").querySelector("button")!);

    const disclosure = screen.getByTestId("pending-input-disclosure");
    expect(disclosure).toHaveTextContent("x".repeat(60) + "…");
    expect(disclosure).toHaveTextContent("line1");
    expect(disclosure).toHaveTextContent("Enter");
    expect(disclosure).toHaveTextContent("line2");
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

  const getArrow = (direction: "up" | "down" | "left" | "right") =>
    screen.getByRole("button", { name: `Arrow ${direction}` });

  it("arrow fires once on pointerdown (no release needed)", () => {
    const { onInput } = renderToolbar();
    const up = getArrow("up");

    act(() => {
      fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
    });

    expect(onInput).toHaveBeenCalledWith(ARROW_UP_INPUT, "typing");
    expect(onInput).toHaveBeenCalledTimes(1);
  });

  it("arrow repeats while held after the initial delay", () => {
    const { onInput } = renderToolbar();
    const up = getArrow("up");

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
      expect(call).toEqual([ARROW_UP_INPUT, "typing"]);
    }
  });

  it("pointerup stops the repeat stream", () => {
    const { onInput } = renderToolbar();
    const left = getArrow("left");
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
    expect(mock.mock.calls[0]).toEqual([ARROW_LEFT_INPUT, "typing"]);
  });

  it("pointerleave (finger dragged off) stops repeats", () => {
    const { onInput } = renderToolbar();
    const up = getArrow("up");
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
    const up = getArrow("up");

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
    expect(onInput).toHaveBeenCalledWith("\x1b", "typing");
  });
});

describe("MobileToolbar — command textbox backspace", () => {
  let originalMaxTouchPoints: number;

  beforeEach(() => {
    try {
      window.localStorage.clear();
    } catch {
      /* no-op */
    }
    // Pretend we're on a touch device — the bug only ever manifested there.
    originalMaxTouchPoints = navigator.maxTouchPoints;
    Object.defineProperty(navigator, "maxTouchPoints", {
      value: 1,
      configurable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(navigator, "maxTouchPoints", {
      value: originalMaxTouchPoints,
      configurable: true,
    });
  });

  // Backspace in a plain textarea is natively supported on every device — tap
  // deletes one char, hold repeats at the OS rate. The toolbar must NOT
  // intercept it (the old custom velocity-repeat belonged to the terminal,
  // whose xterm dependency lacks native key-repeat, and over-deleted here).
  it("leaves backspace to the browser (does not preventDefault the delete event)", () => {
    renderToolbar();
    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;

    fireEvent.change(textarea, { target: { value: "abcdef" } });
    textarea.focus();
    textarea.setSelectionRange(6, 6);

    const event = new InputEvent("beforeinput", {
      inputType: "deleteContentBackward",
      bubbles: true,
      cancelable: true,
    } as InputEventInit);
    act(() => {
      textarea.dispatchEvent(event);
    });

    // Not prevented → the browser performs its own native single-char delete.
    expect(event.defaultPrevented).toBe(false);
  });
});

describe("MobileToolbar — modifiers and optional actions", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("applies a modifier to a toolbar key and clears it after sending", () => {
    setToolbarPrefs({ preset: "dense", density: "compact", arrows: "inline", enabled: { ai: true } });
    useWorkspaceStore.setState({ aiSuggestActive: false });
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    const onFocusTerminal = vi.fn();
    const onExpandComposer = vi.fn();
    renderToolbar({ onInput, onFocusTerminal, onOpenAi: vi.fn(), onUploadImage: vi.fn(), onExpandComposer });

    fireEvent.click(screen.getByTestId("toolbar-mod-ctrl"));
    fireEvent.click(screen.getByTestId("toolbar-key-esc"));
    expect(onInput).toHaveBeenCalledWith("\x1b", "typing");
    expect(onFocusTerminal).toHaveBeenCalled();
    expect(screen.getByTestId("expand-toggle")).toBeInTheDocument();
    expect(screen.getByTestId("expand-toggle")).toHaveClass("min-h-11", "min-w-11");
    expect(screen.getByTestId("mobile-command-submit")).toHaveClass("min-h-11", "min-w-11");
    fireEvent.click(screen.getByTestId("toolbar-ai"));
    fireEvent.click(screen.getByTestId("toolbar-upload-image"));
    fireEvent.click(screen.getByTestId("expand-toggle"));
    expect(onExpandComposer).toHaveBeenCalledOnce();
  });

  it("uses the shared RCL textarea and icon-control contracts", () => {
    renderToolbar({ onOpenAi: vi.fn(), onUploadImage: vi.fn(), onExpandComposer: vi.fn() });

    expect(screen.getByTestId("mobile-command-input")).toHaveAttribute("data-rcl-textarea", "true");
    expect(screen.getByTestId("mobile-command-submit")).toHaveAttribute("data-rcl-control", "true");
    expect(screen.getByRole("button", { name: "Arrow up" })).toHaveAttribute("data-rcl-control", "true");
  });

  it("renders message-mode actions and invokes the view switch after a send", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 2 }));
    const onSwitchToTerminal = vi.fn();
    const { fireSettled } = renderToolbar({ onInput, viewMode: "messages", onSwitchToTerminal, onOpenAi: vi.fn(), onUploadImage: vi.fn() });
    expect(screen.getByTestId("messages-toolbar-actions")).toBeInTheDocument();
    fireEvent.mouseDown(screen.getByTestId("messages-toolbar-actions"));
    fireEvent.pointerDown(screen.getByTestId("toolbar-ai"));
    fireEvent.pointerDown(screen.getByTestId("toolbar-upload-image"));
    fireEvent.change(screen.getByTestId("mobile-command-input"), { target: { value: "hello" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));
    act(() => fireSettled(2, true));
    expect(onSwitchToTerminal).toHaveBeenCalled();
  });

  it("keeps the key area inside the row budget instead of wrapping", () => {
    setToolbarPrefs({ preset: "balanced", maxRows: 2, arrows: "dpad" });
    useWorkspaceStore.setState({ aiSuggestActive: false });
    renderToolbar({ onOpenAi: vi.fn(), onUploadImage: vi.fn() });

    expect(screen.getByTestId("toolbar-row-0")).toBeInTheDocument();
    expect(screen.getByTestId("toolbar-row-1")).toBeInTheDocument();
    // The budget is a ceiling, not a hint: there is no third row to find.
    expect(screen.queryByTestId("toolbar-row-2")).toBeNull();
    expect(screen.getByTestId("toolbar-dpad")).toBeInTheDocument();
  });

  it("collapses to a single row when the budget says one", () => {
    setToolbarPrefs({ maxRows: 1, arrows: "dpad" });
    renderToolbar({ onOpenAi: vi.fn(), onUploadImage: vi.fn() });

    expect(screen.getByTestId("toolbar-row-0")).toBeInTheDocument();
    expect(screen.queryByTestId("toolbar-row-1")).toBeNull();
    // A D-pad needs two rows, so a one-row budget gets the inline run instead.
    expect(screen.queryByTestId("toolbar-dpad")).toBeNull();
  });

  it("always offers More, so a hidden control is never stranded", () => {
    setToolbarPrefs({ enabled: { ai: false, image: false } });
    renderToolbar({ onOpenAi: vi.fn(), onUploadImage: vi.fn() });

    expect(screen.queryByTestId("toolbar-ai")).toBeNull();
    expect(screen.queryByTestId("toolbar-upload-image")).toBeNull();

    fireEvent.click(screen.getByTestId("combo-picker-trigger"));
    expect(screen.getByTestId("more-off-toolbar")).toBeInTheDocument();
    expect(screen.getByTestId("more-control-ai")).toBeInTheDocument();
    expect(screen.getByTestId("more-control-image")).toBeInTheDocument();
  });

  it("sizes the voice control from its slot at every density, not from a class", () => {
    // Regression: RCL size tokens set padding and font, never dimensions, and
    // the voice control clips its overflow instead of letting the glyph spill
    // into the padding the way the other icon controls do. Sizing it by class
    // therefore produced a clipped mic whose box depended on cascade order.
    for (const [density, unit] of [["compact", 32], ["standard", 40], ["large", 44]] as const) {
      setToolbarPrefs({ density, arrows: "inline", maxRows: 2 });
      const { unmount } = renderToolbar({
        onOpenAi: vi.fn(),
        onUploadImage: vi.fn(),
        voice: { supported: true, onStart: vi.fn(), onStop: vi.fn() },
      });

      const slot = screen.getByTestId("toolbar-mic-slot");
      const button = screen.getByTestId("voice-mic-btn");

      expect(button.style.width).toBe(slot.style.width);
      expect(button.style.height).toBe(`${String(unit)}px`);
      expect(button.style.minHeight).toBe(`${String(unit)}px`);
      // The glyph never ramps with density — it matches its neighbours.
      expect(button).toHaveAttribute("data-control-size", "sm");
      // Zero padding is what keeps the glyph inside a clipping box.
      expect(parseFloat(button.style.paddingInline)).toBe(0);

      unmount();
    }
  });

  it("keeps pointer gestures from stealing focus across every preset", () => {
    setToolbarPrefs({ preset: "dense", density: "compact", arrows: "inline" });
    const { rerender } = renderToolbar({ onOpenAi: vi.fn(), onUploadImage: vi.fn() });
    fireEvent.mouseDown(screen.getByTestId("mobile-toolbar"));
    for (const button of screen.getAllByRole("button")) fireEvent.pointerDown(button);

    setToolbarPrefs({ preset: "essential", density: "large", arrows: "dpad" });
    rerender(<MobileToolbar onInput={vi.fn(() => ({ status: "sent" as const, offset: 1 }))} activeSessionId="essential" onFocusTerminal={vi.fn()} onOpenAi={vi.fn()} onUploadImage={vi.fn()} />);
    fireEvent.mouseDown(screen.getByTestId("mobile-toolbar"));
    for (const button of screen.getAllByRole("button")) fireEvent.pointerDown(button);
  });

  it("renders the active AI and voice controls in every terminal view", () => {
    setToolbarPrefs({ preset: "dense", density: "compact", arrows: "inline", enabled: { ai: true } });
    useWorkspaceStore.setState({ aiSuggestActive: true });
    const onOpenAi = vi.fn();
    const onUploadImage = vi.fn();
    const onVoiceStart = vi.fn();
    const onVoiceStop = vi.fn();
    const onVoicePrepare = vi.fn();
    const onExitPassive = vi.fn();
    const onAiSuggestExecute = vi.fn();

    renderToolbar({
      onOpenAi,
      onUploadImage,
      onAiSuggestExecute,
      voice: {
        supported: true,
        preparing: true,
        recording: false,
        persistentMode: true,
        transcribing: false,
        onStart: onVoiceStart,
        onStop: onVoiceStop,
        onPrepare: onVoicePrepare,
        onExitPassive,
      },
    });

    expect(screen.getByTestId("voice-mic-btn")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("toolbar-ai"));
    fireEvent.click(screen.getByTestId("toolbar-upload-image"));
    expect(onOpenAi).toHaveBeenCalledOnce();
    expect(onUploadImage).toHaveBeenCalledOnce();
  });

  it("renders active AI and the complete voice state", () => {
    setToolbarPrefs({ preset: "essential", density: "large", enabled: { ai: true } });
    useWorkspaceStore.setState({
      aiSuggestActive: true,
      modifiers: { ctrl: true, alt: false, shift: false },
    });
    const onOpenAi = vi.fn();
    const onUploadImage = vi.fn();
    const voice = {
      supported: true,
      preparing: true,
      recording: true,
      persistentMode: true,
      listening: true,
      passive: true,
      transcribing: true,
      error: "microphone unavailable",
      level: 0.7,
      backend: "scenario",
      onStart: vi.fn(),
      onStop: vi.fn(),
      onPrepare: vi.fn(),
      onExitPassive: vi.fn(),
    };

    const { rerender } = renderToolbar({ onOpenAi, onUploadImage, voice });

    expect(screen.getByTestId("toolbar-ai")).toHaveAttribute("data-active", "true");
    expect(screen.getByTestId("toolbar-ai").style.getPropertyValue("--color-surface")).toBe("rgb(var(--wc-accent) / 0.2)");
    expect(screen.getByTestId("toolbar-mod-ctrl")).toHaveAttribute("data-active", "true");
    // The glyph stays the size of its neighbours whatever the density does to
    // the box; `toolbar-mic-slot` covers the box itself.
    expect(screen.getByTestId("voice-mic-btn")).toHaveAttribute("data-control-size", "sm");

    setToolbarPrefs({ preset: "dense", density: "compact", arrows: "inline", enabled: { ai: true } });
    rerender(<MobileToolbar onInput={vi.fn(() => ({ status: "sent" as const, offset: 1 }))} activeSessionId="dense" onFocusTerminal={vi.fn()} onOpenAi={onOpenAi} onUploadImage={onUploadImage} voice={voice} />);
    expect(screen.getByTestId("voice-mic-btn")).toHaveAttribute("data-control-size", "sm");

    useWorkspaceStore.setState({ aiSuggestActive: false, modifiers: { ctrl: false, alt: false, shift: false } });
  });

  it("renders the active AI styling in messages mode", () => {
    setToolbarPrefs({ enabled: { ai: true } });
    useWorkspaceStore.setState({ aiSuggestActive: true });
    renderToolbar({ viewMode: "messages", onOpenAi: vi.fn() });

    expect(screen.getByTestId("toolbar-ai")).toHaveAttribute("data-active", "true");
    useWorkspaceStore.setState({ aiSuggestActive: false });
  });

  it("uses optimistic clearing when no settlement callback is supplied", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 9 }));
    render(
      <MobileToolbar
        onInput={onInput}
        awaitOffset={undefined}
        activeSessionId="legacy"
        onFocusTerminal={vi.fn()}
      />,
    );
    const textarea = screen.getByTestId("mobile-command-input");

    fireEvent.change(textarea, { target: { value: "legacy command" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(textarea).toHaveValue("");
  });

  it("renders no toolbar when visibility is disabled", () => {
    renderToolbar({ visible: false });

    expect(screen.queryByTestId("mobile-toolbar")).toBeNull();
  });

  it("uses the interim transcript style while voice text is pending", () => {
    renderToolbar({ voice: { supported: true, onStart: vi.fn(), onStop: vi.fn(), partialTranscript: "draft words" } });

    expect(screen.getByTestId("mobile-command-input")).toHaveClass("text-transparent");
    expect(screen.getByTestId("mobile-interim-overlay")).toHaveTextContent("draft words");
  });
});

describe("MobileToolbar — appendText (voice transcript insertion)", () => {
  beforeEach(() => {
    try {
      window.localStorage.clear();
    } catch {
      /* no-op */
    }
  });

  function renderWithRef() {
    const ref = createRef<MobileToolbarHandle>();
    const utils = render(
      <MobileToolbar
        ref={ref}
        onInput={vi.fn(() => ({ status: "sent" as const, offset: 1 }))}
        activeSessionId="sess-append"
        onFocusTerminal={vi.fn()}
      />,
    );
    const textarea = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    return { ref, textarea, ...utils };
  }

  function getToolbarHandle(ref: RefObject<MobileToolbarHandle>): MobileToolbarHandle {
    if (!ref.current) throw new Error("MobileToolbar ref was not attached");
    return ref.current;
  }

  // RAF used by appendText for caret restoration — flush it synchronously.
  async function flushRaf() {
    await act(async () => {
      await new Promise<void>((r) => requestAnimationFrame(() => r()));
    });
  }

  it("appends to end with a leading space when prior text has no trailing whitespace", async () => {
    const { ref, textarea } = renderWithRef();
    fireEvent.change(textarea, { target: { value: "Hello." } });
    act(() => getToolbarHandle(ref).appendText("World."));
    await flushRaf();
    expect(textarea.value).toBe("Hello. World.");
  });

  it("does not double-space when prior text already ends in whitespace", async () => {
    const { ref, textarea } = renderWithRef();
    fireEvent.change(textarea, { target: { value: "Hello. " } });
    act(() => getToolbarHandle(ref).appendText("World."));
    await flushRaf();
    expect(textarea.value).toBe("Hello. World.");
  });

  it("inserts at the caret position rather than always at the end", async () => {
    const { ref, textarea } = renderWithRef();
    fireEvent.change(textarea, { target: { value: "abc xyz" } });
    // Move caret to between "abc" and " xyz" (index 3).
    textarea.focus();
    textarea.setSelectionRange(3, 3);
    fireEvent.select(textarea);
    act(() => getToolbarHandle(ref).appendText("MID"));
    await flushRaf();
    // Leading: prev char is "c" (non-ws) → add space. Trailing: next char is " " → no space.
    expect(textarea.value).toBe("abc MID xyz");
    // Caret lands at the end of the inserted text ("abc MID".length = 7).
    expect(textarea.selectionStart).toBe(7);
    expect(textarea.selectionEnd).toBe(7);
  });

  it("replaces the selected range when a selection is active", async () => {
    const { ref, textarea } = renderWithRef();
    fireEvent.change(textarea, { target: { value: "keep DROP keep" } });
    textarea.focus();
    textarea.setSelectionRange(5, 9); // selects "DROP"
    fireEvent.select(textarea);
    act(() => getToolbarHandle(ref).appendText("NEW"));
    await flushRaf();
    expect(textarea.value).toBe("keep NEW keep");
  });

  it("uses the last-known caret even after focus has moved away (e.g. to mic button)", async () => {
    const { ref, textarea } = renderWithRef();
    fireEvent.change(textarea, { target: { value: "abc xyz" } });
    textarea.focus();
    textarea.setSelectionRange(3, 3);
    fireEvent.select(textarea);
    // Simulate focus moving to the mic button — blur fires and we record selection.
    fireEvent.blur(textarea);
    act(() => getToolbarHandle(ref).appendText("MID"));
    await flushRaf();
    expect(textarea.value).toBe("abc MID xyz");
  });

  it("does not add a leading space when inserting at position 0", async () => {
    const { ref, textarea } = renderWithRef();
    fireEvent.change(textarea, { target: { value: "world" } });
    textarea.focus();
    textarea.setSelectionRange(0, 0);
    fireEvent.select(textarea);
    act(() => getToolbarHandle(ref).appendText("hello"));
    await flushRaf();
    // Trailing: next char "w" is non-ws → add trailing space.
    expect(textarea.value).toBe("hello world");
  });

  it("exposes focus and clear imperative controls", () => {
    const { ref, textarea } = renderWithRef();
    getToolbarHandle(ref).focusInput();
    expect(document.activeElement).toBe(textarea);
    fireEvent.change(textarea, { target: { value: "clear me" } });
    act(() => getToolbarHandle(ref).clearInput());
    expect(textarea.value).toBe("");
  });
});
