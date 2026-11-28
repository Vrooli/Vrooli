import { describe, expect, it } from 'vitest'
import { biomeSets, scenes } from '../config'
import { presentationManifest } from './presentationManifest'

describe('destination presentation manifest', () => {
  it.each(['park', 'office'] as const)('deduplicates %s assets and resolves interior/exterior sources', sceneId => {
    const scene = scenes[sceneId]
    const decor = [{ propId: 'exterior-fixture' }, { propId: 'exterior-fixture' }, { propId: 'interior-fixture', roomId: 'room' }, { propId: scene.props.desk, roomId: 'room' }]
    const manifest = presentationManifest({ scene: sceneId, decor })
    expect(manifest).toContainEqual({ assetSet: scene.assetSet, id: scene.props.desk, role: 'core' })
    expect(manifest).toContainEqual({ assetSet: scene.assetSet, id: 'interior-fixture', role: 'dressing' })
    expect(manifest.filter(asset => asset.id === 'exterior-fixture')).toEqual([{ assetSet: biomeSets[scene.biomeSet].assetSet, id: 'exterior-fixture', role: 'dressing' }])
    expect(manifest).toEqual(presentationManifest({ scene: sceneId, decor: [...decor].reverse() }))
  })
})
