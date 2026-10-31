import { ambientMeasurements } from '../../engine/diagnostics/ambient'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { BufferAttribute, BufferGeometry, DoubleSide, InstancedBufferAttribute, InstancedMesh, Object3D, ShaderMaterial } from 'three'
import { ambientPolicy } from '../../config/ambient'
import type { WorldClock } from '../../config/clock'
import type { AnimationLeases } from '../../engine/animationLeases'
import { butterflyPose, butterflySiteSteps, type ButterflySite } from '../../sim/ambient/butterflies'
import { birdPose, birdRouteSteps, type BirdRoute } from '../../sim/ambient/birds'
import { runCooperatively } from '../../sim/cooperative'
import { useWorldStore } from '../WorldStoreContext'

interface FlightProps {
  clock: WorldClock; leases: AnimationLeases; enabled: boolean; reducedMotion: boolean;
  profileId: keyof typeof ambientPolicy.butterflies.maximumVisible;
}
export function Butterflies(props: FlightProps) { return <FlyingWildlife {...props} kind="butterflies" /> }
export function Birds(props: FlightProps) { return <FlyingWildlife {...props} kind="birds" /> }

/** Original procedural silhouettes and motion; no third-party assets. */
function FlyingWildlife({ clock, leases, enabled, reducedMotion, profileId, kind }: FlightProps & { kind: 'butterflies' | 'birds' }) {
  const state = useWorldStore().getState()
  const invalidate = useThree(state => state.invalidate)
  useEffect(() => () => ambientMeasurements.clear(kind), [kind])
  const [selection, setSelection] = useState<{ habitat: Uint8Array; decor: typeof state.decor; sites: Array<ButterflySite | BirdRoute> } | null>(null)
  const release = useRef<(() => void) | null>(null)
  const resources = useMemo(() => {
    const capacity = ambientPolicy[kind].maximumVisible.ultra
    const bird = kind === 'birds'
    const geometry = new BufferGeometry()
    const vertices: number[] = []
    for (const side of [-1, 1]) {
      // Two lobes share a narrow root; forward is +z.
      const wings = bird ? [[0,0, .38,.04, .62,-.23], [0,0, .62,-.23, .22,-.15]] : [[0,0, .18,.13, .17,-.01], [0,0, .17,-.01, .12,-.12]]
      for (const triangle of wings) {
        for (let v = 0; v < triangle.length; v += 2) vertices.push((triangle[v] ?? 0) * side, 0, triangle[v + 1] ?? 0)
      }
    }
    const body = new Array<number>(vertices.length / 3).fill(0)
    const bodyTriangles = bird ? [
      [-.035,-.18, .035,-.18, .035,.22], [-.035,-.18, .035,.22, -.035,.22],
      [-.035,.22, .035,.22, 0,.34], [-.10,-.28, .10,-.28, 0,-.10],
    ] : [
      [-.012,-.105, .012,-.105, .012,.135], [-.012,-.105, .012,.135, -.012,.135],
      [0,.125, .045,.20, .048,.20], [0,.125, .048,.20, .003,.125],
      [0,.125, -.045,.20, -.048,.20], [0,.125, -.048,.20, -.003,.125],
    ]
    for (const triangle of bodyTriangles) {
      for (let v = 0; v < triangle.length; v += 2) { vertices.push(triangle[v] ?? 0, .006, triangle[v + 1] ?? 0); body.push(1) }
    }
    geometry.setAttribute('position', new BufferAttribute(Float32Array.from(vertices), 3))
    geometry.setAttribute('body', new BufferAttribute(Float32Array.from(body), 1))
    const phases = new InstancedBufferAttribute(new Float32Array(capacity), 1)
    geometry.setAttribute('phase', phases)
    const material = new ShaderMaterial({ side: DoubleSide,
      uniforms: { seconds: { value: 0 }, bird: { value: Number(bird) } },
      vertexShader: `attribute float phase,body; uniform float seconds,bird; varying vec2 wing; varying float variant,vBody;
        void main(){wing=vec2(abs(position.x)/mix(.18,.62,bird),(position.z+.12)/.25);variant=phase;vBody=body;
          float flap=.25+.85*(.5+.5*sin(seconds*25.132741+phase));
          if(bird>.5){float glide=smoothstep(-.2,.2,sin(seconds*.785398+phase));flap=mix(.05,sin(seconds*12.566371+phase)*.55,glide);}
          vec3 p=position;if(body<.5){p.x=position.x*cos(flap);p.y=abs(position.x)*sin(flap);}
          gl_Position=projectionMatrix*modelViewMatrix*instanceMatrix*vec4(p,1.);}`,
      fragmentShader: `varying vec2 wing; varying float variant,vBody; uniform float bird;
        void main(){vec3 color=mix(vec3(1.,.55,.08),vec3(.45,.65,1.),step(3.14,variant));
          if(bird>.5)color=vec3(.45,.48,.52);
          float edge=smoothstep(.76,.96,wing.x);float vein=pow(abs(sin(wing.y*18.+wing.x*4.)),24.)*.35;
          gl_FragColor=vec4(mix(color,vec3(.08,.055,.04),max(vBody,max(edge,vein))),1.);}`,
    })
    const mesh = new InstancedMesh(geometry, material, capacity)
    mesh.name = `ambient-${kind}`; mesh.count = 0; mesh.frustumCulled = false; mesh.raycast = () => undefined
    return { geometry, phases, material, mesh, dummy: new Object3D() }
  }, [kind])
  useEffect(() => {
    const controller = new AbortController()
    const steps = kind === 'birds' ? birdRouteSteps(state.seed, state.terrain, state.habitats) : butterflySiteSteps(state.seed, state.terrain, state.habitats, state.decor)
    void runCooperatively<unknown, Array<ButterflySite | BirdRoute>>(steps, { signal: controller.signal })
      .then(sites => { if (!controller.signal.aborted) setSelection({ habitat: state.habitats, decor: state.decor, sites }) })
      .catch((error: unknown) => { if (!controller.signal.aborted) console.error(`${kind} habitat selection failed`, error) })
    return () => controller.abort()
  }, [kind, state.seed, state.terrain, state.habitats, state.decor])
  useEffect(() => { invalidate() }, [selection, enabled, reducedMotion, profileId, invalidate])
  useEffect(() => clock.subscribe(invalidate), [clock, invalidate])
  useEffect(() => () => { release.current?.(); release.current = null }, [leases])
  useEffect(() => () => { resources.mesh.dispose(); resources.geometry.dispose(); resources.material.dispose() }, [resources])
  useFrame(() => {
    const measurementStart = performance.now()
    const sites = selection?.habitat === state.habitats && selection.decor === state.decor ? selection.sites : []
    const count = enabled && !reducedMotion ? Math.min(sites.length, ambientPolicy[kind].maximumVisible[profileId]) : 0
    const snapshot = clock.snapshot()
    const seconds = snapshot.utcMilliseconds / 1000
    const moving = count > 0 && snapshot.timeScale > 0
    if (moving && !release.current) release.current = leases.acquire()
    if (!moving && release.current) { release.current(); release.current = null }
    resources.mesh.count = count; resources.mesh.visible = count > 0
    const time = resources.material.uniforms.seconds
    if (time) time.value = seconds % 120
    for (let index = 0; index < count; index++) {
      const site = sites[index]
      if (!site) continue
      const pose: { position: readonly [number, number, number]; yaw: number; bank?: number } = 'vegetationId' in site ? butterflyPose(site, state.terrain, seconds) : birdPose(site, seconds)
      resources.dummy.position.set(...pose.position)
      resources.dummy.rotation.set(0, pose.yaw, pose.bank ?? 0)
      resources.dummy.updateMatrix()
      resources.mesh.setMatrixAt(index, resources.dummy.matrix)
      resources.phases.setX(index, site.phase)
    }
    resources.mesh.instanceMatrix.needsUpdate = true
    resources.phases.needsUpdate = true
    ambientMeasurements.record(kind, performance.now() - measurementStart, count, ambientPolicy[kind].maximumVisible.ultra, ambientPolicy[kind].maximumVisible[profileId])
  })
  return <primitive object={resources.mesh} dispose={null} />
}
