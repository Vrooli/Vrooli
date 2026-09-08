import { BackSide, BufferAttribute, BufferGeometry, Mesh, Points, ShaderMaterial, SphereGeometry } from 'three'
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

/** One static, texture-free sky draw: a mottled galactic band with a dark dust
 * lane. Visibility follows the same sun/cloud/moon occlusion as the stars.
 */
export function createMilkyWay(seed: number) {
  const geometry = new SphereGeometry(100, 48, 24)
  const uniforms = { visibility: { value: 0 }, seed: { value: (hashString(`galaxy:${seed}`) % 1000) / 100 } }
  const material = new ShaderMaterial({ uniforms, side: BackSide, transparent: true, depthWrite: false, toneMapped: false,
    vertexShader: `varying vec3 direction;
      void main(){direction=normalize(position);vec4 p=projectionMatrix*modelViewMatrix*vec4(position,1.);p.z=p.w*.999999;gl_Position=p;}`,
    fragmentShader: `varying vec3 direction;uniform float visibility,seed;
      float hash(vec3 p){return fract(sin(dot(p,vec3(127.1,311.7,74.7)))*43758.5453);}
      float noise(vec3 p){vec3 i=floor(p),f=fract(p);f=f*f*(3.-2.*f);
        return mix(mix(mix(hash(i),hash(i+vec3(1,0,0)),f.x),mix(hash(i+vec3(0,1,0)),hash(i+vec3(1,1,0)),f.x),f.y),
          mix(mix(hash(i+vec3(0,0,1)),hash(i+vec3(1,0,1)),f.x),mix(hash(i+vec3(0,1,1)),hash(i+vec3(1,1,1)),f.x),f.y),f.z);}
      float fbm(vec3 p){return .57*noise(p)+.28*noise(p*2.03)+.1*noise(p*4.07)+.05*noise(p*8.13);}
      void main(){vec3 d=normalize(direction);vec3 n=normalize(vec3(.25,.55,.8));float latitude=dot(d,n);
        float clouds=fbm(d*18.+seed);float fine=fbm(d*65.+seed);
        float core=pow(max(0.,dot(d,normalize(vec3(.5,.65,-.6)))),6.);
        float width=.055+core*.05;float band=exp(-pow(latitude/width,2.));
        float bend=.018*sin(dot(d,vec3(4.,7.,-5.)))+.025*(clouds-.5);
        float lane=exp(-pow((latitude+bend)/(.01+.014*fine),2.))*(.35+.65*fine);
        float glow=band*(.18+.7*clouds+.12*fine)*(1.-.85*lane);
        vec3 color=mix(vec3(.29,.37,.63),vec3(.76,.68,.53),core*.6+clouds*.2);
        float horizon=smoothstep(0.,.22,d.y);
        gl_FragColor=vec4(color,glow*visibility*horizon*.45);}`,
  })
  const mesh = new Mesh(geometry, material)
  mesh.name = 'celestial-milky-way'; mesh.frustumCulled = false; mesh.renderOrder = -1002; mesh.raycast = () => undefined
  return { mesh, geometry, material, uniforms, dispose: () => { geometry.dispose(); material.dispose() } }
}
