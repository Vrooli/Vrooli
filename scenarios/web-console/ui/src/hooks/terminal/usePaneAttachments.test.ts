import { describe, expect, it, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { usePaneAttachments } from "./usePaneAttachments";
import type { GateResult } from "../../components/terminal/inputGate";
import type { ClipboardEvent } from "react";

const uploadAndInject = vi.fn(async () => {});
vi.mock("../useImageUpload", () => ({
  useImageUpload: () => ({ uploadAndInject, uploading: false, error: null }),
}));

describe("usePaneAttachments", () => {
  it("routes image paste and drop files to the upload seam", () => {
    const close = vi.fn();
    const submitInput = vi.fn((): GateResult => ({ status: "sent", offset: 1 }));
    const { result } = renderHook(() => usePaneAttachments("session", submitInput, close));
    const file = new File(["image"], "shot.png", { type: "image/png" });
    const paste = { clipboardData: { items: [{ type: "image/png", getAsFile: () => file }] }, preventDefault: vi.fn() } as unknown as ClipboardEvent;
    act(() => result.current.handlePaste(paste));
    expect(paste.preventDefault).toHaveBeenCalled();
    expect(uploadAndInject).toHaveBeenCalledWith(file);
    act(() => result.current.handleCtxUploadImage());
    expect(close).toHaveBeenCalled();
  });
});
