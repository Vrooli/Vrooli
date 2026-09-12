import { strings } from "../../consts/strings";
import type { Hardware, HostSummary } from "../../api/models";

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
    | typeof strings.models.hardware.gpuAvailable
    | typeof strings.models.hardware.gpuShort
    | typeof strings.models.hardware.cpuOk
    | typeof strings.models.hardware.vram
    | typeof strings.models.hardware.ram
    | typeof strings.models.hardware.speedNote;
  values?: Record<string, string | number>;
  tone: HardwareFitTone;
}

/**
 * leadChip answers "can I run this?" for a model's lead hardware chip. When a
 * host snapshot is supplied it is HOST-AWARE: a GPU-required model on a machine
 * whose GPU has enough TOTAL VRAM reads as a positive "Runs on your GPU" (the
 * fix for the old chip that showed a red "Needs a GPU" warning even on a GPU
 * host). This is a catalog/capability view, so it judges by the card's total
 * capacity (stable) — the per-operation picker separately judges momentary free
 * VRAM for immediate runnability. If the card is simply too small it says how
 * much more is needed; with no host (or no GPU) it falls back to the static
 * requirement. CPU-capable models read as a positive "Runs on your CPU".
 */
function leadChip(hardware: Hardware, host: HostSummary | undefined): HardwareFitChip {
  if (!hardware.gpuRequired) {
    return { key: strings.models.hardware.cpuOk, tone: "positive" };
  }
  if (host?.hasGpu) {
    if (!host.vramKnown || host.vramTotalGb >= hardware.minVramGb) {
      return { key: strings.models.hardware.gpuAvailable, tone: "positive" };
    }
    const shortfall = Math.max(1, hardware.minVramGb - host.vramTotalGb);
    return { key: strings.models.hardware.gpuShort, values: { gb: shortfall }, tone: "caution" };
  }
  return { key: strings.models.hardware.gpuRequired, tone: "caution" };
}

/**
 * Derive the operator-facing hardware-fit chips for a model. The lead chip
 * answers "can I run this?" — CPU-capable reads as a positive "runs on your
 * CPU", GPU-required reads as a caution so the operator checks their hardware.
 * VRAM/RAM minimums surface only when the seed declares a non-zero floor, and
 * the seed's free-text speed note (if any) trails as a neutral hint. Pure +
 * deterministic so it is unit-testable in isolation.
 */
export function hardwareFitChips(
  hardware: Hardware | undefined,
  host?: HostSummary,
): HardwareFitChip[] {
  const chips: HardwareFitChip[] = [];
  if (!hardware) {
    return chips;
  }
  chips.push(leadChip(hardware, host));
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
