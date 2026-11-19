import { strings } from "../../consts/strings";

/**
 * Shared types and constants for the deterministic operation forms.
 *
 * The forms themselves are generated into `opSpecs.generated.ts` from the
 * capability manifests, cross-checked against the proto `OpParams` descriptor.
 * This module is the hand-written half: the vocabulary of controls the Inspector
 * can render, and the one shared option list those controls reference.
 */

/**
 * Which humanized primitive renders a field.
 * - `number` / `text` — plain inputs (last-resort fallback)
 * - `toggle` — on/off switch
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

/** Encode formats offered by the format pills (decode-only HEIC/SVG excluded). */
export const ENCODE_FORMATS = ["png", "jpeg", "webp", "avif", "gif", "tiff", "bmp"] as const;

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
