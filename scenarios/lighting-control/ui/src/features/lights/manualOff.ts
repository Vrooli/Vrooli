import type { ManualOverride } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";

// These predicates describe the persisted request, not observed light output.
// Only explicit all-enrolled scope makes an all-enrolled Off action redundant.
export function manualOffCoversAll(override?: ManualOverride): boolean {
  return override?.active === true && override.allEnrolled;
}

export function manualOffCoversController(override: ManualOverride | undefined, controllerId: string): boolean {
  return Boolean(controllerId) && (manualOffCoversAll(override) ||
    (override?.active === true && override.controllerIds.includes(controllerId)));
}
