import { describe, expect, it } from "vitest";

import { cn } from "./utils";

describe("cn", () => {
  it("combines conditional classes and resolves Tailwind conflicts", () => {
    const conditionalClass = (enabled: boolean) => enabled ? "hidden" : false;
    expect(cn("p-2", conditionalClass(false), "p-4")).toContain("p-4");
    expect(cn("text-sm", { "font-bold": true, hidden: false })).toContain("font-bold");
  });
});
