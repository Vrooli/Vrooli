// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/capability/manifests.seed.json (presentation.fields)
//         cross-checked against packages/proto/schemas/image-tools/v1/ops/ops.proto
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/capability/render_opspecs_test.go fails the build if this file is stale.
//
// One parameter form per deterministic operation that declares one. The
// Inspector renders the field's control; the field name is the OpParams field
// the value is sent as. Adding a knob is a manifest + proto edit, not a UI map.

import { ENCODE_FORMATS, type OpSpec } from "./opSpecTypes";
import { strings } from "../../consts/strings";

export const OP_SPECS: Readonly<Record<string, OpSpec>> = {
  "adjust": {
    fields: [
      { name: "brightness", control: "slider", labelKey: strings.workspace.field.brightness, default: 0, min: -100, max: 100, step: 1 },
      { name: "contrast", control: "slider", labelKey: strings.workspace.field.contrast, default: 0, min: -100, max: 100, step: 1 },
      { name: "gamma", control: "slider", labelKey: strings.workspace.field.gamma, default: 0, min: 0, max: 3, step: 0.05 },
      { name: "saturation", control: "slider", labelKey: strings.workspace.field.saturation, default: 0, min: -100, max: 100, step: 1 },
      { name: "hue", control: "slider", labelKey: strings.workspace.field.hue, default: 0, min: -180, max: 180, step: 1, unit: "°" },
    ],
  },
  "canvas": {
    fields: [
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 512, min: 1, unit: "px" },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 512, min: 1, unit: "px" },
      { name: "background", control: "color", labelKey: strings.workspace.field.background, default: "" },
      { name: "gravity", control: "position", labelKey: strings.workspace.field.gravity, default: "" },
    ],
  },
  "compress": {
    fields: [
      { name: "format", control: "format", labelKey: strings.workspace.field.format, default: "jpeg", options: ENCODE_FORMATS },
      { name: "quality", control: "slider", labelKey: strings.workspace.field.quality, default: 80, min: 1, max: 100, step: 1 },
      { name: "lossless", control: "toggle", labelKey: strings.workspace.field.lossless, default: false },
      { name: "target_bytes", control: "targetSize", labelKey: strings.workspace.field.targetBytes, default: 0 },
    ],
  },
  "convert": {
    fields: [
      { name: "format", control: "format", labelKey: strings.workspace.field.format, default: "png", options: ENCODE_FORMATS },
      { name: "quality", control: "slider", labelKey: strings.workspace.field.quality, default: 90, min: 1, max: 100, step: 1 },
      { name: "lossless", control: "toggle", labelKey: strings.workspace.field.lossless, default: false },
    ],
  },
  "crop": {
    fields: [
      { name: "x", control: "number", labelKey: strings.workspace.field.x, default: 0, min: 0, unit: "px", advanced: true },
      { name: "y", control: "number", labelKey: strings.workspace.field.y, default: 0, min: 0, unit: "px", advanced: true },
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 100, min: 1, unit: "px", advanced: true },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 100, min: 1, unit: "px", advanced: true },
    ],
  },
  "deskew": {
    fields: [
      { name: "background", control: "color", labelKey: strings.workspace.field.background, default: "" },
    ],
  },
  "filter": {
    fields: [
      { name: "filter", control: "filterGrid", labelKey: strings.workspace.field.filter, default: "grayscale", options: ["grayscale", "sepia", "invert", "blur", "sharpen"] },
      { name: "amount", control: "slider", labelKey: strings.workspace.field.amount, default: 1, min: 0, max: 10, step: 0.5 },
    ],
  },
  "flip": {
    fields: [
      { name: "axis", control: "segmented", labelKey: strings.workspace.field.axis, default: "horizontal", options: ["horizontal", "vertical"] },
    ],
  },
  "metadata": {
    fields: [
      { name: "strip_all", control: "toggle", labelKey: strings.workspace.field.stripAll, default: false },
      { name: "strip_gps", control: "toggle", labelKey: strings.workspace.field.stripGps, default: false },
      { name: "auto_orient", control: "toggle", labelKey: strings.workspace.field.autoOrient, default: false },
    ],
  },
  "overlay": {
    acceptsOverlay: true,
    fields: [
      { name: "text", control: "text", labelKey: strings.workspace.field.text, default: "" },
      { name: "position", control: "position", labelKey: strings.workspace.field.position, default: "" },
      { name: "opacity", control: "slider", labelKey: strings.workspace.field.opacity, default: 1, min: 0, max: 1, step: 0.05 },
      { name: "color", control: "color", labelKey: strings.workspace.field.color, default: "" },
      { name: "font_size", control: "slider", labelKey: strings.workspace.field.fontSize, default: 0, min: 0, max: 128, step: 1, unit: "px" },
    ],
  },
  "resize": {
    fields: [
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 256, min: 0, unit: "px" },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 0, min: 0, unit: "px" },
      { name: "fit", control: "segmented", labelKey: strings.workspace.field.fit, default: "fit", options: ["fit", "fill", "stretch"] },
      { name: "gravity", control: "position", labelKey: strings.workspace.field.gravity, default: "" },
    ],
  },
  "rotate": {
    fields: [
      { name: "angle", control: "slider", labelKey: strings.workspace.field.angle, default: 90, min: -180, max: 180, step: 1, unit: "°" },
      { name: "background", control: "color", labelKey: strings.workspace.field.background, default: "" },
    ],
  },
  "screenshot_cleanup": {
    fields: [
      { name: "padding_px", control: "number", labelKey: strings.workspace.field.padding, default: 0, min: 0, unit: "px" },
      { name: "tolerance", control: "slider", labelKey: strings.workspace.field.tolerance, default: 0.04, min: 0, max: 0.3, step: 0.01, advanced: true },
      { name: "max_trim_percent", control: "slider", labelKey: strings.workspace.field.maxTrimPercent, default: 0.25, min: 0, max: 0.45, step: 0.05, advanced: true },
    ],
  },
  "thumbnail": {
    fields: [
      { name: "width", control: "number", labelKey: strings.workspace.field.width, default: 128, min: 1, unit: "px" },
      { name: "height", control: "number", labelKey: strings.workspace.field.height, default: 128, min: 1, unit: "px" },
    ],
  },
  "print_prep": {
    fields: [
      { name: "width_in", control: "number", labelKey: strings.workspace.field.widthIn, default: 6.0, min: 0.01, max: 30, step: 0.01, unit: "in" },
      { name: "height_in", control: "number", labelKey: strings.workspace.field.heightIn, default: 9.0, min: 0.01, max: 30, step: 0.01, unit: "in" },
      { name: "dpi", control: "number", labelKey: strings.workspace.field.dpi, default: 300, min: 72, max: 1200, step: 1 },
      { name: "ink_mode", control: "segmented", labelKey: strings.workspace.field.inkMode, default: "1bit", options: ["1bit", "colour"] },
      { name: "threshold", control: "slider", labelKey: strings.workspace.field.threshold, default: 0.5, min: 0, max: 1, step: 0.01 },
      { name: "gravity", control: "position", labelKey: strings.workspace.field.gravity, default: "center" },
    ],
  },
};
