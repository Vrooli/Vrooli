import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, MeshStandardMaterial, Object3D, SphereGeometry } from 'three'
import { ambientPolicy } from '../../config/ambient'
import type { QualityProfile, QualityProfileId } from '../../config'
import type { WorldClock } from '../../config/clock'
import type { AnimationLeases } from '../../engine/animationLeases'
import { ambientMeasurements } from '../../engine/diagnostics/ambient'
import { squirrelPose, squirrelRouteSteps, nextSquirrelBoundary, type SquirrelRoute } from '../../sim/ambient/squirrels'
import { hashString } from '../../sim/rng'
import { runCooperatively } from '../../sim/cooperative'
import { useWorldStore } from '../WorldStoreContext'
import { bindAmbientWake } from './wake'

/** Original russet squirrel with a curved plume tail, pale belly and small ears.
 * Nearby tree routes occupy a fixed instance pool and never enter agent nav.
 */
export function Squirrels({ clock, leases, enabled, reducedMotion, profileId, profile }: {
  clock: WorldClock; leases: AnimationLeases; enabled: boolean; reducedMotion: boolean; profileId: QualityProfileId; profile: QualityProfile;
}) {
  const state = useWorldStore().getState()
  const invalidate = useThree(state => state.invalidate)
  const [selection, setSelection] = useState<{ nav: typeof state.nav; decor: typeof state.decor; habitat: Uint8Array; routes: SquirrelRoute[] } | null>(null)
  const release = useRef<(() => void) | null>(null)
  const routes = useMemo(() => enabled && !reducedMotion && selection?.nav === state.nav && selection.decor === state.decor && selection.habitat === state.habitats
    ? selection.routes.filter(route => hashString(`density:${route.treeId}`) / 0xffffffff <= profile.vegetationDensityScale) : [],
  [enabled, reducedMotion, selection, state.nav, state.decor, state.habitats, profile.vegetationDensityScale])
  const resources = useMemo(() => {
    const geometry = new SphereGeometry(1, 8, 6), material = new MeshStandardMaterial({ roughness: 1, flatShading: true })
    const parts = [
      { p: [0,.22,0], s: [.14,.18,.24], color: '#ad6034' },
      { p: [0,.2,.12], s: [.1,.13,.14], color: '#ead5b7' },
      { p: [0,.36,.2], s: [.12,.12,.13], color: '#c37640', head: true },
      { p: [0,.31,.32], s: [.06,.055,.065], color: '#e0b18a', head: true },
      { p: [0,.32,.375], s: [.025,.023,.025], color: '#382a23', head: true },
      { p: [-.075,.48,.18], s: [.042,.075,.035], color: '#a35934', head: true },
      { p: [.075,.48,.18], s: [.042,.075,.035], color: '#a35934', head: true },
      { p: [-.101,.38,.27], s: [.025,.03,.025], color: '#231d19', head: true },
      { p: [.101,.38,.27], s: [.025,.03,.025], color: '#231d19', head: true },
      { p: [-.11,.055,-.1], s: [.065,.055,.105], color: '#854526', foot: true },
      { p: [.11,.055,-.1], s: [.065,.055,.105], color: '#854526', foot: true },
      { p: [-.08,.06,.17], s: [.045,.05,.085], color: '#854526', foot: true },
      { p: [.08,.06,.17], s: [.045,.05,.085], color: '#854526', foot: true },
      { p: [0,.19,-.23], s: [.11,.1,.13], color: '#98512f' },
      { p: [0,.26,-.35], s: [.15,.13,.15], color: '#a95e35' },
      { p: [0,.43,-.42], s: [.16,.18,.15], color: '#b16c3e' },
      { p: [0,.63,-.4], s: [.145,.17,.13], color: '#bb7948' },
      { p: [0,.77,-.31], s: [.11,.12,.13], color: '#c48b57' },
      { p: [0,.79,-.19], s: [.07,.08,.1], color: '#d39c65' },
    ]
    const mesh = new InstancedMesh(geometry, material, parts.length * ambientPolicy.squirrels.maximumVisible.ultra)
    mesh.name = 'ambient-squirrels'; mesh.count = 0; mesh.frustumCulled = false; mesh.raycast = () => undefined
    for (let slot = 0; slot < ambientPolicy.squirrels.maximumVisible.ultra; slot++) for (const [index, part] of parts.entries()) mesh.setColorAt(slot * parts.length + index, new Color(part.color))
    return { mesh, geometry, material, parts, root: new Object3D(), local: new Object3D(), nearest: [] as SquirrelRoute[] }
  }, [])
  useEffect(() => {
    const controller = new AbortController()
    void runCooperatively(squirrelRouteSteps(state.seed, state.terrain, state.habitats, state.nav, state.decor), { signal: controller.signal })
      .then(routes => { if (!controller.signal.aborted) setSelection({ nav: state.nav, decor: state.decor, habitat: state.habitats, routes }) })
      .catch((error: unknown) => { if (!controller.signal.aborted) console.error('Squirrel route selection failed', error) })
    return () => controller.abort()
  }, [state.seed, state.terrain, state.habitats, state.nav, state.decor])
  useEffect(() => { invalidate() }, [routes, invalidate])
  useEffect(() => bindAmbientWake(now => routes.length ? Math.min(...routes.map(route => nextSquirrelBoundary(route, now))) : null, invalidate, clock), [routes, clock, invalidate])
  useEffect(() => () => { release.current?.(); release.current = null }, [leases])
  useEffect(() => () => { resources.mesh.dispose(); resources.geometry.dispose(); resources.material.dispose(); ambientMeasurements.clear('squirrels') }, [resources])
  useFrame(({ camera }) => {
    const start = performance.now(), snapshot = clock.snapshot(), limit = ambientPolicy.squirrels.maximumVisible[profileId]
    const distance = (route: SquirrelRoute) => (route.tree[0] - camera.position.x) ** 2 + (route.tree[1] - camera.position.z) ** 2
    resources.nearest.length = 0
    for (const route of routes) if (distance(route) < 2025) resources.nearest.push(route)
    resources.nearest.sort((a, b) => distance(a) - distance(b) || a.rank - b.rank)
    resources.nearest.length = Math.min(limit, resources.nearest.length)
    let moving = false
    resources.mesh.count = resources.nearest.length * resources.parts.length
    resources.mesh.visible = resources.mesh.count > 0
    for (const [slot, route] of resources.nearest.entries()) {
      const pose = squirrelPose(route, state.terrain, snapshot.utcMilliseconds / 1000)
      moving ||= pose.animating && snapshot.timeScale > 0
      resources.root.position.set(...pose.position); resources.root.rotation.set(pose.pitch, pose.yaw, 0, 'YXZ'); resources.root.updateMatrix()
      for (const [index, part] of resources.parts.entries()) {
        resources.local.position.set(part.p[0] ?? 0, (part.p[1] ?? 0) + (part.head ? pose.nibble : part.foot ? pose.stride * .035 * (index % 2 ? -1 : 1) : 0), part.p[2] ?? 0)
        resources.local.scale.set(part.s[0] ?? 1, part.s[1] ?? 1, part.s[2] ?? 1)
        resources.local.updateMatrix(); resources.local.matrix.premultiply(resources.root.matrix)
        resources.mesh.setMatrixAt(slot * resources.parts.length + index, resources.local.matrix)
      }
    }
    resources.mesh.instanceMatrix.needsUpdate = true
    if (moving) release.current ??= leases.acquire()
    else { release.current?.(); release.current = null }
    ambientMeasurements.record('squirrels', performance.now() - start, resources.nearest.length, ambientPolicy.squirrels.maximumVisible.ultra, limit)
  })
  return <primitive object={resources.mesh} dispose={null} />
}
