import { describe, expect, it } from "vitest";

import { cn } from "./utils";

describe("cn", () => {
  it("merges conditional Tailwind classes", () => {
    const optionalClass: string | false = false;
    expect(cn("rounded", optionalClass, "p-2", "p-4")).toBe("rounded p-4");
  });
});
