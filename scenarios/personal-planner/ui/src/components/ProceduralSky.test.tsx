import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../test-utils";
import { ProceduralSky } from "./ProceduralSky";
import { generateStarField, zodiacForDate } from "../visual/proceduralSkyModel";

describe("ProceduralSky", () => {
  it("generates a stable field with a natural range of size, brightness, and temperature", () => {
    const first = generateStarField(42, 240);
    const second = generateStarField(42, 240);

    expect(first).toEqual(second);
    expect(new Set(first.map((star) => star.radius.toFixed(2))).size).toBeGreaterThan(20);
    expect(new Set(first.map((star) => star.opacity.toFixed(2))).size).toBeGreaterThan(20);
    expect(new Set(first.map((star) => star.color))).toEqual(new Set(["warm", "neutral", "cool"]));
    expect(first.filter((star) => star.radius > 1.45).length).toBeLessThan(first.length * 0.12);
  });

  it("maps seasonal boundaries to the conventional zodiac motif", () => {
    expect(zodiacForDate(new Date(2026, 7, 22))).toBe("leo");
    expect(zodiacForDate(new Date(2026, 7, 23))).toBe("virgo");
    expect(zodiacForDate(new Date(2026, 8, 22))).toBe("virgo");
    expect(zodiacForDate(new Date(2026, 8, 23))).toBe("libra");
    expect(zodiacForDate(new Date(2026, 11, 31))).toBe("capricorn");
  });

  it("renders the selected constellation only when the scene asks for it", () => {
    const { container, rerender } = renderWithProviders(<ProceduralSky date={new Date(2026, 7, 23)} />);
    expect(container.querySelector(".sky-constellation")).toBeNull();

    rerender(<ProceduralSky showConstellation date={new Date(2026, 7, 23)} />);
    expect(container.querySelector('[data-constellation="virgo"]')).toBeTruthy();
    expect(container.querySelectorAll(".sky-constellation line").length).toBeGreaterThan(4);
  });
});
