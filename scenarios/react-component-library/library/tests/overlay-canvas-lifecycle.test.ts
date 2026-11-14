import { describe, expect, it } from "vitest";
import { layerManager } from "@vrooli/react-component-library/LayerManager/2.0.2";

describe("overlay canvas lifecycle", () => {
  it("holds the sheet canvas across nested overlays and releases it after the last sheet closes", () => {
    const closeSheet = layerManager.push({ id: "sheet", kind: "sheet", modal: true });
    const closeMenu = layerManager.push({ id: "menu", kind: "menu" });
    const closeDrawer = layerManager.push({ id: "drawer", kind: "drawer", modal: true });
    try {
      expect(document.documentElement).toHaveAttribute("data-rcl-overlay-canvas");
      closeSheet();
      expect(document.documentElement).toHaveAttribute("data-rcl-overlay-canvas");
      closeDrawer();
      expect(document.documentElement).not.toHaveAttribute("data-rcl-overlay-canvas");
      expect(layerManager.top()?.id).toBe("menu");
    } finally { closeMenu(); closeSheet(); closeDrawer(); }
  });

  it("leaves the canvas unchanged for desktop dialogs and menus", () => {
    const closeDialog = layerManager.push({ id: "dialog", kind: "dialog", modal: true });
    const closeMenu = layerManager.push({ id: "menu", kind: "menu" });
    try { expect(document.documentElement).not.toHaveAttribute("data-rcl-overlay-canvas"); }
    finally { closeMenu(); closeDialog(); }
  });
});
