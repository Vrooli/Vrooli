export const MAX_FONT_BYTES = 10 * 1024 * 1024;
export const MAX_FONT_EXPANSION = 32 * 1024 * 1024;
export const MAX_FONT_CHARACTERS = 131072;
export const FONT_PAGE_SIZE = 96;
export type FontError = "fetch" | "invalid" | "limit" | "unsupported" | "unavailable";
export type FontFormat = "ttf" | "otf" | "woff" | "woff2";
export interface FontAxis { tag: string; name: string; min: number; max: number; default: number }
export interface FontInfo { family: string; style: string; format: FontFormat; glyphs: number; unitsPerEm: number; characters: number[]; axes: FontAxis[] }
export function fontFormat(name: string): FontFormat | null { return /\.(ttf|otf|woff|woff2)$/i.exec(name)?.[1]?.toLowerCase() as FontFormat | undefined ?? null; }
export function fontError(cause: unknown): FontError {
  if (cause instanceof RangeError) return "limit";
  const message = cause instanceof Error ? cause.message : "";
  return (["fetch", "invalid", "limit", "unsupported", "unavailable"] as const).find(value => value === message) ?? "invalid";
}
export function unicodeLabel(code: number) { return `U+${code.toString(16).toUpperCase().padStart(4, "0")}`; }
export function filterFontCharacters(characters: number[], query: string): number[] {
  const term = query.trim();
  if (!term) return characters;
  if (/^(?:U\+|0x)[\da-f]{1,6}$/i.test(term)) { const prefix = term.replace(/^(?:U\+|0x)/i, "").toUpperCase(); const exact = parseInt(prefix, 16); if (characters.includes(exact)) return [exact]; return characters.filter(code => code.toString(16).toUpperCase().padStart(4, "0").startsWith(prefix)); }
  const requested = new Set(Array.from(term, character => character.codePointAt(0)));
  return characters.filter(code => requested.has(code));
}
// Exclude invisible controls from the grid; their mappings remain in metadata.
export function visibleFontCharacter(code: number) {
  // Keep private-use mappings: icon fonts commonly store real artwork there.
  return !/[\p{White_Space}\p{Cc}\p{Cs}\p{Default_Ignorable_Code_Point}]/u.test(String.fromCodePoint(code));
}

export function fontSample(characters: number[], candidates: string[]): string {
  const mapped = new Set(characters);
  for (const candidate of candidates) {
    const letters = Array.from(candidate).filter(char => !/\p{White_Space}/u.test(char));
    if (letters.length && letters.filter(char => mapped.has(char.codePointAt(0) ?? 0)).length / letters.length >= 0.8) return candidate;
  }
  return characters.filter(visibleFontCharacter).slice(0, 32).map(code => String.fromCodePoint(code)).join(" ");
}
export function fontHero(characters: number[], sample: string): string {
  if (characters.includes(65) && characters.includes(97)) return "Aa";
  const mapped = new Set(characters);
  return Array.from(sample).filter(char => { const code = char.codePointAt(0) ?? 0; return mapped.has(code) && visibleFontCharacter(code); }).slice(0, 2).join("");
}
