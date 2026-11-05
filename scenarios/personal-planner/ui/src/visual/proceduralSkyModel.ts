export type ZodiacName = "aries" | "taurus" | "gemini" | "cancer" | "leo" | "virgo" | "libra" | "scorpio" | "sagittarius" | "capricorn" | "aquarius" | "pisces";

export type SkyStar = {
  x: number;
  y: number;
  radius: number;
  opacity: number;
  color: "warm" | "neutral" | "cool";
};

const SKY_WIDTH = 1000;
const SKY_HEIGHT = 700;

function seededRandom(seed: number): () => number {
  let value = seed >>> 0;
  return () => {
    value += 0x6d2b79f5;
    let next = value;
    next = Math.imul(next ^ (next >>> 15), next | 1);
    next ^= next + Math.imul(next ^ (next >>> 7), next | 61);
    return ((next ^ (next >>> 14)) >>> 0) / 4294967296;
  };
}

function gaussian(random: () => number): number {
  const left = Math.max(random(), Number.EPSILON);
  return Math.sqrt(-2 * Math.log(left)) * Math.cos(2 * Math.PI * random());
}

/**
 * A stable sky avoids the visual jump that a freshly-randomized particle field
 * creates on every route change. Most stars are deliberately tiny and dim;
 * only a short magnitude tail earns a visible glow.
 */
export function generateStarField(seed = 0x51a7f13d, count = 340): SkyStar[] {
  const random = seededRandom(seed);
  return Array.from({ length: count }, (_, index) => {
    const followsGalacticBand = index < Math.round(count * 0.44);
    const x = random() * SKY_WIDTH;
    const bandCenter = 610 - x * 0.49 + Math.sin(x / 145) * 24;
    const y = followsGalacticBand
      ? Math.max(12, Math.min(SKY_HEIGHT - 12, bandCenter + gaussian(random) * 82))
      : random() * SKY_HEIGHT;
    const magnitudeTail = Math.pow(random(), 6.2);
    const temperature = random();
    return {
      x,
      y,
      radius: 0.38 + magnitudeTail * 1.72,
      opacity: 0.2 + Math.pow(random(), 2.6) * 0.54 + magnitudeTail * 0.24,
      color: temperature < 0.12 ? "warm" : temperature > 0.86 ? "cool" : "neutral",
    };
  });
}

const ZODIAC_WINDOWS: ReadonlyArray<readonly [number, number, ZodiacName]> = [
  [120, 218, "aquarius"], [219, 320, "pisces"], [321, 419, "aries"],
  [420, 520, "taurus"], [521, 620, "gemini"], [621, 722, "cancer"],
  [723, 822, "leo"], [823, 922, "virgo"], [923, 1022, "libra"],
  [1023, 1121, "scorpio"], [1122, 1221, "sagittarius"], [1222, 1231, "capricorn"],
];

/** Conventional tropical-zodiac window, used as a quiet seasonal motif. */
export function zodiacForDate(date: Date): ZodiacName {
  const key = (date.getMonth() + 1) * 100 + date.getDate();
  if (key <= 119) return "capricorn";
  return ZODIAC_WINDOWS.find(([start, end]) => key >= start && key <= end)?.[2] ?? "capricorn";
}
