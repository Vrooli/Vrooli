import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import { BackSide, Color, Mesh, ShaderMaterial, SphereGeometry } from 'three'
import type { LightingPeriod, QualityProfile, WeatherPreset } from '../config'
import type { WorldClock } from '../config/clock'

/** Camera-centred atmosphere. The HDRI supplies reflections only; this sky and
 * the celestial layers share the resolved weather/time palette at every hour.
 */
export function SceneEnvironment({ period, weather, profile, clock }: {
  period: LightingPeriod; weather: WeatherPreset; profile: QualityProfile; clock: WorldClock;
}) {
  const invalidate = useThree(state => state.invalidate)
  const resources = useMemo(() => {
    const geometry = new SphereGeometry(100, 48, 24)
    const uniforms = { zenith: { value: new Color() }, horizon: { value: new Color() }, cloudTint: { value: new Color() },
      coverage: { value: 0 }, time: { value: 0 }, clouds: { value: 1 } }
    const material = new ShaderMaterial({ uniforms, side: BackSide, depthWrite: false, toneMapped: false,
      vertexShader: `varying vec3 direction;
        void main(){direction=normalize(position);vec4 p=projectionMatrix*modelViewMatrix*vec4(position,1.);p.z=p.w*.999999;gl_Position=p;}`,
      fragmentShader: `varying vec3 direction;uniform vec3 zenith,horizon,cloudTint;uniform float coverage,time,clouds;
        float hash(vec2 p){return fract(sin(dot(p,vec2(127.1,311.7)))*43758.5453);}
        float noise(vec2 p){vec2 i=floor(p),f=fract(p);f=f*f*(3.-2.*f);
          return mix(mix(hash(i),hash(i+vec2(1,0)),f.x),mix(hash(i+vec2(0,1)),hash(i+vec2(1,1)),f.x),f.y);}
        float fbm(vec2 p){return .55*noise(p)+.28*noise(p*2.03)+.12*noise(p*4.07)+.05*noise(p*8.13);}
        void main(){vec3 d=normalize(direction);float height=max(0.,d.y);
          vec3 sky=mix(horizon,zenith,smoothstep(0.,.7,height));
          vec2 p=d.xz/max(.12,d.y)*1.4+vec2(time,time*.23);
          float n=fbm(p);float threshold=mix(.54,.22,coverage);
          float body=smoothstep(threshold,threshold+.13,n)*smoothstep(.025,.18,d.y)*clouds;
          vec3 cloud=cloudTint*(.7+.3*n);
          gl_FragColor=vec4(mix(sky,cloud,body*.9),1.);}`,
    })
    const mesh = new Mesh(geometry, material)
    mesh.name = 'atmosphere-sky'; mesh.frustumCulled = false; mesh.renderOrder = -1003; mesh.raycast = () => undefined
    return { mesh, material, geometry, uniforms }
  }, [])
  useEffect(() => () => { resources.geometry.dispose(); resources.material.dispose() }, [resources])
  useEffect(() => clock.subscribe(invalidate), [clock, invalidate])
  useEffect(() => {
    const daylight = Math.max(0, Math.min(1, (period.sunElevationDeg + 8) / 25))
    resources.uniforms.zenith.value.set(period.backgroundColor).lerp(new Color('#549cdb'), daylight * .8).multiplyScalar(.65 + daylight * .35)
    resources.uniforms.horizon.value.set(period.fogColor)
    resources.uniforms.cloudTint.value.set(period.fogColor).lerp(new Color('#ffffff'), daylight * .75)
    resources.uniforms.coverage.value = weather.cloudCoverage
    resources.uniforms.clouds.value = profile.clouds ? 1 : 0
    invalidate()
  }, [resources, period, weather.cloudCoverage, profile.clouds, invalidate])
  useFrame(({ camera }) => {
    resources.mesh.position.copy(camera.position)
    resources.uniforms.time.value = clock.snapshot().utcMilliseconds / 100000000
  })
  return <primitive object={resources.mesh} dispose={null} />
}
