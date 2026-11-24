import type { CompositionDefinition, Scene, SlotManifest } from "./engine";
import { compositionDefinition as conversionFunnel } from "./conversionFunnel";
import { compositionDefinition as flowCurrent } from "./flowCurrent";
import { compositionDefinition as hiveLattice } from "./hiveLattice";
import { compositionDefinition as ledgerRiver } from "./ledgerRiver";
import { compositionDefinition as orbitalField } from "./orbitalField";
import { compositionDefinition as panoramaConstellation } from "./panoramaConstellation";
import { compositionDefinition as signalConstellation } from "./signalConstellation";
import { registerEnabledPacks } from "../packs/types";
import { vrooliPack } from "../packs/vrooli";

/** Each scene module owns its id, factory, and slot contract as one definition. */
const engineCompositions: CompositionDefinition[] = [
  orbitalField,
  hiveLattice,
  flowCurrent,
  ledgerRiver,
  signalConstellation,
  panoramaConstellation,
  conversionFunnel,
];

// Keep manifests available for parity checks even while a connector pack is
// disabled; runtime scene factories still register only enabled-pack members.
const allCompositionDefinitions = [...engineCompositions, ...vrooliPack.compositions];
const compositionManifests: Record<string, SlotManifest> = {};
for (const definition of allCompositionDefinitions) {
  if (Object.prototype.hasOwnProperty.call(compositionManifests, definition.id)) throw new Error(`duplicate composition id "${definition.id}"`);
  compositionManifests[definition.id] = definition.slots;
}
export const compositionSlotManifests = compositionManifests;

export let compositions: Record<string, () => Scene> = {};
const defaultScene = orbitalField.create;

export function configureEnabledPacks(enabled: Set<string>): void {
  const registered = registerEnabledPacks({ readouts: {}, compositions: engineCompositions }, [vrooliPack], enabled);
  compositions = Object.fromEntries(registered.compositions.map(({ id, create }) => [id, create]));
}

// The shipped operator instance enables the Vrooli pack; a no-pack instance
// can call configureEnabledPacks(new Set()) before mounting the board.
configureEnabledPacks(new Set(["vrooli"]));

export const createScene = (composition: string): Scene => (compositions[composition] ?? defaultScene)();
