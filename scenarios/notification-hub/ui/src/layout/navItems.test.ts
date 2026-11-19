import { describe, expect, it } from "vitest";

import { NAV_ITEMS, isCurrent } from "./navItems";

const currentKeys = (pathname: string) => NAV_ITEMS.filter((item) => isCurrent(item, pathname)).map((item) => item.key);

describe("navigation sections", () => {
  it("offers the four destinations in order, with no Delivery item of its own", () => {
    expect(NAV_ITEMS.map((item) => item.key)).toEqual(["asks", "dashboard", "history", "settings"]);
  });

  it("opens on Updates at the index route and Needs you at a decision", () => {
    expect(currentKeys("/")).toEqual(["dashboard"]);
    expect(currentKeys("/asks")).toEqual(["asks"]);
    expect(currentKeys("/asks/ask-1")).toEqual(["asks"]);
  });

  it("keeps Updates current across the Delivery pages", () => {
    expect(currentKeys("/updates")).toEqual(["dashboard"]);
    expect(currentKeys("/delivery")).toEqual(["dashboard"]);
    expect(currentKeys("/delivery/ntf-1")).toEqual(["dashboard"]);
  });

  it("lets a page claim its destination over what the URL says", () => {
    const claimedKeys = (pathname: string, claimed: "asks" | "history") => NAV_ITEMS.filter((item) => isCurrent(item, pathname, claimed)).map((item) => item.key);
    expect(claimedKeys("/asks/ask-1", "history")).toEqual(["history"]);
    expect(claimedKeys("/asks/ask-1", "asks")).toEqual(["asks"]);
  });

  it("keeps History and Settings current for their own pages only", () => {
    expect(currentKeys("/history")).toEqual(["history"]);
    expect(currentKeys("/settings")).toEqual(["settings"]);
    expect(currentKeys("/settings/reach")).toEqual(["settings"]);
    expect(currentKeys("/historical")).toEqual([]);
  });
});
