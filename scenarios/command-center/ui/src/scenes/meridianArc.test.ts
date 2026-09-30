import { describe, expect, it } from "vitest";
import { destinationForCountry } from "./meridianArc";

describe("meridian country placement", () => {
  it("resolves ISO country keys and leaves unknown traffic unmapped", () => {
    expect(destinationForCountry("US")).toEqual([-100, 38]);
    expect(destinationForCountry("GB")).toEqual([-3.4, 55.4]);
    expect(destinationForCountry("unknown")).toBeNull();
  });
});
