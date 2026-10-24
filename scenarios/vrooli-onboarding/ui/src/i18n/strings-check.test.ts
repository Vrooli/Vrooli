import { describe, expect, it } from "vitest";
import { findHardcodedStrings } from "../../scripts/strings-check.mjs";

describe("production UI copy", () => {
  it("keeps user-visible literals in the i18n resource", () => {
    expect(findHardcodedStrings()).toEqual([]);
  });
});
