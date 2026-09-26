import { describe, expect, it } from "vitest";

import { cn } from "./utils";

describe("cn", () => {
  it("merges conflicting Tailwind classes and ignores falsey values", () => {
    expect(cn("px-2", false && "px-4", "px-4")).toBe("px-4");
  });
});
