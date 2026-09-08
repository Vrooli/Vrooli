import { BufferAttribute, BufferGeometry, Points, ShaderMaterial } from 'three'
import { celestialStyle } from '../config/celestial'
import { hashString, Rng } from '../sim/rng'

/** Fictional seeded stars; profile budgets draw prefixes of one stable field. */
export function createStarField(seed: number) {
  const count = celestialStyle.starCounts.ultra
  const position = new Float32Array(count * 3), brightness = new Float32Array(count)
  const rng = new Rng(hashString(`celestial-stars-v1:${seed}`))
  for (let index = 0; index < count; index++) {
    const y = rng.range(-1, 1), angle = rng.range(0, Math.PI * 2), radius = Math.sqrt(1 - y * y)
    position.set([Math.cos(angle) * radius * 100, y * 100, Math.sin(angle) * radius * 100], index * 3)
    brightness[index] = rng.range(.35, 1)
  }
  const geometry = new BufferGeometry()
  geometry.setAttribute('position', new BufferAttribute(position, 3))
  geometry.setAttribute('brightness', new BufferAttribute(brightness, 1))
  const uniforms = { visibility: { value: 0 }, dpr: { value: 1 } }
  const material = new ShaderMaterial({ uniforms, transparent: true, depthWrite: false, toneMapped: false,
    vertexShader: `attribute float brightness; varying float vBrightness; varying float horizon; uniform float dpr;
      void main(){vBrightness=brightness;horizon=smoothstep(0.,15.,position.y);
        vec4 p=projectionMatrix*modelViewMatrix*vec4(position,1.);p.z=p.w*.999999;gl_Position=p;
        gl_PointSize=(1.+2.*brightness)*dpr;}`,
    fragmentShader: `varying float vBrightness;varying float horizon;uniform float visibility;
      void main(){float r=length(gl_PointCoord*2.-1.);float alpha=(1.-smoothstep(.1,1.,r))*vBrightness*visibility*horizon;
        gl_FragColor=vec4(mix(vec3(.72,.82,1.),vec3(1.,.94,.8),vBrightness),alpha);}`,
  })
  const points = new Points(geometry, material)
  points.name = 'celestial-stars'; points.frustumCulled = false; points.renderOrder = -1001; points.raycast = () => undefined
  return { points, geometry, material, uniforms, dispose: () => { geometry.dispose(); material.dispose() } }
}
