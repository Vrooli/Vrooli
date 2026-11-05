// Minimal boundary for fontkit's metadata-only worker use. No glyph outline or
// filesystem API crosses this boundary. Upstream 2.0.4 does not publish types.
declare module "fontkit" {
  interface TableRecord { length: number; transformLength?: number }
  interface Font {
    directory: { tables: Record<string, TableRecord> };
    _getTableStream(tag: string): { buffer: Uint8Array; pos: number } | null;
    familyName: string; subfamilyName: string; numGlyphs: number; unitsPerEm: number;
    variationAxes: Record<string, { name: string; min: number; max: number; default: number }>;
    hasGlyphForCodePoint(code: number): boolean;
  }
  export function create(bytes: Uint8Array): Font;
}
