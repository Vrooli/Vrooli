import { biomeSets, scenes, type SceneId } from '../config'

export interface PresentationAsset { assetSet: string; id: string; role: 'core' | 'dressing' }

/** All assets currently referenced by the destination, independent of transient
 * camera culling or quality density. Room fillers and exterior dressing use the
 * same asset-set rule as the vegetation renderer.
 */
export function presentationManifest(world: { scene: SceneId; decor: readonly { propId?: string; roomId?: string }[] }): PresentationAsset[] {
  const scene = scenes[world.scene]
  const assets = new Map<string, PresentationAsset>()
  for (const id of [scene.props.desk, scene.props.chair, scene.props.table, scene.props.seat, scene.props.hearth, scene.props.lamp, scene.props.board]) {
    assets.set(`${scene.assetSet}/${id}`, { assetSet: scene.assetSet, id, role: 'core' })
  }
  for (const spot of world.decor) {
    if (!spot.propId) continue
    const assetSet = (spot.roomId ? scene : biomeSets[scene.biomeSet]).assetSet
    const key = `${assetSet}/${spot.propId}`
    if (!assets.has(key)) assets.set(key, { assetSet, id: spot.propId, role: 'dressing' })
  }
  return [...assets.values()].sort((a, b) => `${a.assetSet}/${a.id}`.localeCompare(`${b.assetSet}/${b.id}`))
}
