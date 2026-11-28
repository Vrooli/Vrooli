import { describe, expect, it } from "vitest";

import { colorHex, rgbLabel, scaledChannels } from "./colorEvidence";

describe("color evidence formatting", () => {
  it("preserves the intended hue while exposing the dim scaled channel payload", () => {
    const amber = { red: 217, green: 154, blue: 36 };
    const scaled = scaledChannels(amber, 25);

    expect(colorHex(amber)).toBe("#D99A24");
    expect(scaled).toEqual({ red: 54, green: 39, blue: 9 });
    expect(colorHex(scaled)).toBe("#362709");
    expect(rgbLabel(scaled)).toBe("54, 39, 9");
  });

  it("rounds each channel like the backend and clamps brightness to its supported range", () => {
    expect(scaledChannels({ red: 1, green: 3, blue: 5 }, 50)).toEqual({ red: 1, green: 2, blue: 3 });
    expect(scaledChannels({ red: 255, green: 64, blue: 32 }, 150)).toEqual({ red: 255, green: 64, blue: 32 });
    expect(scaledChannels({ red: 255, green: 64, blue: 32 }, -1)).toEqual({ red: 0, green: 0, blue: 0 });
  });

  it("formats missing colors as missing instead of black", () => {
    expect(colorHex(undefined)).toBeUndefined();
  });
});
