import type { Scene } from "../scenes/engine";

export interface ConnectorPack {
  id: string;
  sourceDescriptors: string[];
  readouts: Record<string, string>;
  compositions: Record<string, () => Scene>;
}

export interface CatalogRegistries {
  readouts: Record<string, string>;
  compositions: Record<string, () => Scene>;
}

export function registerEnabledPacks(engine: CatalogRegistries, packs: ConnectorPack[], enabled: Set<string>): CatalogRegistries {
  const readouts = { ...engine.readouts };
  const compositions = { ...engine.compositions };
  for (const pack of packs) {
    if (!enabled.has(pack.id)) continue;
    Object.assign(readouts, pack.readouts);
    Object.assign(compositions, pack.compositions);
  }
  return { readouts, compositions };
}
