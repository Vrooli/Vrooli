import { render, renderHook, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { en } from "../locales/en.js";
import { FileViewerProvider, useFileViewerHost, useFileViewerText } from "../context.js";
import { strings } from "../strings.js";

function Label() {
  const { t, locale, dir } = useFileViewerText();
  return <p data-locale={locale} data-dir={dir}>{t(strings.viewer.download)}</p>;
}

describe("FileViewerProvider", () => {
  it("renders English without a provider", () => {
    render(<Label />);
    expect(screen.getByText(en.viewer.download)).toHaveAttribute("data-locale", "en");
  });

  it("follows the provider's locale and its text direction", () => {
    render(<FileViewerProvider locale="ar-SA"><Label /></FileViewerProvider>);
    const label = document.querySelector("p");
    expect(label?.textContent).not.toBe(en.viewer.download);
    expect(label).toHaveAttribute("data-dir", "rtl");
    expect(label).toHaveAttribute("data-locale", "ar-SA");
  });

  it("falls back to English for a locale it has no catalog for", () => {
    render(<FileViewerProvider locale="de"><Label /></FileViewerProvider>);
    expect(screen.getByText(en.viewer.download)).toBeInTheDocument();
  });

  it("lets the host override direction, per-key copy and the whole lookup", () => {
    const { rerender } = render(<FileViewerProvider locale="en" dir="rtl" overrides={{ [strings.viewer.download]: "Save a copy" }}><Label /></FileViewerProvider>);
    expect(screen.getByText("Save a copy")).toHaveAttribute("data-dir", "rtl");
    rerender(<FileViewerProvider locale="en" translate={(key) => (key === strings.viewer.download ? "Adapter copy" : undefined)}><Label /></FileViewerProvider>);
    expect(screen.getByText("Adapter copy")).toBeInTheDocument();
  });

  it("formats the keys pseudo-locale as English while returning keys", () => {
    render(<FileViewerProvider locale="keys"><Label /></FileViewerProvider>);
    expect(screen.getByText("viewer.download")).toHaveAttribute("data-locale", "en");
  });

  it("exposes host preferences and hooks, and nothing by default", () => {
    expect(renderHook(() => useFileViewerHost()).result.current).toEqual({ readerFontSize: undefined, rewriteHref: undefined, loaders: {}, fileInfo: "bar" });
    const readerFontSize = { value: 20, onChange: () => {} };
    const rewriteHref = () => "x";
    const { result } = renderHook(() => useFileViewerHost(), { wrapper: ({ children }) => <FileViewerProvider readerFontSize={readerFontSize} rewriteHref={rewriteHref}>{children}</FileViewerProvider> });
    expect(result.current).toEqual({ readerFontSize, rewriteHref, loaders: {}, fileInfo: "bar" });
    const host = renderHook(() => useFileViewerHost(), { wrapper: ({ children }) => <FileViewerProvider fileInfo="host">{children}</FileViewerProvider> });
    expect(host.result.current.fileInfo).toBe("host");
  });

  it("exposes engine loaders, keeping the context stable when the host re-creates the loaders object", () => {
    const spreadsheet = () => Promise.reject(new Error("unused"));
    const { result, rerender } = renderHook(() => useFileViewerHost(), { wrapper: ({ children }) => <FileViewerProvider loaders={{ spreadsheet }}>{children}</FileViewerProvider> });
    const first = result.current.loaders;
    expect(first).toEqual({ spreadsheet, diagram: undefined });
    rerender();
    expect(result.current.loaders).toBe(first);
  });
});
