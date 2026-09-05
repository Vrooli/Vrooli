import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, fireEvent, screen, renderHook } from "@testing-library/react";
import { useState } from "react";
import FullScreenComposer from "../components/FullScreenComposer";
import { useComposerDraft } from "../hooks/useComposerDraft";
import { useComposerAttachments } from "../hooks/useComposerAttachments";
import type { GateResult } from "../components/terminal/inputGate";

type SettledCb = (offset: number, ok: boolean) => void;

function makeSettlement() {
  const subs = new Set<SettledCb>();
  return {
    subscribe: (cb: SettledCb) => {
      subs.add(cb);
      return () => subs.delete(cb);
    },
    fire: (ok: boolean) => {
      for (const cb of subs) cb(1, ok);
    },
  };
}

function pngFile(name: string): File {
  return new File(["fake"], name, { type: "image/png" });
}

interface HarnessProps {
  onInput?: (data: string, source: string) => GateResult;
  subscribe?: (cb: SettledCb) => () => void;
  resolve?: () => Promise<string[]>;
}

function AttachHarness({ onInput = () => ({ status: "sent", offset: 1 }), subscribe, resolve }: HarnessProps) {
  const draft = useComposerDraft("sess-att");
  const att = useComposerAttachments();
  const [open, setOpen] = useState(true);
  return (
    <>
      <button data-testid="ext-open" onClick={() => setOpen(true)} />
      <FullScreenComposer
        open={open}
        onClose={() => setOpen(false)}
        draft={draft}
        onInput={onInput as never}
        subscribeInputSettled={subscribe}
        attachments={att.attachments}
        onAttachFiles={att.addFiles}
        onRemoveAttachment={att.removeFile}
        resolveAttachmentPaths={resolve ?? (async () => att.attachments.map((_, i) => `/up/${i}.png`))}
        onClearAttachments={att.clearAll}
      />
    </>
  );
}

describe("useComposerAttachments", () => {
  beforeEach(() => {
    let n = 0;
    (URL as unknown as { createObjectURL: unknown }).createObjectURL = vi.fn(() => `blob:mock/${++n}`);
    (URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = vi.fn();
  });

  it("stages image files and ignores non-images", () => {
    const { result } = renderHook(() => useComposerAttachments());
    act(() => result.current.addFiles([pngFile("a.png"), new File(["x"], "s.sh", { type: "text/plain" })]));
    expect(result.current.attachments).toHaveLength(1);
    expect(result.current.attachments[0]?.file.name).toBe("a.png");
  });

  it("removes a single attachment and clears all", () => {
    const { result } = renderHook(() => useComposerAttachments());
    act(() => result.current.addFiles([pngFile("a.png"), pngFile("b.png")]));
    const firstId = result.current.attachments[0]?.id ?? "";
    act(() => result.current.removeFile(firstId));
    expect(result.current.attachments).toHaveLength(1);
    act(() => result.current.clearAll());
    expect(result.current.attachments).toHaveLength(0);
  });
});

describe("FullScreenComposer — staged attachments", () => {
  beforeEach(() => {
    try {
      window.localStorage.clear();
    } catch {
      /* no-op */
    }
    let n = 0;
    (URL as unknown as { createObjectURL: unknown }).createObjectURL = vi.fn(() => `blob:mock/${++n}`);
    (URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = vi.fn();
  });

  it("stages picked files into the review tray", () => {
    render(<AttachHarness />);
    expect(screen.queryByTestId("composer-attachment-tray")).toBeNull();
    const input = screen.getByTestId("composer-file-input") as HTMLInputElement;
    act(() => fireEvent.change(input, { target: { files: [pngFile("a.png")] } }));
    expect(screen.getByTestId("composer-attachment-tray")).toBeTruthy();
  });

  it("composes ONE payload = text + resolved paths in order, then clears on ok", async () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    const settlement = makeSettlement();
    const resolve = vi.fn(async () => ["/up/a.png", "/up/b.png"]);
    render(<AttachHarness onInput={onInput} subscribe={settlement.subscribe} resolve={resolve} />);

    fireEvent.change(screen.getByTestId("composer-input"), { target: { value: "look at these" } });
    const input = screen.getByTestId("composer-file-input") as HTMLInputElement;
    act(() => fireEvent.change(input, { target: { files: [pngFile("a.png"), pngFile("b.png")] } }));

    await act(async () => {
      fireEvent.click(screen.getByTestId("composer-send"));
    });

    expect(resolve).toHaveBeenCalledTimes(1);
    expect(onInput).toHaveBeenCalledWith("look at these /up/a.png /up/b.png", "bulk_text");

    act(() => settlement.fire(true));
    // Minimized + attachments cleared: reopen shows an empty tray.
    expect(screen.queryByTestId("full-screen-composer")).toBeNull();
    fireEvent.click(screen.getByTestId("ext-open"));
    expect(screen.queryByTestId("composer-attachment-tray")).toBeNull();
  });

  it("keeps attachments + text and shows error when upload fails", async () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    const resolve = vi.fn(async () => {
      throw new Error("upload boom");
    });
    render(<AttachHarness onInput={onInput} resolve={resolve} />);

    fireEvent.change(screen.getByTestId("composer-input"), { target: { value: "with pic" } });
    const input = screen.getByTestId("composer-file-input") as HTMLInputElement;
    act(() => fireEvent.change(input, { target: { files: [pngFile("a.png")] } }));

    await act(async () => {
      fireEvent.click(screen.getByTestId("composer-send"));
    });

    // Nothing sent, error surfaced, attachments + text preserved.
    expect(onInput).not.toHaveBeenCalled();
    expect(screen.getByTestId("composer-error")).toBeTruthy();
    expect(screen.getByTestId("composer-attachment-tray")).toBeTruthy();
    expect((screen.getByTestId("composer-input") as HTMLTextAreaElement).value).toBe("with pic");
  });

  it("prompts to discard when minimizing with staged images", () => {
    render(<AttachHarness />);
    const input = screen.getByTestId("composer-file-input") as HTMLInputElement;
    act(() => fireEvent.change(input, { target: { files: [pngFile("a.png")] } }));

    // Escape does NOT immediately close — it prompts.
    act(() => fireEvent.keyDown(window, { key: "Escape" }));
    expect(screen.getByTestId("composer-discard-dialog")).toBeTruthy();
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();

    // Keep editing dismisses the prompt, composer stays open.
    fireEvent.click(screen.getByTestId("composer-discard-cancel"));
    expect(screen.queryByTestId("composer-discard-dialog")).toBeNull();
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();

    // Escape with the prompt open cancels the topmost surface (the prompt),
    // never the composer underneath it.
    act(() => fireEvent.keyDown(window, { key: "Escape" }));
    expect(screen.getByTestId("composer-discard-dialog")).toBeTruthy();
    act(() => fireEvent.keyDown(window, { key: "Escape" }));
    expect(screen.queryByTestId("composer-discard-dialog")).toBeNull();
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();

    // Discard clears + closes.
    act(() => fireEvent.keyDown(window, { key: "Escape" }));
    fireEvent.click(screen.getByTestId("composer-discard-confirm"));
    expect(screen.queryByTestId("full-screen-composer")).toBeNull();
  });

  it("traps focus in the discard prompt (topmost trap wins over the drawer's)", () => {
    render(<AttachHarness />);
    const input = screen.getByTestId("composer-file-input") as HTMLInputElement;
    act(() => fireEvent.change(input, { target: { files: [pngFile("a.png")] } }));
    act(() => fireEvent.keyDown(window, { key: "Escape" }));

    const dialog = screen.getByTestId("composer-discard-dialog");
    // Cancel is auto-focused on open.
    expect(document.activeElement).toBe(screen.getByTestId("composer-discard-cancel"));
    // Tab from the confirm (last control) wraps inside the dialog, not into
    // the composer drawer behind it.
    screen.getByTestId("composer-discard-confirm").focus();
    fireEvent.keyDown(screen.getByTestId("composer-discard-confirm"), { key: "Tab" });
    expect(dialog.contains(document.activeElement)).toBe(true);
  });
});
