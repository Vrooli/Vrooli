import {
  Crop,
  FileType2,
  FlipHorizontal2,
  Frame,
  Image,
  Info,
  Maximize2,
  Minimize2,
  RotateCw,
  Ruler,
  SlidersHorizontal,
  Type,
  Wand2,
  type LucideIcon,
} from "lucide-react";

import { strings } from "../../consts/strings";
import type { PositionToken } from "../../components/ui/position-picker";

type OpStrings = typeof strings.workspace.op;
type OpStringEntry = OpStrings[keyof OpStrings];

/** Per-op presentation: friendly label key, one-line description key, icon. */
export interface OpPresentation {
  labelKey: OpStringEntry["label"];
  descKey: OpStringEntry["desc"];
  Icon: LucideIcon;
}

/**
 * Maps each deterministic op to its humanized label/description i18n keys and a
 * lucide icon. The keys resolve through the typed registry, so a renamed op
 * string fails the build at this table.
 */
export const OP_CATALOG: Readonly<Record<string, OpPresentation>> = {
  resize: { labelKey: strings.workspace.op.resize.label, descKey: strings.workspace.op.resize.desc, Icon: Maximize2 },
  crop: { labelKey: strings.workspace.op.crop.label, descKey: strings.workspace.op.crop.desc, Icon: Crop },
  rotate: { labelKey: strings.workspace.op.rotate.label, descKey: strings.workspace.op.rotate.desc, Icon: RotateCw },
  flip: { labelKey: strings.workspace.op.flip.label, descKey: strings.workspace.op.flip.desc, Icon: FlipHorizontal2 },
  deskew: { labelKey: strings.workspace.op.deskew.label, descKey: strings.workspace.op.deskew.desc, Icon: Ruler },
  thumbnail: { labelKey: strings.workspace.op.thumbnail.label, descKey: strings.workspace.op.thumbnail.desc, Icon: Image },
  canvas: { labelKey: strings.workspace.op.canvas.label, descKey: strings.workspace.op.canvas.desc, Icon: Frame },
  adjust: { labelKey: strings.workspace.op.adjust.label, descKey: strings.workspace.op.adjust.desc, Icon: SlidersHorizontal },
  filter: { labelKey: strings.workspace.op.filter.label, descKey: strings.workspace.op.filter.desc, Icon: Wand2 },
  convert: { labelKey: strings.workspace.op.convert.label, descKey: strings.workspace.op.convert.desc, Icon: FileType2 },
  compress: { labelKey: strings.workspace.op.compress.label, descKey: strings.workspace.op.compress.desc, Icon: Minimize2 },
  overlay: { labelKey: strings.workspace.op.overlay.label, descKey: strings.workspace.op.overlay.desc, Icon: Type },
  metadata: { labelKey: strings.workspace.op.metadata.label, descKey: strings.workspace.op.metadata.desc, Icon: Info },
};

/** Presentation for an op, or `undefined` if it is unknown. */
export const opPresentation = (operation: string): OpPresentation | undefined => OP_CATALOG[operation];

/** A translatable key path from the typed string registry. */
type FitKey = (typeof strings.workspace.fitOption)[keyof typeof strings.workspace.fitOption];
type AxisKey = (typeof strings.workspace.axisOption)[keyof typeof strings.workspace.axisOption];
type FilterKey = (typeof strings.workspace.filterOption)[keyof typeof strings.workspace.filterOption];
type PositionNameKey =
  (typeof strings.workspace.position.name)[keyof typeof strings.workspace.position.name];

/** Label key for each `fit` option token (resize). */
export const FIT_OPTION_LABEL: Readonly<Record<string, FitKey>> = {
  fit: strings.workspace.fitOption.fit,
  fill: strings.workspace.fitOption.fill,
  stretch: strings.workspace.fitOption.stretch,
};

/** Label key for each `axis` option token (flip). */
export const AXIS_OPTION_LABEL: Readonly<Record<string, AxisKey>> = {
  horizontal: strings.workspace.axisOption.horizontal,
  vertical: strings.workspace.axisOption.vertical,
};

/** Label key + CSS approximation for each filter token (filter op). */
export const FILTER_OPTION: Readonly<Record<string, { labelKey: FilterKey; css: string }>> = {
  grayscale: { labelKey: strings.workspace.filterOption.grayscale, css: "grayscale(1)" },
  sepia: { labelKey: strings.workspace.filterOption.sepia, css: "sepia(1)" },
  invert: { labelKey: strings.workspace.filterOption.invert, css: "invert(1)" },
  blur: { labelKey: strings.workspace.filterOption.blur, css: "blur(2px)" },
  sharpen: { labelKey: strings.workspace.filterOption.sharpen, css: "contrast(1.4) saturate(1.2)" },
};

/**
 * Label key for each of the nine position/gravity tokens. The backend tokens
 * are hyphenated (`top-left`); the i18n keys are camelCase (`topLeft`) so the
 * unused-key audit can trace literal accessors.
 */
export const POSITION_NAME_LABEL: Readonly<Record<PositionToken, PositionNameKey>> = {
  "top-left": strings.workspace.position.name.topLeft,
  top: strings.workspace.position.name.top,
  "top-right": strings.workspace.position.name.topRight,
  left: strings.workspace.position.name.left,
  center: strings.workspace.position.name.center,
  right: strings.workspace.position.name.right,
  "bottom-left": strings.workspace.position.name.bottomLeft,
  bottom: strings.workspace.position.name.bottom,
  "bottom-right": strings.workspace.position.name.bottomRight,
};

/** Label key for each crop aspect preset (referenced literally for tracing). */
export const ASPECT_LABEL = {
  free: strings.workspace.crop.aspect.free,
  square: strings.workspace.crop.aspect.square,
  standard: strings.workspace.crop.aspect.standard,
  wide: strings.workspace.crop.aspect.wide,
  original: strings.workspace.crop.aspect.original,
} as const;
