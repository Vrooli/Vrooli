import { describe, expect, it } from "vitest";
import { Slider } from "../../../../components/Slider/versions/1.2.0/Slider";

describe("Slider drag integration", () => {
  it("publishes the shared-drag slider", () => expect(Slider).toBeDefined());
});
