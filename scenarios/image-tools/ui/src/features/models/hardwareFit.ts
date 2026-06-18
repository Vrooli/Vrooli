import { strings } from "../../consts/strings";
import type { Hardware } from "../../api/models";

/**
 * Visual tone for a hardware-fit chip. Maps to a paired token treatment
 * (background + text + an icon/word) so the meaning never rides on color
 * alone: `positive` = clear-to-run, `caution` = a real requirement to check,
 * `neutral` = an informational floor.
 */
export type HardwareFitTone = "positive" | "caution" | "neutral";

/**
 * A single hardware-fit indicator: a translation key plus the interpolation
 * values it needs, and the tone that drives its token treatment. Kept
 * declarative (key, not translated copy) so the card resolves them through
 * `t()` and tests assert on stable key paths.
 */
export interface HardwareFitChip {
  key:
    | typeof strings.models.hardware.gpuRequired
    | typeof strings.models.hardware.cpuOk
    | typeof strings.models.hardware.vram
    | typeof strings.models.hardware.ram
    | typeof strings.models.hardware.speedNote;
  values?: Record<string, string | number>;
  tone: HardwareFitTone;
}

/**
 * Derive the operator-facing hardware-fit chips for a model. The lead chip
 * answers "can I run this?" — CPU-capable reads as a positive "runs on your
 * CPU", GPU-required reads as a caution so the operator checks their hardware.
 * VRAM/RAM minimums surface only when the seed declares a non-zero floor, and
 * the seed's free-text speed note (if any) trails as a neutral hint. Pure +
 * deterministic so it is unit-testable in isolation.
 */
export function hardwareFitChips(hardware: Hardware | undefined): HardwareFitChip[] {
  const chips: HardwareFitChip[] = [];
  if (!hardware) {
    return chips;
  }
  chips.push(
    hardware.gpuRequired
      ? { key: strings.models.hardware.gpuRequired, tone: "caution" }
      : { key: strings.models.hardware.cpuOk, tone: "positive" },
  );
  if (hardware.minVramGb > 0) {
    chips.push({
      key: strings.models.hardware.vram,
      values: { gb: hardware.minVramGb },
      tone: "neutral",
    });
  }
  if (hardware.minRamGb > 0) {
    chips.push({
      key: strings.models.hardware.ram,
      values: { gb: hardware.minRamGb },
      tone: "neutral",
    });
  }
  if (hardware.speedNote.trim().length > 0) {
    chips.push({
      key: strings.models.hardware.speedNote,
      values: { note: hardware.speedNote },
      tone: "neutral",
    });
  }
  return chips;
}
