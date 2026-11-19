import { strings } from "../../consts/strings";

type SizeKey = keyof typeof strings.workspace.create.size;

/** A canvas-size preset offered for prompt-driven generation. */
export interface SizePreset {
  key: SizeKey;
  labelKey: (typeof strings.workspace.create.size)[SizeKey];
  width: number;
  height: number;
}

/**
 * The default size preset (a standalone const so callers always have a
 * non-undefined fallback under `noUncheckedIndexedAccess`). Square is the safe
 * default for the seeded SD-family models.
 */
export const DEFAULT_SIZE_PRESET: SizePreset = {
  key: "square",
  labelKey: strings.workspace.create.size.square,
  width: 512,
  height: 512,
};

/** Size presets for text-to-image; portrait/landscape keep a 2:3 / 3:2 ratio. */
export const SIZE_PRESETS: readonly SizePreset[] = [
  DEFAULT_SIZE_PRESET,
  { key: "portrait", labelKey: strings.workspace.create.size.portrait, width: 512, height: 768 },
  { key: "landscape", labelKey: strings.workspace.create.size.landscape, width: 768, height: 512 },
];

/** The variation-count options the inspector offers (1..4). */
export const VARIATION_OPTIONS = ["1", "2", "3", "4"] as const;

/**
 * The backend emits all output blob keys for an N-variations job in the
 * terminal job message as `variations: [k0 k1 k2]` (Go `%v` of a `[]string`;
 * keys carry no spaces). The primary `result_ref` is `k0`, which is also the
 * first element of that list — so parsing the message yields the full set.
 * When the message has no such marker (variations == 1, or an older job), fall
 * back to the single primary ref.
 */
const VARIATIONS_RE = /variations:\s*\[([^\]]*)\]/;

export function parseVariationKeys(message: string, primaryRef: string): string[] {
  const match = VARIATIONS_RE.exec(message);
  if (match && match[1]) {
    const keys = match[1].trim().split(/\s+/).filter(Boolean);
    if (keys.length > 0) {
      return keys;
    }
  }
  return primaryRef ? [primaryRef] : [];
}
