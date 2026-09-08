import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { AdditiveBlending, DoubleSide, Frustum, InstancedMesh, Matrix4, Object3D, PlaneGeometry, ShaderMaterial, Sphere, Vector3 } from 'three'
import type { WorldClock } from '../config/clock'
import type { AnimationLeases } from '../engine/animationLeases'
import type { Placement } from './Props'

/** Two crossed flame cards and one ember card per fire, in one instanced draw.
 * Logs and point lights stay in the shared prop/light pipeline.
 */
export function Campfires({ placements, flame, embers, clock, leases, reducedMotion }: {
  placements: readonly Placement[]; flame: number; embers: number; clock: WorldClock; leases: AnimationLeases; reducedMotion: boolean;
}) {
  const invalidate = useThree(state => state.invalidate)
  const release = useRef<(() => void) | null>(null)
  const resources = useMemo(() => {
    const geometry = new PlaneGeometry(1, 1)
    const uniforms = { time: { value: 0 }, flame: { value: 0 }, embers: { value: 0 } }
    const material = new ShaderMaterial({ uniforms, side: DoubleSide, transparent: true, depthWrite: false, toneMapped: false, blending: AdditiveBlending,
      vertexShader: `varying vec2 vUv;varying float ember,phase;
        void main(){vUv=uv;ember=step(.5,abs(instanceMatrix[1].z));phase=instanceMatrix[3].x*7.13+instanceMatrix[3].z*3.71;
          gl_Position=projectionMatrix*modelViewMatrix*instanceMatrix*vec4(position,1.);}`,
      fragmentShader: `varying vec2 vUv;varying float ember,phase;uniform float time,flame,embers;
        void main(){vec2 p=vec2(vUv.x*2.-1.,vUv.y);float t=time+phase;
          if(ember>.5){float r=length((vUv-.5)*2.);float coals=.6+.4*sin(vUv.x*37.)*sin(vUv.y*29.);
            gl_FragColor=vec4(1.,.15,.025,(1.-smoothstep(.45,1.,r))*coals*embers*.75);return;}
          float sway=sin(p.y*8.-t*3.)*.1*p.y+sin(p.y*17.-t*5.)*.035;
          float width=(1.-p.y)*(.48+.09*sin(t*4.+p.y*15.));
          float body=(1.-smoothstep(width*.4,width,abs(p.x-sway)))*smoothstep(0.,.1,p.y)*(1.-smoothstep(.78,1.,p.y));
          float core=1.-smoothstep(width*.1,width*.48,abs(p.x-sway));
          vec3 color=mix(vec3(1.,.15,.015),vec3(1.,.78,.2),core*(1.-p.y));
          gl_FragColor=vec4(color,body*flame*.8);}`,
    })
    const mesh = new InstancedMesh(geometry, material, Math.max(1, placements.length * 3))
    mesh.name = 'campfire-effects'; mesh.frustumCulled = false; mesh.raycast = () => undefined
    const local = new Object3D()
    for (const [index, placement] of placements.entries()) {
      for (let card = 0; card < 3; card++) {
        const ember = card === 2
        local.position.set(placement.position[0], (placement.y ?? 0) + (ember ? .15 : .63), placement.position[1])
        local.rotation.set(ember ? -Math.PI / 2 : 0, card === 1 ? Math.PI / 2 : 0, 0)
        local.scale.set(ember ? .9 : 1.15, ember ? .9 : 1.2, 1)
        local.updateMatrix(); mesh.setMatrixAt(index * 3 + card, local.matrix)
      }
    }
    mesh.count = placements.length * 3
    return { mesh, geometry, material, uniforms, frustum: new Frustum(), matrix: new Matrix4(), sphere: new Sphere(new Vector3(), 1.5) }
  }, [placements])
  useEffect(() => () => { resources.mesh.dispose(); resources.geometry.dispose(); resources.material.dispose() }, [resources])
  useEffect(() => () => { release.current?.(); release.current = null }, [leases])
  useEffect(() => clock.subscribe(invalidate), [clock, invalidate])
  useEffect(() => { invalidate() }, [flame, embers, reducedMotion, invalidate])
  useFrame(({ camera }) => {
    const snapshot = clock.snapshot()
    resources.uniforms.time.value = reducedMotion ? 0 : snapshot.utcMilliseconds / 1000 % (Math.PI * 2)
    resources.uniforms.flame.value = flame; resources.uniforms.embers.value = embers
    resources.mesh.visible = embers > 0 && placements.length > 0
    let moving = false
    if (flame > 0 && !reducedMotion && snapshot.timeScale > 0) {
      resources.frustum.setFromProjectionMatrix(resources.matrix.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse))
      moving = placements.some(p => {
        resources.sphere.center.set(p.position[0], (p.y ?? 0) + .6, p.position[1])
        return resources.sphere.center.distanceToSquared(camera.position) < 6400 && resources.frustum.intersectsSphere(resources.sphere)
      })
    }
    if (moving) release.current ??= leases.acquire()
    else { release.current?.(); release.current = null }
  })
  return <primitive object={resources.mesh} dispose={null} />
}
