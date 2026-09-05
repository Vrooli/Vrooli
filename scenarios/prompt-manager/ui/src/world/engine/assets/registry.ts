/**
 * The baked prop registry: one entry per scene prop with path, bounds and
 * triangle count, produced by `pnpm world:assets` from assets-src/world.
 */
import generated from './registry.generated.json'
import { WORLD_ASSETS, worldAssetUrl } from './urls'

export interface PropRecord {
  scene: string
  id: string
  path: string
  source: string
  kit: string
  bounds: { min: [number, number, number]; max: [number, number, number] }
  size: [number, number, number]
  triangles: number
  materials: number
  bytes: number
}

const props: Record<string, PropRecord> = Object.fromEntries(
  Object.entries(generated.props).map(([key, value]) => [
    key,
    {
      ...value,
      bounds: { min: value.bounds.min as [number, number, number], max: value.bounds.max as [number, number, number] },
      size: value.size as [number, number, number],
    },
  ]),
)

export function propRecord(assetSet: string, id: string): PropRecord | undefined {
  return props[`${assetSet}/${id}`]
}

export function propUrl(record: PropRecord): string {
  return worldAssetUrl(record.path)
}

export function allProps(): PropRecord[] {
  return Object.values(props)
}

export { WORLD_ASSETS }
