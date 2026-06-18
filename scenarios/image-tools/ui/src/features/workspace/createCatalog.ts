import { ImagePlus, Replace, Stamp, Eraser, Wand, type LucideIcon } from "lucide-react";

import { strings } from "../../consts/strings";

type CreateOpStrings = typeof strings.workspace.createOp;
type CreateOpEntry = CreateOpStrings[keyof CreateOpStrings];

/** Per-create-op presentation: friendly label/description keys and an icon. */
export interface CreateOpPresentation {
  labelKey: CreateOpEntry["label"];
  descKey: CreateOpEntry["desc"];
  Icon: LucideIcon;
}

/**
 * Maps each generation op (discovered from `AIService.ListAIOperations`,
 * category `generation`) to its humanized label/description keys and a lucide
 * icon. Driven by the live catalog, so a backend that adds a generation op
 * still renders (via the fallback) without a code change here.
 */
export const CREATE_CATALOG: Readonly<Record<string, CreateOpPresentation>> = {
  text_to_image: {
    labelKey: strings.workspace.createOp.text_to_image.label,
    descKey: strings.workspace.createOp.text_to_image.desc,
    Icon: ImagePlus,
  },
  image_to_image: {
    labelKey: strings.workspace.createOp.image_to_image.label,
    descKey: strings.workspace.createOp.image_to_image.desc,
    Icon: Replace,
  },
  edit_instruct: {
    labelKey: strings.workspace.createOp.edit_instruct.label,
    descKey: strings.workspace.createOp.edit_instruct.desc,
    Icon: Wand,
  },
  inpaint: {
    labelKey: strings.workspace.createOp.inpaint.label,
    descKey: strings.workspace.createOp.inpaint.desc,
    Icon: Stamp,
  },
  object_removal: {
    labelKey: strings.workspace.createOp.object_removal.label,
    descKey: strings.workspace.createOp.object_removal.desc,
    Icon: Eraser,
  },
};

/** Fallback icon for a generation op the catalog doesn't yet name. */
export const CREATE_FALLBACK_ICON: LucideIcon = ImagePlus;

/** Presentation for a known create op, or `undefined` if it isn't in the catalog. */
export const createPresentation = (operation: string): CreateOpPresentation | undefined =>
  CREATE_CATALOG[operation];

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
