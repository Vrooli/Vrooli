import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { FileViewerProvider } from "../i18n/context.js";
import { useReaderFontSize } from "../preview/useReaderFontSize.js";
import { clampReaderFontSize, READER_FONT_DEFAULT, READER_FONT_MAX, READER_FONT_MIN } from "./typography.js";

describe("clampReaderFontSize", () => {
  it("rounds into the supported range", () => {
    expect(clampReaderFontSize(3)).toBe(READER_FONT_MIN);
    expect(clampReaderFontSize(99)).toBe(READER_FONT_MAX);
    expect(clampReaderFontSize(16.6)).toBe(17);
  });
});

describe("useReaderFontSize", () => {
  it("keeps its own size, starting at the default, when the host supplies none", () => {
    const { result } = renderHook(() => useReaderFontSize());
    expect(result.current[0]).toBe(READER_FONT_DEFAULT);
    act(() => { result.current[1](22); });
    expect(result.current[0]).toBe(22);
    act(() => { result.current[1](500); });
    expect(result.current[0]).toBe(READER_FONT_MAX);
  });

  it("reads and writes the host's persisted preference when it supplies one", () => {
    const onChange = vi.fn();
    const { result } = renderHook(() => useReaderFontSize(), {
      wrapper: ({ children }) => <FileViewerProvider readerFontSize={{ value: 20, onChange }}>{children}</FileViewerProvider>,
    });
    expect(result.current[0]).toBe(20);
    act(() => { result.current[1](24.4); });
    expect(onChange).toHaveBeenCalledWith(24);
    expect(result.current[0]).toBe(20);
  });
});
