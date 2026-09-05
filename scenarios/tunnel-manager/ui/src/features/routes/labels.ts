import { PublicExposure } from "@vrooli/proto-types/tunnel-manager/v1/routes/routes_pb";

import { strings } from "../../consts/strings";
import type { BadgeTone } from "../../components/ui/StatusBadge";

type ExposureKey = (typeof strings.routes.publicExposure.option)[keyof typeof strings.routes.publicExposure.option];

// Static enum→key map. The `strings/no-unused-keys` eslint rule is a literal
// static scan, so every leaf needs a literal `strings.routes.publicExposure.*`
// reference — this record provides exactly that, and keeps the helpers typed to
// the strings-subtree union (never `string`).
const EXPOSURE_LABEL: Record<PublicExposure, ExposureKey> = {
  [PublicExposure.UNSPECIFIED]: strings.routes.publicExposure.option.inherit,
  [PublicExposure.INHERIT]: strings.routes.publicExposure.option.inherit,
  [PublicExposure.ENABLED]: strings.routes.publicExposure.option.enabled,
  [PublicExposure.DISABLED]: strings.routes.publicExposure.option.disabled,
};

const EXPOSURE_TONE: Record<PublicExposure, BadgeTone> = {
  [PublicExposure.UNSPECIFIED]: "neutral",
  [PublicExposure.INHERIT]: "neutral",
  [PublicExposure.ENABLED]: "success",
  [PublicExposure.DISABLED]: "warning",
};

/** Map a per-route public-exposure override to its translation key. */
export function publicExposureLabel(value: PublicExposure): ExposureKey {
  return EXPOSURE_LABEL[value];
}

/** Map a per-route public-exposure override to a badge tone. */
export function publicExposureTone(value: PublicExposure): BadgeTone {
  return EXPOSURE_TONE[value];
}

/**
 * Normalize a route's stored override to a concrete selectable value. Legacy /
 * unset rows persist as UNSPECIFIED but mean INHERIT, so the select never shows
 * an empty option.
 */
export function normalizeExposure(value: PublicExposure): PublicExposure {
  return value === PublicExposure.UNSPECIFIED ? PublicExposure.INHERIT : value;
}

/** The three operator-selectable override values, in display order. */
export const PUBLIC_EXPOSURE_OPTIONS: readonly PublicExposure[] = [
  PublicExposure.INHERIT,
  PublicExposure.ENABLED,
  PublicExposure.DISABLED,
];
