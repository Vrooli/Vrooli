import { describe, expect, it } from "vitest";
import { operatorAnswerValue } from "./operatorAnswer";

describe("operatorAnswerValue", () => {
  it("preserves primitive and list answers as readable strings", () => {
    expect(operatorAnswerValue("region-a")).toBe("region-a");
    expect(operatorAnswerValue(7)).toBe("7");
    expect(operatorAnswerValue(false)).toBe("false");
    expect(operatorAnswerValue(["one", "two"])).toBe("one,two");
    expect(operatorAnswerValue(null)).toBe("");
  });

  it("serializes object answers instead of leaking Object's default label", () => {
    expect(operatorAnswerValue({ region: "east", zones: [1, 2] })).toBe('{"region":"east","zones":[1,2]}');
  });
});
