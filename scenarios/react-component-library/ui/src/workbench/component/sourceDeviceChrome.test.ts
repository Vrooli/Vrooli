import { describe, expect, it } from "vitest";

import { node } from "../templates/spec";
import { deviceChromeVisibility } from "./sourceDeviceChrome";

describe("deviceChromeVisibility", () => {
  it("keeps standard phone chrome when the source has no authored equivalent", () => {
    expect(deviceChromeVisibility(node("page", "page", {}))).toEqual({
      statusBar: true,
      island: true,
    });
  });

  it("suppresses only phone chrome already present in the imported source", () => {
    const imported = node("page", "page", {
      sourceHtml: '<main><div class="statusbar">9:41</div><div class="island"></div></main>',
    });
    expect(deviceChromeVisibility(imported)).toEqual({
      statusBar: false,
      island: false,
    });
  });

  it("keeps the island when source markup only has a status bar", () => {
    const imported = node("page", "page", {
      sourceHtml: "<main><div data-device-status-bar>9:41</div></main>",
    });
    expect(deviceChromeVisibility(imported)).toEqual({
      statusBar: false,
      island: true,
    });
  });

  it("suppresses device chrome for an expanded editable HTML import tree", () => {
    const imported = node("page", "page", {}, [
      node("page-html-32-row", "row", {}, [node("page-html-33-text", "text", { text: "9:41" })]),
    ]);
    expect(deviceChromeVisibility(imported)).toEqual({ statusBar: false, island: false });
  });
});
