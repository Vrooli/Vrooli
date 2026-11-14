import { ambientMeasurements } from '../../engine/diagnostics/ambient'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { AdditiveBlending, BufferAttribute, BufferGeometry, Points, ShaderMaterial } from 'three'
import { ambientPolicy } from '../../config/ambient'
import type { WorldClock } from '../../config/clock'
import type { AnimationLeases } from '../../engine/animationLeases'
import { fireflyPose, fireflySiteSteps, type FireflySite } from '../../sim/ambient/fireflies'
import { runCooperatively } from '../../sim/cooperative'
import { useWorldStore } from '../WorldStoreContext'

export function Fireflies({ clock, leases, enabled, reducedMotion, profileId }: {
  clock: WorldClock; leases: AnimationLeases; enabled: boolean; reducedMotion: boolean;
  profileId: keyof typeof ambientPolicy.fireflies.maximumVisible;
}) {
  const state = useWorldStore().getState()
  const invalidate = useThree(state => state.invalidate)
  useEffect(() => () => ambientMeasurements.clear('fireflies'), [])
  const [selection, setSelection] = useState<{ habitat: Uint8Array; sites: FireflySite[] } | null>(null)
  const release = useRef<(() => void) | null>(null)
  const resources = useMemo(() => {
    const capacity = ambientPolicy.fireflies.maximumVisible.ultra
    const geometry = new BufferGeometry()
    const positions = new BufferAttribute(new Float32Array(capacity * 3), 3)
    const glows = new BufferAttribute(new Float32Array(capacity), 1)
    geometry.setAttribute('position', positions)
    geometry.setAttribute('glow', glows)
    geometry.setDrawRange(0, 0)
    const material = new ShaderMaterial({ transparent: true, depthWrite: false, blending: AdditiveBlending, toneMapped: false,
      uniforms: { pixelScale: { value: 1 } },
      vertexShader: `attribute float glow; varying float vGlow; uniform float pixelScale;
        void main(){vGlow=glow;vec4 p=modelViewMatrix*vec4(position,1.);gl_Position=projectionMatrix*p;
          gl_PointSize=clamp(pixelScale*.16/max(1.,-p.z),1.,18.);}`,
      fragmentShader: `varying float vGlow; void main(){vec2 p=gl_PointCoord*2.-1.;float r=dot(p,p);
        if(r>1.) discard;float halo=exp(-4.*r);float core=exp(-30.*r);
        gl_FragColor=vec4(mix(vec3(.55,1.,.08),vec3(1.,1.,.7),core),halo*vGlow);}`,
    })
    const points = new Points(geometry, material)
    points.name = 'ambient-fireflies'; points.frustumCulled = false; points.raycast = () => undefined
    return { geometry, material, positions, glows, points }
  }, [])
  useEffect(() => {
    const controller = new AbortController()
    void runCooperatively(fireflySiteSteps(state.seed, state.terrain, state.habitats, state.habitatRegions), { signal: controller.signal })
      .then(sites => { if (!controller.signal.aborted) setSelection({ habitat: state.habitats, sites }) })
      .catch((error: unknown) => { if (!controller.signal.aborted) console.error('Firefly habitat selection failed', error) })
    return () => controller.abort()
  }, [state.seed, state.terrain, state.habitats, state.habitatRegions])
  useEffect(() => { invalidate() }, [selection, enabled, reducedMotion, profileId, invalidate])
  useEffect(() => clock.subscribe(invalidate), [clock, invalidate])
  useEffect(() => () => { release.current?.(); release.current = null }, [leases])
  useEffect(() => () => { resources.geometry.dispose(); resources.material.dispose() }, [resources])
  useFrame(({ gl, camera, size }) => {
    const measurementStart = performance.now()
    const sites = selection?.habitat === state.habitats ? selection.sites : []
    const count = enabled ? Math.min(sites.length, ambientPolicy.fireflies.maximumVisible[profileId]) : 0
    const snapshot = clock.snapshot()
    const moving = count > 0 && !reducedMotion && snapshot.timeScale > 0
    if (moving && !release.current) release.current = leases.acquire()
    if (!moving && release.current) { release.current(); release.current = null }
    resources.geometry.setDrawRange(0, count)
    resources.points.visible = count > 0
    const pixelScale = resources.material.uniforms.pixelScale
    if (pixelScale) pixelScale.value = size.height * gl.getPixelRatio() * camera.projectionMatrix.elements[5]
    for (let index = 0; index < count; index++) {
      const site = sites[index]
      if (!site) continue
      const pose = fireflyPose(site, reducedMotion ? 0 : snapshot.utcMilliseconds / 1000)
      resources.positions.setXYZ(index, ...pose.position)
      resources.glows.setX(index, reducedMotion ? .6 : pose.glow)
    }
    resources.positions.needsUpdate = true
    resources.glows.needsUpdate = true
    ambientMeasurements.record('fireflies', performance.now() - measurementStart, count, ambientPolicy.fireflies.maximumVisible.ultra, ambientPolicy.fireflies.maximumVisible[profileId])
  })
  return <primitive object={resources.points} dispose={null} />
}
