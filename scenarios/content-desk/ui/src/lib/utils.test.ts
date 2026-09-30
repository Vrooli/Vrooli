import { describe, expect, it } from "vitest";
import { cn } from "./utils";

describe("cn", () => {
  it("merges conditional Tailwind classes deterministically", () => {
    expect(cn("px-2", false && "hidden", "px-4")).toBe("px-4");
  });
});
