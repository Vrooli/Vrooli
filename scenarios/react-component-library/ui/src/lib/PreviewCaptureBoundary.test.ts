import { beforeEach, describe, expect, it } from "vitest";

import harnessSource from "../../../api/handlers/preview/assets/harness.js?raw";

function sourceDeclaration(start: string, end: string): string {
  const from = harnessSource.indexOf(start);
  const to = harnessSource.indexOf(end, from);
  expect(from, `missing source declaration: ${start}`).not.toBe(-1);
  expect(to, `missing source boundary: ${end}`).not.toBe(-1);
  return harnessSource.slice(from, to).trim();
}

interface HarnessRuntime {
  setHarnessState: (state: string) => void;
  syncPortalCaptureBoundary: () => HTMLElement | null;
}

function installHarnessRuntime(root: HTMLElement): HarnessRuntime {
  const declarations = [
    sourceDeclaration("const setHarnessState = (state) => {", "const showPreviewError ="),
    sourceDeclaration("const syncPortalCaptureBoundary = () => {", "const measurePaintedContent ="),
  ].join("\n");
  const createRuntime = new Function(
    "document",
    "harnessRoot",
    "getComputedStyle",
    "captureTheme",
    `${declarations}; return { setHarnessState, syncPortalCaptureBoundary };`,
  ) as (
    document: Document,
    harnessRoot: HTMLElement,
    getComputedStyle: typeof window.getComputedStyle,
    captureTheme: string | undefined,
  ) => HarnessRuntime;
  return createRuntime(document, root, window.getComputedStyle.bind(window), undefined);
}

describe("preview capture boundary", () => {
  beforeEach(() => {
    document.body.replaceChildren();
  });

  it("does not keep an unmounted dialog active via its readiness marker", () => {
    const root = document.createElement("main");
    root.dataset.rclStoryStatus = "passed";
    const sheet = document.createElement("section");
    sheet.dataset.previewSheet = "specimen";
    sheet.dataset.previewCaptureBoundary = "component-sheet";
    root.appendChild(sheet);
    document.body.appendChild(root);

    const portal = document.createElement("div");
    portal.id = "rcl-layer-root";
    const dialog = document.createElement("div");
    dialog.dataset.overlayContent = "true";
    dialog.setAttribute("role", "dialog");
    portal.appendChild(dialog);
    document.body.appendChild(portal);

    const runtime = installHarnessRuntime(root);
    runtime.setHarnessState("ready");
    expect(portal.dataset.previewCaptureBoundary).toBe("component-sheet");
    expect(portal.querySelector("[data-preview-readiness-marker]")?.getAttribute("data-preview-ready")).toBe("true");
    expect(sheet.querySelector("[data-preview-readiness-marker]")).toBeNull();

    // The renderer removes the dialog subtree, while the harness-owned marker
    // remains until the next state synchronization.
    dialog.remove();
    expect(portal.querySelector("[data-preview-readiness-marker]")).not.toBeNull();
    runtime.setHarnessState("loading");
    expect(runtime.syncPortalCaptureBoundary()).toBe(sheet);
    expect(sheet.dataset.previewCaptureBoundary).toBe("component-sheet");
    expect(sheet.dataset.previewSheet).toBe("specimen");
    expect(sheet.querySelector<HTMLElement>("[data-preview-readiness-marker]")?.dataset.experienceState).toBe("loading");
    expect(portal.querySelector("[data-preview-readiness-marker]")).toBeNull();
    expect(portal.hasAttribute("data-preview-capture-boundary")).toBe(false);

    portal.appendChild(document.createElement("div")).setAttribute("data-overlay-content", "true");
    runtime.setHarnessState("ready");
    expect(runtime.syncPortalCaptureBoundary()).toBe(portal);
    expect(portal.querySelector("[data-preview-readiness-marker]")?.getAttribute("data-preview-ready")).toBe("true");
    expect(sheet.querySelector("[data-preview-readiness-marker]")).toBeNull();
  });
});
