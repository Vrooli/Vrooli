import { useEffect, useMemo, useState } from 'react'
import { BufferAttribute, BufferGeometry, PointsMaterial } from 'three'
import { heightAt, type NavGrid } from '../sim'
import type { TerrainField } from '../sim/terrain'
import { navigationOverlaySteps } from './navigationOverlay'
import { runCooperatively } from '../sim/cooperative'
import type { BiomeSet } from '../config'
import { biomeOverlaySteps, habitatOverlaySteps } from './classificationOverlay'

/** Opt-in debug markers, visible through structures and excluded from picking. */
export function GridOverlay({ kind, nav, terrain, biomes, habitats, biomeSet, onStatus }: {
  kind: 'navigation' | 'biomes' | 'habitats'; nav: NavGrid; terrain: TerrainField; biomes: Uint8Array; habitats: Uint8Array; biomeSet: BiomeSet; onStatus: (status: string) => void
}) {
  const [prepared, setPrepared] = useState<{ kind: string; nav: NavGrid; terrain: TerrainField; biomes: Uint8Array; habitats: Uint8Array; geometry: BufferGeometry } | null>(null)
  useEffect(() => {
    const controller = new AbortController()
    let geometry: BufferGeometry | null = null
    onStatus('Preparing grid markers…')
    const steps = kind === 'navigation' ? navigationOverlaySteps(nav, (x, z) => heightAt(terrain, x, z))
      : kind === 'habitats' ? habitatOverlaySteps(terrain, habitats) : biomeOverlaySteps(terrain, biomes, biomeSet)
    void runCooperatively<{ completed: number; total: number }, { positions: Float32Array; colors: Float32Array; bytes: number; summary: string }>(steps, { signal: controller.signal }).then(data => {
      if (controller.signal.aborted) return
      geometry = new BufferGeometry()
      geometry.setAttribute('position', new BufferAttribute(data.positions, 3))
      geometry.setAttribute('color', new BufferAttribute(data.colors, 3))
      setPrepared({ kind, nav, terrain, biomes, habitats, geometry })
      onStatus(`${data.summary} · ${(data.bytes / 1048576).toFixed(2)} MiB buffers`)
    }).catch(() => { if (!controller.signal.aborted) onStatus('Could not prepare grid markers.') })
    return () => { controller.abort(); geometry?.dispose() }
  }, [kind, nav, terrain, biomes, habitats, biomeSet, onStatus])
  const material = useMemo(() => new PointsMaterial({ size: 3, sizeAttenuation: false, vertexColors: true,
    depthTest: false, depthWrite: false, toneMapped: false, transparent: true, opacity: 0.8 }), [])
  useEffect(() => () => material.dispose(), [material])
  if (prepared?.kind !== kind || prepared.nav !== nav || prepared.terrain !== terrain || prepared.biomes !== biomes || prepared.habitats !== habitats) return null
  return <points name={`${kind}-overlay`} geometry={prepared.geometry} material={material} dispose={null} renderOrder={1000} raycast={() => {}} />
}
