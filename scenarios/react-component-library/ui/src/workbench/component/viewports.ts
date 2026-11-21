/**
 * One viewport table for every preview surface.
 *
 * The frame (physical chrome) and the viewport (what is being laid out) are
 * separate choices. Keeping the widths here means "phone" is 390 × 844 in the
 * studio, the in-context view and the share popout, instead of each surface
 * inventing its own height.
 */

export type DeviceKind = "desktop" | "tablet" | "phone";

export interface ViewportPreset {
  id: string;
  width: number;
  height: number;
  label: string;
}

export const DEVICE_VIEWPORT: Record<DeviceKind, ViewportPreset> = {
  phone: { id: "390", width: 390, height: 844, label: "390" },
  tablet: { id: "834", width: 834, height: 1112, label: "834" },
  desktop: { id: "1180", width: 1180, height: 760, label: "1180" },
};

/** The reference widths the overflow check always measures. */
export const REFERENCE_WIDTHS = [390, 834, 1180] as const;

export const VIEWPORTS: ViewportPreset[] = [
  { id: "320", width: 320, height: 640, label: "320" },
  DEVICE_VIEWPORT.phone,
  DEVICE_VIEWPORT.tablet,
  DEVICE_VIEWPORT.desktop,
];

/** Safe-area insets the frame applies inside the viewport. */
export const SAFE_TOP: Record<DeviceKind, number> = { desktop: 0, tablet: 24, phone: 54 };
export const SAFE_BOTTOM: Record<DeviceKind, number> = { desktop: 0, tablet: 22, phone: 22 };

export function presetFor(device: DeviceKind): ViewportPreset {
  return DEVICE_VIEWPORT[device];
}
