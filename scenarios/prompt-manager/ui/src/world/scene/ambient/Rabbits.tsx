import { ambientMeasurements } from '../../engine/diagnostics/ambient'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, MeshStandardMaterial, Object3D, SphereGeometry } from 'three'
import { ambientPolicy } from '../../config/ambient'
import type { WorldClock } from '../../config/clock'
import type { AnimationLeases } from '../../engine/animationLeases'
import { nextRabbitBoundary, rabbitPose, rabbitRouteSteps, type RabbitRoute } from '../../sim/ambient/rabbits'
import { runCooperatively } from '../../sim/cooperative'
import { useWorldStore } from '../WorldStoreContext'
import { bindAmbientWake } from './wake'

/** Original low-poly rabbit, assembled from one pooled primitive geometry. */
export function Rabbits({ clock, leases, enabled, reducedMotion, profileId }: {
  clock: WorldClock; leases: AnimationLeases; enabled: boolean; reducedMotion: boolean;
  profileId: keyof typeof ambientPolicy.rabbits.maximumVisible;
}) {
  const state = useWorldStore().getState()
  const invalidate = useThree(state => state.invalidate)
  useEffect(() => () => ambientMeasurements.clear('rabbits'), [])
  const [selection, setSelection] = useState<{ nav: typeof state.nav; habitat: Uint8Array; routes: RabbitRoute[] } | null>(null)
  const release = useRef<(() => void) | null>(null)
  const routes = useMemo(() => enabled && !reducedMotion && selection?.nav === state.nav && selection.habitat === state.habitats
    ? selection.routes.slice(0, ambientPolicy.rabbits.maximumVisible[profileId]) : [], [enabled, reducedMotion, selection, state.nav, state.habitats, profileId])
  const resources = useMemo(() => {
    const geometry = new SphereGeometry(1, 8, 6)
    const material = new MeshStandardMaterial({ roughness: 1, flatShading: true })
    const parts = [
      { p: [0,.23,0], s: [.18,.21,.27], color: '#9b8068' },
      { p: [0,.38,.21], s: [.13,.14,.13], color: '#ae9174' },
      { p: [-.065,.59,.19], s: [.04,.19,.045], color: '#a58a70' },
      { p: [.065,.59,.19], s: [.04,.19,.045], color: '#a58a70' },
      { p: [-.065,.6,.229], s: [.022,.13,.012], color: '#c59891' },
      { p: [.065,.6,.229], s: [.022,.13,.012], color: '#c59891' },
      { p: [-.12,.06,-.1], s: [.085,.06,.13], color: '#876f5b' },
      { p: [.12,.06,-.1], s: [.085,.06,.13], color: '#876f5b' },
      { p: [-.09,.045,.19], s: [.05,.045,.1], color: '#876f5b' },
      { p: [.09,.045,.19], s: [.05,.045,.1], color: '#876f5b' },
      { p: [0,.25,-.27], s: [.085,.085,.085], color: '#d9cfbf' },
      { p: [-.105,.42,.277], s: [.022,.025,.022], color: '#241e1a' },
      { p: [.105,.42,.277], s: [.022,.025,.022], color: '#241e1a' },
    ]
    const mesh = new InstancedMesh(geometry, material, parts.length * ambientPolicy.rabbits.maximumVisible.ultra)
    mesh.name = 'ambient-rabbits'; mesh.count = 0; mesh.frustumCulled = false; mesh.raycast = () => undefined
    for (let i = 0; i < ambientPolicy.rabbits.maximumVisible.ultra; i++) for (const [part, value] of parts.entries()) mesh.setColorAt(i * parts.length + part, new Color(value.color))
    return { geometry, material, mesh, parts, root: new Object3D(), local: new Object3D() }
  }, [])
  useEffect(() => {
    const controller = new AbortController()
    void runCooperatively(rabbitRouteSteps(state.seed, state.terrain, state.habitats, state.nav), { signal: controller.signal })
      .then(routes => { if (!controller.signal.aborted) setSelection({ nav: state.nav, habitat: state.habitats, routes }) })
      .catch((error: unknown) => { if (!controller.signal.aborted) console.error('Rabbit route selection failed', error) })
    return () => controller.abort()
  }, [state.seed, state.terrain, state.habitats, state.nav])
  useEffect(() => { invalidate() }, [routes, invalidate])
  useEffect(() => bindAmbientWake(now => routes.length ? Math.min(...routes.map(route => nextRabbitBoundary(route, now))) : null, invalidate, clock), [routes, clock, invalidate])
  useEffect(() => () => { release.current?.(); release.current = null }, [leases])
  useEffect(() => () => { resources.mesh.dispose(); resources.geometry.dispose(); resources.material.dispose() }, [resources])
  useFrame(() => {
    const measurementStart = performance.now()
    const snapshot = clock.snapshot()
    let moving = false
    resources.mesh.count = routes.length * resources.parts.length
    resources.mesh.visible = routes.length > 0
    for (const [index, route] of routes.entries()) {
      const pose = rabbitPose(route, state.terrain, snapshot.utcMilliseconds / 1000)
      moving ||= pose.moving && snapshot.timeScale > 0
      resources.root.position.set(pose.position[0], pose.position[1] + pose.hop, pose.position[2])
      resources.root.rotation.set(0, pose.yaw, 0); resources.root.updateMatrix()
      for (const [part, value] of resources.parts.entries()) {
        resources.local.position.set(value.p[0] ?? 0, value.p[1] ?? 0, value.p[2] ?? 0)
        resources.local.scale.set(value.s[0] ?? 1, value.s[1] ?? 1, value.s[2] ?? 1)
        resources.local.updateMatrix(); resources.local.matrix.premultiply(resources.root.matrix)
        resources.mesh.setMatrixAt(index * resources.parts.length + part, resources.local.matrix)
      }
    }
    resources.mesh.instanceMatrix.needsUpdate = true
    if (moving && !release.current) release.current = leases.acquire()
    if (!moving && release.current) { release.current(); release.current = null }
    ambientMeasurements.record('rabbits', performance.now() - measurementStart, routes.length, ambientPolicy.rabbits.maximumVisible.ultra, ambientPolicy.rabbits.maximumVisible[profileId])
  })
  return <primitive object={resources.mesh} dispose={null} />
}
