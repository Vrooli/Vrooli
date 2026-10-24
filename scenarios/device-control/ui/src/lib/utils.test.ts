import { describe, expect, it } from "vitest";

import { cn } from "./utils";

describe("cn", () => {
  it("merges conditional utility classes", () => {
    const hidden: false | "hidden" = window.location.hash === "#hidden" && "hidden";
    expect(cn("px-2", hidden && "hidden", "px-4")).toBe("px-4");
  });
});
