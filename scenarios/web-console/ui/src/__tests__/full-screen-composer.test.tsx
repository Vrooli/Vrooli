import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";
import { useState } from "react";
import FullScreenComposer from "../components/FullScreenComposer";
import { composeComposerPayload } from "../lib/composerPayload";
import { useComposerDraft } from "../hooks/useComposerDraft";
import type { GateResult } from "../components/terminal/inputGate";
import type { InputSettlementCallback } from "../hooks/terminal/useStdinStream";

type SettledCb = (offset: number, ok: boolean) => void;

function makeSettlement() {
  const subs = new Set<SettledCb>();
  const subscribe = (cb: SettledCb) => {
    subs.add(cb);
    return () => subs.delete(cb);
  };
  const fire = (ok: boolean) => {
    for (const cb of subs) cb(1, ok);
  };
  return { subscribe, fire };
}

interface HarnessProps {
  onInput?: (data: string, source: string) => GateResult;
  subscribe?: (cb: SettledCb) => () => void;
  awaitOffset?: (offset: number, cb: InputSettlementCallback) => () => void;
  initialOpen?: boolean;
}

function Harness({ onInput = () => ({ status: "sent", offset: 1 }), subscribe, initialOpen = true }: HarnessProps) {
  const draft = useComposerDraft("sess-composer");
  const [open, setOpen] = useState(initialOpen);
  const settlement = makeSettlement();
  const awaitOffset = (offset: number, cb: InputSettlementCallback) => {
    const source = subscribe ?? settlement.subscribe;
    return source((ackSeq, ok) => { if (ackSeq === offset) cb(ok); });
  };
  return (
    <>
      <button data-testid="ext-open" onClick={() => setOpen(true)} />
      <FullScreenComposer
        open={open}
        onClose={() => setOpen(false)}
        draft={draft}
        onInput={onInput as never}
        subscribeInputSettled={subscribe}
        awaitOffset={awaitOffset}
        onFocusTerminal={vi.fn()}
      />
    </>
  );
}

describe("composeComposerPayload", () => {
  it("returns text unchanged when no paths", () => {
    expect(composeComposerPayload("hello", [])).toBe("hello");
  });
  it("space-joins text and paths in order", () => {
    expect(composeComposerPayload("look", ["/a.png", "/b.png"])).toBe("look /a.png /b.png");
  });
  it("returns just paths when text is empty", () => {
    expect(composeComposerPayload("", ["/a.png", "/b.png"])).toBe("/a.png /b.png");
  });
});

describe("FullScreenComposer", () => {
  beforeEach(() => {
    try {
      window.localStorage.clear();
    } catch {
      /* no-op */
    }
  });

  it("does not render terminal keys/modifiers", () => {
    render(<Harness />);
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
    expect(screen.queryByTestId("toolbar-mod-ctrl")).toBeNull();
    expect(screen.queryByTestId("toolbar-key-esc")).toBeNull();
    expect(screen.queryByTestId(/toolbar-key-/)).toBeNull();
  });

  it("round-trips the draft across minimize/expand (Escape)", () => {
    render(<Harness />);
    const input = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "a long multi-line prompt" } });

    // Escape minimizes without losing the draft.
    act(() => {
      fireEvent.keyDown(window, { key: "Escape" });
    });
    expect(screen.queryByTestId("full-screen-composer")).toBeNull();

    // Re-open shows the same text.
    fireEvent.click(screen.getByTestId("ext-open"));
    const reopened = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    expect(reopened.value).toBe("a long multi-line prompt");
  });

  it("preserves draft when the backdrop is clicked", () => {
    const { container } = render(<Harness />);
    const input = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "keep me" } });
    const backdrop = container.querySelector(".bg-wc-backdrop") as HTMLElement;
    expect(backdrop).toBeTruthy();
    act(() => fireEvent.click(backdrop));
    expect(screen.queryByTestId("full-screen-composer")).toBeNull();
    fireEvent.click(screen.getByTestId("ext-open"));
    expect((screen.getByTestId("composer-input") as HTMLTextAreaElement).value).toBe("keep me");
  });

  it("sends through onInput and clears+minimizes only on ok settlement", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    const settlement = makeSettlement();
    render(<Harness onInput={onInput} subscribe={settlement.subscribe} />);

    const input = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "deploy now" } });
    fireEvent.click(screen.getByTestId("composer-send"));

    expect(onInput).toHaveBeenCalledWith("deploy now", "bulk_text");
    // Still open + spinner until settlement.
    expect(screen.getByTestId("composer-sending")).toBeTruthy();
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();

    act(() => settlement.fire(true));
    // Auto-minimized after ok; draft cleared (reopen shows empty).
    expect(screen.queryByTestId("full-screen-composer")).toBeNull();
    fireEvent.click(screen.getByTestId("ext-open"));
    expect((screen.getByTestId("composer-input") as HTMLTextAreaElement).value).toBe("");
  });

  it("keeps draft open and surfaces error on ok=false", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    const settlement = makeSettlement();
    render(<Harness onInput={onInput} subscribe={settlement.subscribe} />);

    const input = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "risky payload" } });
    fireEvent.click(screen.getByTestId("composer-send"));

    act(() => settlement.fire(false));
    // Still open, draft preserved, error surfaced.
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
    expect((screen.getByTestId("composer-input") as HTMLTextAreaElement).value).toBe("risky payload");
    expect(screen.getByTestId("composer-error")).toBeTruthy();
  });

  it("does nothing when send is pressed with an empty draft", () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    render(<Harness onInput={onInput} />);
    fireEvent.click(screen.getByTestId("composer-send"));
    expect(onInput).not.toHaveBeenCalled();
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
  });
});
