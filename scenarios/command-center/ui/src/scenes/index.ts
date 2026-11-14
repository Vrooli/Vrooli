import type { Scene, SlotManifest } from "./engine";
import { flowCurrent, slotManifest as flowCurrentSlots } from "./flowCurrent";
import { hiveLattice, slotManifest as hiveLatticeSlots } from "./hiveLattice";
import { ledgerRiver, slotManifest as ledgerRiverSlots } from "./ledgerRiver";
import { orbitalField, slotManifest as orbitalFieldSlots } from "./orbitalField";
import { panoramaConstellation, slotManifest as panoramaConstellationSlots } from "./panoramaConstellation";
import { signalConstellation, slotManifest as signalConstellationSlots } from "./signalConstellation";
import { conversionFunnel, slotManifest as conversionFunnelSlots } from "./conversionFunnel";
import { slotManifest as meridianArcSlots } from "./meridianArc";
import { slotManifest as funnelCascadeSlots } from "./funnelCascade";
import { vrooliPack } from "../packs/vrooli";
import { registerEnabledPacks } from "../packs/types";

/** Generic engine compositions. Connector-specific compositions are registered below. */
const engineCompositions: Record<string, () => Scene> = {
  "orbital-field": orbitalField,
  "hive-lattice": hiveLattice,
  "flow-current": flowCurrent,
  "ledger-river": ledgerRiver,
  "signal-constellation": signalConstellation,
  "panorama-constellation": panoramaConstellation,
  "conversion-funnel": conversionFunnel,
};

export const compositionSlotManifests: Record<string, SlotManifest> = {
  "orbital-field": orbitalFieldSlots,
  "hive-lattice": hiveLatticeSlots,
  "flow-current": flowCurrentSlots,
  "ledger-river": ledgerRiverSlots,
  "signal-constellation": signalConstellationSlots,
  "panorama-constellation": panoramaConstellationSlots,
  "conversion-funnel": conversionFunnelSlots,
  "meridian-arc": meridianArcSlots,
  "funnel-cascade": funnelCascadeSlots,
};

export const compositions: Record<string, () => Scene> = {};

export function configureEnabledPacks(enabled: Set<string>): void {
  const registered = registerEnabledPacks({ readouts: {}, compositions: engineCompositions }, [vrooliPack], enabled);
  for (const key of Object.keys(compositions)) delete compositions[key];
  Object.assign(compositions, registered.compositions);
}

// The shipped operator instance enables the Vrooli pack; a no-pack instance
// can call configureEnabledPacks(new Set()) before mounting the board.
configureEnabledPacks(new Set(["vrooli"]));

export const createScene = (composition: string): Scene => (compositions[composition] ?? orbitalField)();
