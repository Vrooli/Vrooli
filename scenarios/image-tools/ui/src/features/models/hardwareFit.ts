import { strings } from "../../consts/strings";
import type { Hardware } from "../../api/models";

/**
 * A single hardware-fit indicator: a translation key plus the interpolation
 * values it needs. Kept declarative (key, not translated copy) so the card
 * resolves them through `t()` and tests assert on stable key paths.
 */
export interface HardwareFitChip {
  key:
    | typeof strings.models.hardware.gpuRequired
    | typeof strings.models.hardware.cpuOk
    | typeof strings.models.hardware.vram
    | typeof strings.models.hardware.ram;
  values?: Record<string, number>;
}

/**
 * Derive the operator-facing hardware-fit chips for a model. GPU-required vs
 * CPU-capable is always shown; VRAM/RAM minimums only when the seed declares a
 * non-zero floor. Pure + deterministic so it is unit-testable in isolation.
 */
export function hardwareFitChips(hardware: Hardware | undefined): HardwareFitChip[] {
  const chips: HardwareFitChip[] = [];
  if (!hardware) {
    return chips;
  }
  chips.push({
    key: hardware.gpuRequired
      ? strings.models.hardware.gpuRequired
      : strings.models.hardware.cpuOk,
  });
  if (hardware.minVramGb > 0) {
    chips.push({ key: strings.models.hardware.vram, values: { gb: hardware.minVramGb } });
  }
  if (hardware.minRamGb > 0) {
    chips.push({ key: strings.models.hardware.ram, values: { gb: hardware.minRamGb } });
  }
  return chips;
}
