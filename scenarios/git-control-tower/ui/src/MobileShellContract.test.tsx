import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { StatusBarFill, chromeTheme } from "@vrooli/react-component-library/ChromeTheme";
import indexHTML from "../index.html?raw";
import app from "./App.tsx?raw";
import diffViewer from "./components/DiffViewer.tsx?raw";
import fileList from "./components/FileList.tsx?raw";

describe("GCT-MOBILE embedded shell regression contract", () => {
  it("GCT-MOBILE-001/002/006: declares a viewport-filling safe-area shell and chrome contract", () => {
    expect(indexHTML).toContain('name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover"');
    expect(indexHTML).toContain('name="theme-color" content="#0f172a"');
    expect(app).toContain('data-mobile-shell="true"');
    expect(app).toContain('data-testid="gct-mobile-content"');
  });

  it("GCT-MOBILE-002: resolves theme-color and the RCL status strip to one opaque value", () => {
    document.head.innerHTML = '<meta name="theme-color" content="">';
    chromeTheme._reset();
    render(<StatusBarFill testId="status-bar-fill" />);
    chromeTheme.setBase({ statusColor: "#0f172a" });
    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute("content")).toBe("#0f172a");
    expect(document.documentElement.style.getPropertyValue("--rcl-status-fill")).toBe("#0f172a");
    expect(app).toContain("chromeTheme.setBase({ statusColor: GCT_CHROME_COLOR, fillColor: GCT_CHROME_COLOR })");
  });

  it("GCT-MOBILE-005: uses a bounded flex footer instead of an overlay action bar", () => {
    expect(diffViewer).toContain('<ScrollArea className="flex-1 min-h-0"');
    expect(diffViewer).toContain('className="flex-none p-3 bg-slate-900/95 backdrop-blur-sm border-t border-slate-800" data-testid="diff-mobile-actions"');
    expect(diffViewer).not.toContain("Mobile spacer to account for fixed action bar");
  });

  it("GCT-MOBILE-004: keeps the selection toolbar inside the owned Changes scroller", () => {
    const scrollRegion = fileList.indexOf('data-testid="changes-scroll-region"');
    const toolbar = fileList.indexOf('data-testid="mobile-selection-toolbar"');
    expect(scrollRegion).toBeGreaterThan(-1);
    expect(toolbar).toBeGreaterThan(scrollRegion);
    const toolbarMarkup = fileList.slice(toolbar, fileList.indexOf("</div>", toolbar));
    expect(toolbarMarkup).not.toContain("sticky top-");
  });
});
