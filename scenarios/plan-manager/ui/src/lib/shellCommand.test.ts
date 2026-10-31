import { describe, expect, it } from "vitest";

import { shellCommand, shellQuote } from "./shellCommand";

describe("shellCommand", () => {
  it("quotes only args that need shell escaping", () => {
    expect(shellQuote("plain/path")).toBe("plain/path");
    expect(shellQuote("needs review")).toBe("'needs review'");
    expect(shellQuote("it's")).toBe("'it'\\''s'");
  });

  it("formats optional command prefixes", () => {
    expect(shellCommand(["exec", "status", "e1"], ["vrooli", "scenario", "plan-manager"])).toBe(
      "vrooli scenario plan-manager exec status e1",
    );
  });
});
