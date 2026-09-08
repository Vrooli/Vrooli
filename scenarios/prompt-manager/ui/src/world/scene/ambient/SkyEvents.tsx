import { ambientMeasurements } from '../../engine/diagnostics/ambient'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { AdditiveBlending, Color, Mesh, PlaneGeometry, ShaderMaterial, Vector3 } from 'three'
import type { AnimationLeases } from '../../engine/animationLeases'
import { nextSkyBoundary, type SkyEligibility, type SkyEvent } from '../../sim/ambient/schedule'
import { Rng } from '../../sim/rng'
import { SkyPresentation } from './skyPresentation'
import { bindAmbientWake } from './wake'
import type { WorldClock } from '../../config/clock'

const colors = { white: '#fff3dd', blue: '#79bcff', green: '#85ffb4', violet: '#ca95ff', 'great-fireball': '#ffb96b' }
function material() {
  const uniforms = { tint: { value: new Color() }, phase: { value: 0 }, opacity: { value: 0 }, fireball: { value: 0 }, comet: { value: 0 } }
  return Object.assign(new ShaderMaterial({ transparent: true, depthWrite: false, blending: AdditiveBlending, toneMapped: false,
    uniforms,
    vertexShader: 'varying vec2 vUv; void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}',
    fragmentShader: `varying vec2 vUv; uniform vec3 tint; uniform float phase, opacity, fireball, comet;
      void main(){
        float x=vUv.x; float y=vUv.y-.5;
        float head=exp(-1000.*((x-.86)*(x-.86)+y*y));
        float width=.012+.065*(1.-x)+comet*.06*(1.-x);
        float tail=exp(-y*y/(width*width))*smoothstep(.02,.7,x)*(1.-smoothstep(.82,.9,x));
        float wake=exp(-y*y/.018)*smoothstep(0.,.6,x)*(1.-smoothstep(.6,.85,x))*.18;
        float fragments=0.;
        for(int i=0;i<4;i++){float f=float(i);vec2 p=vec2(.18+f*.13,.06*sin(f*3.+phase*7.));
          fragments+=exp(-1800.*dot(vec2(x,y)-p,vec2(x,y)-p));}
        float alpha=(head+tail*.65+wake+fireball*fragments*.6)*opacity;
        gl_FragColor=vec4(mix(tint,vec3(1.),head*.6),min(alpha,1.));
      }`,
  }), { uniforms })
}

export function SkyEvents({ seed, eligibility, leases, preview, onStatus, clock }: {
  clock: WorldClock;
  seed: number; eligibility: SkyEligibility; leases: AnimationLeases; preview?: SkyEvent;
  onStatus: (status: string) => void;
}) {
  const invalidate = useThree(state => state.invalidate)
  useEffect(() => () => ambientMeasurements.clear('sky'), [])
  const resources = useMemo(() => {
    const geometry = new PlaneGeometry(1, 1)
    const meshes = Array.from({ length: 3 }, () => {
      const mesh = new Mesh(geometry, material()); mesh.visible = false; mesh.frustumCulled = false
      mesh.raycast = () => undefined
      return mesh
    })
    return { geometry, meshes, right: new Vector3(), direction: new Vector3() }
  }, [])
  const controllerRef = useRef<SkyPresentation | null>(null)
  const lastStatus = useRef('')
  useEffect(() => {
    const controller = new SkyPresentation(leases)
    controllerRef.current = controller
    return () => { controller.dispose(); controllerRef.current = null }
  }, [leases])
  useEffect(() => () => { resources.geometry.dispose(); resources.meshes.forEach(mesh => mesh.material.dispose()) }, [resources])
  useEffect(() => { invalidate() }, [eligibility, invalidate, preview])
  useEffect(() => bindAmbientWake(now => nextSkyBoundary(seed, now, eligibility, preview), invalidate, clock),
    [clock, seed, eligibility.ambientEnabled, eligibility.night, eligibility.clearSky, eligibility.reducedMotion, preview, invalidate]) // eslint-disable-line react-hooks/exhaustive-deps -- depend on eligibility values, not the render-created object
  useFrame(({ camera }) => {
    const measurementStart = performance.now()
    const controller = controllerRef.current
    if (!controller) return
    const snapshot = clock.snapshot()
    const now = snapshot.utcMilliseconds / 1000
    const currentPreview = preview && now < preview.end ? preview : undefined
    controller.update(seed, now, eligibility, currentPreview, snapshot.timeScale > 0)
    const slots = [...controller.meteors.slots, ...controller.comets.slots]
    resources.right.set(1, 0, 0).applyQuaternion(camera.quaternion)
    for (const [index, slot] of slots.entries()) {
      const mesh = resources.meshes[index]
      if (!mesh) continue
      const event = slot.event
      mesh.visible = event !== null
      if (!event) continue
      mesh.material.depthTest = !currentPreview
      const rng = new Rng(event.seed)
      const phase = (now - event.start) / (event.end - event.start)
      const comet = event.family === 'comet'
      const fireball = !comet && event.variant === 'great-fireball'
      if (currentPreview) resources.direction.set(0, .17, -1).normalize().applyQuaternion(camera.quaternion)
      else resources.direction.set(rng.range(-1, 1), rng.range(.45, 1), rng.range(-1, 1)).normalize()
      mesh.position.copy(camera.position).addScaledVector(resources.direction, 100)
      if (!comet) mesh.position.addScaledVector(resources.right, (phase - .5) * 36)
      mesh.quaternion.copy(camera.quaternion)
      mesh.rotateZ(comet ? .35 : -.3)
      mesh.scale.set(comet ? 28 : fireball ? 30 : 18, comet ? 9 : fireball ? 6 : 3, 1)
      mesh.material.uniforms.tint.value.set(comet ? '#b4dcff' : colors[event.variant])
      mesh.material.uniforms.phase.value = phase
      mesh.material.uniforms.fireball.value = Number(fireball)
      mesh.material.uniforms.comet.value = Number(comet)
      mesh.material.uniforms.opacity.value = comet ? .8 : Math.min(1, phase * 12, (1 - phase) * 5)
    }
    const status = `${controller.meteors.stats().active} meteors · ${controller.comets.stats().active} comets · ${leases.count} animation leases`
    if (status !== lastStatus.current) { lastStatus.current = status; onStatus(status) }
    ambientMeasurements.record('sky', performance.now() - measurementStart, controller.meteors.stats().active + controller.comets.stats().active, 3)
  })
  return <group name="ambient-sky">{resources.meshes.map((mesh, index) => <primitive key={index} object={mesh} dispose={null} />)}</group>
}
