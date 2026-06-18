import { strings } from "../../consts/strings";

/**
 * Declarative spec for each deterministic op's parameter form. The
 * discovery RPC (`ListOperations`) tells us *which* ops exist; this table
 * tells the form *which fields* each op takes and how to render them. The
 * field `name` is the proto field name (snake_case) sent inside the
 * operation-keyed params object — see `runOp` in `api/ops.ts`.
 *
 * Keep this in sync with the proto `OpParams` oneof. Adding a field is a
 * one-row edit here plus a label key in `i18n/locales/*`.
 */
export type FieldKind = "number" | "text" | "checkbox" | "select";

export interface OpField {
  /** Proto field name (snake_case); also the params object key. */
  name: string;
  kind: FieldKind;
  /** Translation key for the field label. */
  labelKey: (typeof strings.workspace.field)[keyof typeof strings.workspace.field];
  /** Default value; drives the controlled input's initial state. */
  default: string | number | boolean;
  /** Allowed values for `select` fields (technical API enum tokens). */
  options?: readonly string[];
}

export interface OpSpec {
  fields: readonly OpField[];
  /** True when the op consumes a second `overlay` image part. */
  acceptsOverlay?: boolean;
}

export const OP_SPECS: Readonly<Record<string, OpSpec>> = {
  resize: {
    fields: [
      { name: "width", kind: "number", labelKey: strings.workspace.field.width, default: 256 },
      { name: "height", kind: "number", labelKey: strings.workspace.field.height, default: 0 },
      {
        name: "fit",
        kind: "select",
        labelKey: strings.workspace.field.fit,
        default: "fit",
        options: ["fit", "fill", "stretch"],
      },
      { name: "gravity", kind: "text", labelKey: strings.workspace.field.gravity, default: "" },
    ],
  },
  crop: {
    fields: [
      { name: "x", kind: "number", labelKey: strings.workspace.field.x, default: 0 },
      { name: "y", kind: "number", labelKey: strings.workspace.field.y, default: 0 },
      { name: "width", kind: "number", labelKey: strings.workspace.field.width, default: 100 },
      { name: "height", kind: "number", labelKey: strings.workspace.field.height, default: 100 },
      { name: "gravity", kind: "text", labelKey: strings.workspace.field.gravity, default: "" },
    ],
  },
  rotate: {
    fields: [
      { name: "angle", kind: "number", labelKey: strings.workspace.field.angle, default: 90 },
      { name: "background", kind: "text", labelKey: strings.workspace.field.background, default: "" },
    ],
  },
  flip: {
    fields: [
      {
        name: "axis",
        kind: "select",
        labelKey: strings.workspace.field.axis,
        default: "horizontal",
        options: ["horizontal", "vertical"],
      },
    ],
  },
  deskew: {
    fields: [{ name: "background", kind: "text", labelKey: strings.workspace.field.background, default: "" }],
  },
  thumbnail: {
    fields: [
      { name: "width", kind: "number", labelKey: strings.workspace.field.width, default: 128 },
      { name: "height", kind: "number", labelKey: strings.workspace.field.height, default: 128 },
    ],
  },
  canvas: {
    fields: [
      { name: "width", kind: "number", labelKey: strings.workspace.field.width, default: 512 },
      { name: "height", kind: "number", labelKey: strings.workspace.field.height, default: 512 },
      { name: "background", kind: "text", labelKey: strings.workspace.field.background, default: "" },
      { name: "gravity", kind: "text", labelKey: strings.workspace.field.gravity, default: "" },
    ],
  },
  adjust: {
    fields: [
      { name: "brightness", kind: "number", labelKey: strings.workspace.field.brightness, default: 0 },
      { name: "contrast", kind: "number", labelKey: strings.workspace.field.contrast, default: 0 },
      { name: "gamma", kind: "number", labelKey: strings.workspace.field.gamma, default: 0 },
      { name: "saturation", kind: "number", labelKey: strings.workspace.field.saturation, default: 0 },
      { name: "hue", kind: "number", labelKey: strings.workspace.field.hue, default: 0 },
    ],
  },
  filter: {
    fields: [
      {
        name: "filter",
        kind: "select",
        labelKey: strings.workspace.field.filter,
        default: "grayscale",
        options: ["grayscale", "sepia", "invert", "blur", "sharpen"],
      },
      { name: "amount", kind: "number", labelKey: strings.workspace.field.amount, default: 1 },
    ],
  },
  convert: {
    fields: [
      { name: "format", kind: "text", labelKey: strings.workspace.field.format, default: "png" },
      { name: "quality", kind: "number", labelKey: strings.workspace.field.quality, default: 90 },
      { name: "lossless", kind: "checkbox", labelKey: strings.workspace.field.lossless, default: false },
    ],
  },
  compress: {
    fields: [
      { name: "format", kind: "text", labelKey: strings.workspace.field.format, default: "jpeg" },
      { name: "quality", kind: "number", labelKey: strings.workspace.field.quality, default: 80 },
      { name: "lossless", kind: "checkbox", labelKey: strings.workspace.field.lossless, default: false },
      { name: "target_bytes", kind: "number", labelKey: strings.workspace.field.targetBytes, default: 0 },
    ],
  },
  overlay: {
    acceptsOverlay: true,
    fields: [
      { name: "text", kind: "text", labelKey: strings.workspace.field.text, default: "" },
      { name: "position", kind: "text", labelKey: strings.workspace.field.position, default: "" },
      { name: "opacity", kind: "number", labelKey: strings.workspace.field.opacity, default: 1 },
      { name: "color", kind: "text", labelKey: strings.workspace.field.color, default: "" },
      { name: "font_size", kind: "number", labelKey: strings.workspace.field.fontSize, default: 0 },
    ],
  },
  metadata: {
    fields: [
      { name: "strip_all", kind: "checkbox", labelKey: strings.workspace.field.stripAll, default: false },
      { name: "strip_gps", kind: "checkbox", labelKey: strings.workspace.field.stripGps, default: false },
      { name: "auto_orient", kind: "checkbox", labelKey: strings.workspace.field.autoOrient, default: false },
    ],
  },
};

/** Spec for an operation name, or `undefined` if the op is unknown. */
export const opSpec = (operation: string): OpSpec | undefined => OP_SPECS[operation];
