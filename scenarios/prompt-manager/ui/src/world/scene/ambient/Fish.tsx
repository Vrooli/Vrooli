import { ambientMeasurements } from '../../engine/diagnostics/ambient'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { DoubleSide, Group, Mesh, MeshBasicMaterial, MeshStandardMaterial, RingGeometry, SphereGeometry } from 'three'
import { ambientPolicy } from '../../config/ambient'
import type { WorldClock } from '../../config/clock'
import type { AnimationLeases } from '../../engine/animationLeases'
import { fishEventsAt, fishPondSteps, fishPose, nextFishBoundary, type FishEvent, type FishPond } from '../../sim/ambient/fish'
import { runCooperatively } from '../../sim/cooperative'
import { useWorldStore } from '../WorldStoreContext'
import { PresentationPool } from './pool'
import { bindAmbientWake } from './wake'
import { worldTextureProps } from '../materialTextures'

export function Fish({ clock, leases, enabled, reducedMotion, profileId }: {
  clock: WorldClock; leases: AnimationLeases; enabled: boolean; reducedMotion: boolean;
  profileId: keyof typeof ambientPolicy.fish.maximumVisible;
}) {
  const state = useWorldStore().getState()
  const invalidate = useThree(state => state.invalidate)
  useEffect(() => () => ambientMeasurements.clear('fish'), [])
  const [selection, setSelection] = useState<{ water: typeof state.waterGeometry; habitat: Uint8Array; ponds: FishPond[] } | null>(null)
  const ponds = useMemo(() => enabled && !reducedMotion && selection?.water === state.waterGeometry && selection.habitat === state.habitats ? selection.ponds : [], [enabled, reducedMotion, selection, state.waterGeometry, state.habitats])
  const release = useRef<(() => void) | null>(null)
  const pool = useMemo(() => new PresentationPool<FishEvent>(ambientPolicy.fish.maximumConcurrent), [])
  const resources = useMemo(() => {
    const bodyGeometry = new SphereGeometry(1, 8, 6)
    const ringGeometry = new RingGeometry(.88, 1, 32)
    const fishMaterial = new MeshStandardMaterial({ color: '#bda479', roughness: .65, metalness: .15, ...worldTextureProps('leaf') })
    const eyeMaterial = new MeshBasicMaterial({ color: '#181b1b' })
    const splashMaterial = new MeshBasicMaterial({ color: '#c5eced', transparent: true, opacity: .7, depthWrite: false })
    const slots = Array.from({ length: ambientPolicy.fish.maximumConcurrent }, (_, index) => {
      const root = new Group(); root.visible = false; root.name = `fish-event-slot-${index}`
      const fish = new Mesh(bodyGeometry, fishMaterial); fish.scale.set(.055, .07, .16)
      fish.name = 'fish-body'
      const tail = new Mesh(bodyGeometry, fishMaterial); fish.add(tail)
      // Child dimensions inherit the body scale, so compensate to retain a visible tail.
      tail.scale.set(.015 / .055, .075 / .07, .07 / .16); tail.position.z = -.17 / .16
      for (const side of [-1, 1]) {
        const eye = new Mesh(bodyGeometry, eyeMaterial)
        eye.position.set(side * .044 / .055, .02 / .07, .105 / .16)
        eye.scale.set(.012 / .055, .012 / .07, .012 / .16); fish.add(eye)
      }
      const ring = new Mesh(ringGeometry, new MeshBasicMaterial({ color: '#d8efea', transparent: true, opacity: 0, depthWrite: false, side: DoubleSide }))
      ring.name = 'fish-ripple'
      ring.renderOrder = 3 // Water uses order 2 and does not write depth.
      ring.rotation.x = -Math.PI / 2
      const drops = Array.from({ length: 8 }, () => { const drop = new Mesh(bodyGeometry, splashMaterial); drop.scale.setScalar(.025); drop.renderOrder = 3; root.add(drop); return drop })
      root.add(fish, ring); root.traverse(object => { object.raycast = () => undefined })
      return { root, fish, ring, drops }
    })
    return { bodyGeometry, ringGeometry, fishMaterial, eyeMaterial, splashMaterial, slots }
  }, [])
  useEffect(() => {
    const controller = new AbortController()
    void runCooperatively(fishPondSteps(state.waterGeometry, state.terrain, state.habitats), { signal: controller.signal })
      .then(ponds => { if (!controller.signal.aborted) setSelection({ water: state.waterGeometry, habitat: state.habitats, ponds }) })
      .catch((error: unknown) => { if (!controller.signal.aborted) console.error('Fish pond selection failed', error) })
    return () => controller.abort()
  }, [state.waterGeometry, state.terrain, state.habitats])
  useEffect(() => { invalidate() }, [ponds, profileId, invalidate])
  useEffect(() => bindAmbientWake(now => nextFishBoundary(state.seed, now, ponds), invalidate, clock), [state.seed, ponds, clock, invalidate])
  useEffect(() => () => { release.current?.(); release.current = null; pool.clear() }, [leases, pool])
  useEffect(() => () => {
    resources.bodyGeometry.dispose(); resources.ringGeometry.dispose(); resources.fishMaterial.dispose(); resources.eyeMaterial.dispose(); resources.splashMaterial.dispose()
    resources.slots.forEach(slot => slot.ring.material.dispose())
  }, [resources])
  useFrame(() => {
    const measurementStart = performance.now()
    const snapshot = clock.snapshot(), now = snapshot.utcMilliseconds / 1000
    pool.sync(fishEventsAt(state.seed, now, ponds), now, ambientPolicy.fish.maximumVisible[profileId])
    const moving = pool.stats().active > 0 && snapshot.timeScale > 0
    if (moving && !release.current) release.current = leases.acquire()
    if (!moving && release.current) { release.current(); release.current = null }
    for (const [index, slot] of pool.slots.entries()) {
      const meshes = resources.slots[index]
      if (!meshes) continue
      const event = slot.event; meshes.root.visible = event !== null
      if (!event) continue
      const pose = fishPose(event, now), size = Math.min(1, event.site.radius / .3)
      meshes.root.position.set(...event.site.position)
      meshes.fish.visible = pose.jumping
      meshes.fish.position.y = pose.height
      meshes.fish.scale.set(.055 * size, .07 * size, .16 * size)
      meshes.fish.rotation.set(pose.pitch, event.seed / 4294967296 * Math.PI * 2, 0)
      meshes.ring.visible = !pose.jumping
      meshes.ring.position.y = .008
      meshes.ring.scale.setScalar(pose.radius)
      meshes.ring.material.opacity = pose.opacity * .65
      const splash = (now - event.start - ambientPolicy.fish.jumpSeconds) / .5
      for (const [dropIndex, drop] of meshes.drops.entries()) {
        drop.visible = splash >= 0 && splash < 1
        const angle = dropIndex / 8 * Math.PI * 2
        const radius = event.site.radius * .6 * splash
        drop.position.set(Math.cos(angle) * radius, Math.max(0, Math.sin(splash * Math.PI)) * .25, Math.sin(angle) * radius)
        drop.scale.setScalar(Math.min(.025, event.site.radius * .1))
      }
    }
    ambientMeasurements.record('fish', performance.now() - measurementStart, pool.stats().active, ambientPolicy.fish.maximumConcurrent, ambientPolicy.fish.maximumVisible[profileId])
  })
  return <group name="ambient-fish">{resources.slots.map((slot, index) => <primitive key={index} object={slot.root} dispose={null} />)}</group>
}
