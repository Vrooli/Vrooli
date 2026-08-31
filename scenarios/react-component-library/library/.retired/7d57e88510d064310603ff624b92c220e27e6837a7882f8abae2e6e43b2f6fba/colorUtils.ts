const HEX_COLOR = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/;

export function isHexColor(value: string | null | undefined): value is string {
  return typeof value === "string" && HEX_COLOR.test(value);
}

export function isLightColor(value: string | null | undefined): boolean {
  if (!isHexColor(value)) return false;
  const hex =
    value.length === 4
      ? value
          .slice(1)
          .split("")
          .map((part) => part + part)
          .join("")
      : value.slice(1);
  const rgb = [0, 2, 4].map(
    (offset) => Number.parseInt(hex.slice(offset, offset + 2), 16) / 255,
  );
  const [red = 0, green = 0, blue = 0] = rgb.map((channel) =>
    channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2,
  );
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue > 0.5;
}

export function parseColorValue(value: string | null | undefined): {
  colors: string[];
  transparent: boolean;
} {
  const colors = (value ?? "")
    .split("|")
    .map((part) => part.trim())
    .filter(isHexColor)
    .slice(0, 2);
  return { colors, transparent: colors.length === 0 };
}

export function serializeColorValue(colors: readonly string[]): string {
  const valid = colors.filter(isHexColor).slice(0, 2);
  return valid.length ? valid.join("|") : "transparent";
}
