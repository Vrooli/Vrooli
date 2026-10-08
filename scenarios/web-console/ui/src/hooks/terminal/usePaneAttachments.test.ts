import { describe, expect, it, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { usePaneAttachments } from "./usePaneAttachments";
import type { GateResult } from "../../components/terminal/inputGate";
import type { ClipboardEvent, DragEvent } from "react";

const uploadAndInject = vi.fn(async () => {});
vi.mock("../useImageUpload", () => ({
  useImageUpload: () => ({ uploadAndInject, uploading: false, error: null }),
}));

function sentGate(): GateResult {
  return { status: "sent", offset: 1 };
}

describe("usePaneAttachments", () => {
  it("routes image paste and the context-menu upload action to the seam", () => {
    const close = vi.fn();
    const submitInput = vi.fn(sentGate);
    const { result } = renderHook(() => usePaneAttachments("session", submitInput, close));
    const file = new File(["image"], "shot.png", { type: "image/png" });
    const preventDefault = vi.fn();
    const paste = {
      clipboardData: { items: [{ type: "image/png", getAsFile: () => file }] },
      preventDefault,
    } as unknown as ClipboardEvent;

    act(() => { result.current.handlePaste(paste); });
    expect(preventDefault).toHaveBeenCalled();
    expect(uploadAndInject).toHaveBeenCalledWith(file);

    act(() => { result.current.handleCtxUploadImage(); });
    expect(close).toHaveBeenCalled();
  });

  it("uploads any non-executable dropped file and skips executables", () => {
    const submitInput = vi.fn(sentGate);
    const { result } = renderHook(() => usePaneAttachments("session", submitInput, vi.fn()));
    uploadAndInject.mockClear();

    const text = new File(["notes"], "notes.txt", { type: "text/plain" });
    const video = new File(["clip"], "clip.mp4", { type: "video/mp4" });
    const exe = new File(["MZ"], "evil.exe", { type: "application/octet-stream" });
    const drop = {
      preventDefault: vi.fn(),
      dataTransfer: { files: [text, video, exe] },
    } as unknown as DragEvent;

    act(() => { result.current.handleDrop(drop); });

    expect(uploadAndInject).toHaveBeenCalledWith(text);
    expect(uploadAndInject).toHaveBeenCalledWith(video);
    expect(uploadAndInject).toHaveBeenCalledTimes(2);
  });
});
