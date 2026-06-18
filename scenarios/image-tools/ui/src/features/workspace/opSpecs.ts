import { strings } from "../../consts/strings";

/**
 * Declarative spec for each deterministic op's parameter form. The discovery
 * RPC (`ListOperations`) tells us *which* ops exist; this table tells the
 * Inspector *which fields* each op takes and *which humanized control* renders
 * it. The field `name` is the proto field name (snake_case) sent inside the
 * operation-keyed params object — see `runOp` in `api/ops.ts`.
 *
 * Keep this in sync with the proto `OpParams` oneof. Adding a field is a
 * one-row edit here plus a label key in `i18n/locales/*`. The `control` chooses
 * the primitive; `min/max/step/unit` feed sliders; `advanced` tucks a field
 * under the Inspector's "Advanced" disclosure (e.g. crop's numeric box, which
 * the canvas drag-box drives directly).
 */

/**
 * Which humanized primitive renders a field.
 * - `number` / `text` — plain inputs (last-resort fallback)
 * - `toggle` — on/off switch (was a checkbox)
 * - `segmented` — small pill group for a short closed set (`options`)
 * - `slider` — numeric range with value + per-control reset (`min/max/step/unit`)
 * - `position` — 3×3 gravity/position picker
 * - `color` — swatch + picker + alpha (hex)
 * - `format` — encode-format pills
 * - `targetSize` — KB/MB target-file-size field (bytes on the wire)
 * - `filterGrid` — visual filter-thumbnail picker
 */
export type ControlKind =
  | "number"
  | "text"
  | "toggle"
  | "segmented"
  | "slider"
  | "position"
  | "color"
  | "format"
  | "targetSize"
  | "filterGrid";

type FieldLabelKey = (typeof strings.workspace.field)[keyof typeof strings.workspace.field];

export interface OpField {
  /** Proto field name (snake_case); also the params object key. */
  name: string;
  /** The humanized control that renders this field. */
  control: ControlKind;
  /** Translation key for the field label. */
  labelKey: FieldLabelKey;
  /** Default value; drives the controlled input's initial state. */
  default: string | number | boolean;
  /** Allowed values for closed-set controls (technical API enum tokens). */
  options?: readonly string[];
  /** Slider bounds (also clamps numeric entry). */
  min?: number;
  max?: number;
  step?: number;
  /** Unit symbol shown after a slider value (e.g. "px", "°", "%"). */
  unit?: string;
  /** When true, render under the Inspector's collapsible "Advanced" group. */
  advanced?: boolean;
}

export interface OpSpec {
  fields: readonly OpField[];
  /** True when the op consumes a second `overlay` image part. */
  acceptsOverlay?: boolean;
}

/** Encode formats offered by the format pills (decode-only HEIC/SVG excluded). */
export const ENCODE_FORMATS = ["png", "jpeg", "webp", "avif", "gif", "tiff", "bmp"] as const;

export const OP_SPECS: Readonly<Record<string, OpSpec>> = {
  resize: {
    fields: [
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 256, min: 0, unit: "px" },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 0, min: 0, unit: "px" },
      {
        name: "fit",
        control: "segmented",
        labelKey: strings.workspace.field.fit,
        default: "fit",
        options: ["fit", "fill", "stretch"],
      },
      { name: "gravity", control: "position", labelKey: strings.workspace.field.gravity, default: "" },
    ],
  },
  crop: {
    fields: [
      { name: "x", control: "number", labelKey: strings.workspace.field.x, default: 0, min: 0, unit: "px", advanced: true },
      { name: "y", control: "number", labelKey: strings.workspace.field.y, default: 0, min: 0, unit: "px", advanced: true },
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 100, min: 1, unit: "px", advanced: true },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 100, min: 1, unit: "px", advanced: true },
    ],
  },
  rotate: {
    fields: [
      { name: "angle", control: "slider", labelKey: strings.workspace.field.angle, default: 90, min: -180, max: 180, step: 1, unit: "°" },
      { name: "background", control: "color", labelKey: strings.workspace.field.background, default: "" },
    ],
  },
  flip: {
    fields: [
      {
        name: "axis",
        control: "segmented",
        labelKey: strings.workspace.field.axis,
        default: "horizontal",
        options: ["horizontal", "vertical"],
      },
    ],
  },
  deskew: {
    fields: [{ name: "background", control: "color", labelKey: strings.workspace.field.background, default: "" }],
  },
  thumbnail: {
    fields: [
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 128, min: 1, unit: "px" },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 128, min: 1, unit: "px" },
    ],
  },
  canvas: {
    fields: [
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 512, min: 1, unit: "px" },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 512, min: 1, unit: "px" },
      { name: "background", control: "color", labelKey: strings.workspace.field.background, default: "" },
      { name: "gravity", control: "position", labelKey: strings.workspace.field.gravity, default: "" },
    ],
  },
  adjust: {
    fields: [
      { name: "brightness", control: "slider", labelKey: strings.workspace.field.brightness, default: 0, min: -100, max: 100, step: 1 },
      { name: "contrast", control: "slider", labelKey: strings.workspace.field.contrast, default: 0, min: -100, max: 100, step: 1 },
      { name: "gamma", control: "slider", labelKey: strings.workspace.field.gamma, default: 0, min: 0, max: 3, step: 0.05 },
      { name: "saturation", control: "slider", labelKey: strings.workspace.field.saturation, default: 0, min: -100, max: 100, step: 1 },
      { name: "hue", control: "slider", labelKey: strings.workspace.field.hue, default: 0, min: -180, max: 180, step: 1, unit: "°" },
    ],
  },
  filter: {
    fields: [
      {
        name: "filter",
        control: "filterGrid",
        labelKey: strings.workspace.field.filter,
        default: "grayscale",
        options: ["grayscale", "sepia", "invert", "blur", "sharpen"],
      },
      { name: "amount", control: "slider", labelKey: strings.workspace.field.amount, default: 1, min: 0, max: 10, step: 0.5 },
    ],
  },
  convert: {
    fields: [
      { name: "format", control: "format", labelKey: strings.workspace.field.format, default: "png", options: ENCODE_FORMATS },
      { name: "quality", control: "slider", labelKey: strings.workspace.field.quality, default: 90, min: 1, max: 100, step: 1 },
      { name: "lossless", control: "toggle", labelKey: strings.workspace.field.lossless, default: false },
    ],
  },
  compress: {
    fields: [
      { name: "format", control: "format", labelKey: strings.workspace.field.format, default: "jpeg", options: ENCODE_FORMATS },
      { name: "quality", control: "slider", labelKey: strings.workspace.field.quality, default: 80, min: 1, max: 100, step: 1 },
      { name: "lossless", control: "toggle", labelKey: strings.workspace.field.lossless, default: false },
      { name: "target_bytes", control: "targetSize", labelKey: strings.workspace.field.targetBytes, default: 0 },
    ],
  },
  overlay: {
    acceptsOverlay: true,
    fields: [
      { name: "text", control: "text", labelKey: strings.workspace.field.text, default: "" },
      { name: "position", control: "position", labelKey: strings.workspace.field.position, default: "" },
      { name: "opacity", control: "slider", labelKey: strings.workspace.field.opacity, default: 1, min: 0, max: 1, step: 0.05 },
      { name: "color", control: "color", labelKey: strings.workspace.field.color, default: "" },
      { name: "font_size", control: "slider", labelKey: strings.workspace.field.fontSize, default: 0, min: 0, max: 128, step: 1, unit: "px" },
    ],
  },
  metadata: {
    fields: [
      { name: "strip_all", control: "toggle", labelKey: strings.workspace.field.stripAll, default: false },
      { name: "strip_gps", control: "toggle", labelKey: strings.workspace.field.stripGps, default: false },
      { name: "auto_orient", control: "toggle", labelKey: strings.workspace.field.autoOrient, default: false },
    ],
  },
};

/** Spec for an operation name, or `undefined` if the op is unknown. */
export const opSpec = (operation: string): OpSpec | undefined => OP_SPECS[operation];
