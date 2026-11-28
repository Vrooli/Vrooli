import type { CompositionDefinition } from "../scenes/engine";

export interface ConnectorPack {
  id: string;
  sourceDescriptors: string[];
  readouts: Record<string, string>;
  compositions: CompositionDefinition[];
}

export interface CatalogRegistries {
  readouts: Record<string, string>;
  compositions: CompositionDefinition[];
}

export function registerEnabledPacks(engine: CatalogRegistries, packs: ConnectorPack[], enabled: Set<string>): CatalogRegistries {
  const readouts = { ...engine.readouts };
  const compositions = [...engine.compositions];
  const compositionIDs = new Set(compositions.map((composition) => composition.id));
  for (const pack of packs) {
    if (!enabled.has(pack.id)) continue;
    Object.assign(readouts, pack.readouts);
    for (const composition of pack.compositions) {
      if (compositionIDs.has(composition.id)) throw new Error(`duplicate composition id "${composition.id}"`);
      compositionIDs.add(composition.id);
      compositions.push(composition);
    }
  }
  return { readouts, compositions };
}
