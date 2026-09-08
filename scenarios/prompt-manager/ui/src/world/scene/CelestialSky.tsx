import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useLayoutEffect, useMemo } from 'react'
import { Color, Mesh, PlaneGeometry, Quaternion, ShaderMaterial, Vector3 } from 'three'
import type { WorldClock } from '../config/clock'
import { celestialStyle, deepNightAmount, lunarPhase, starVisibility, stylizedSunDirection } from '../config/celestial'
import type { LightingPeriod, PeriodId, QualityProfileId } from '../config'
import { createMilkyWay, createStarField } from './starField'

/** Camera-centred background body. Clip-space depth keeps it behind world
 * geometry without depending on the camera's finite far plane or zoom distance.
 */
export function CelestialSky({ clock, mode, period, cloudCoverage, seed, profileId }: {
  seed: number; profileId: QualityProfileId;
  clock: WorldClock; mode: 'clock' | PeriodId; period: LightingPeriod; cloudCoverage: number;
}) {
  const invalidate = useThree(state => state.invalidate)
  const stars = useMemo(() => createStarField(seed), [seed])
  const galaxy = useMemo(() => createMilkyWay(seed), [seed])
  useEffect(() => () => galaxy.dispose(), [galaxy])
  useEffect(() => () => stars.dispose(), [stars])
  useLayoutEffect(() => { stars.geometry.setDrawRange(0, celestialStyle.starCounts[profileId]); invalidate() }, [stars, profileId, invalidate])
  const resources = useMemo(() => {
    const geometry = new PlaneGeometry(1, 1)
    const uniforms = { tint: { value: new Color() }, opacity: { value: 1 },
      coreRadius: { value: celestialStyle.sunRadiusDegrees / celestialStyle.sunHaloRadiusDegrees }, haloOpacity: { value: celestialStyle.sunHaloOpacity } }
    const material = new ShaderMaterial({ uniforms, transparent: true, depthWrite: false, toneMapped: false,
      vertexShader: `varying vec2 vUv; void main(){vUv=uv;vec4 p=projectionMatrix*modelViewMatrix*vec4(position,1.);p.z=p.w*.999999;gl_Position=p;}`,
      fragmentShader: `varying vec2 vUv;uniform vec3 tint;uniform float opacity,coreRadius,haloOpacity;
        void main(){float r=length(vUv*2.-1.);float edge=max(fwidth(r),.002);
          float core=1.-smoothstep(coreRadius-edge,coreRadius+edge,r);
          float halo=exp(-5.*r*r)*(1.-smoothstep(.75,1.,r))*haloOpacity;
          gl_FragColor=vec4(tint,min(1.,core+halo)*opacity);}`,
    })
    const sun = new Mesh(geometry, material)
    sun.name = 'celestial-sun'; sun.frustumCulled = false; sun.renderOrder = -1000; sun.raycast = () => undefined
    const moonUniforms = { lightDirection: { value: new Vector3() }, opacity: { value: 1 } }
    const moonMaterial = new ShaderMaterial({ uniforms: moonUniforms, transparent: true, depthWrite: false, toneMapped: false,
      vertexShader: material.vertexShader,
      fragmentShader: `varying vec2 vUv; uniform vec3 lightDirection; uniform float opacity;
        void main(){vec2 p=vUv*2.-1.;float r2=dot(p,p);if(r2>1.)discard;
          vec3 normal=vec3(p,sqrt(max(0.,1.-r2)));
          float light=smoothstep(-.012,.012,dot(normal,lightDirection));
          float relief=.8+.06*sin(p.x*18.+sin(p.y*13.))+.04*cos(p.y*31.+p.x*9.);
          for(int i=0;i<12;i++){float f=float(i);vec2 centre=vec2(sin(f*7.13),cos(f*3.71))*.78;
            float radius=.025+.025*(1.+sin(f*4.));float d=length(p-centre)/radius;
            relief-=.12*exp(-d*d*1.8);relief+=.06*exp(-pow((d-1.)*5.,2.));}
          vec3 color=vec3(.88,.89,.92)*relief*(.025+.975*light);
          float edge=1.-smoothstep(1.-max(fwidth(r2),.002),1.,r2);
          gl_FragColor=vec4(color,edge*opacity);}`,
    })
    const moon = new Mesh(geometry, moonMaterial)
    moon.name = 'celestial-moon'; moon.frustumCulled = false; moon.renderOrder = -999; moon.raycast = () => undefined
    return { geometry, material, uniforms, sun, moon, moonMaterial, moonUniforms,
      sunDirection: new Vector3(), normal: new Vector3(), tangent: new Vector3(), moonDirection: new Vector3(), inverse: new Quaternion(),
      phaseKey: NaN, phaseFrozen: false, phase: lunarPhase(0), horizon: new Color('#ffc777'), high: new Color('#fff5df') }
  }, [])
  useEffect(() => clock.subscribe(invalidate), [clock, invalidate])
  useEffect(() => () => { resources.geometry.dispose(); resources.material.dispose(); resources.moonMaterial.dispose() }, [resources])
  useFrame(({ camera, gl }) => {
    const snapshot = clock.snapshot()
    let direction = stylizedSunDirection(snapshot.localMinutes)
    if (mode !== 'clock') {
      const angle = period.sunElevationDeg * Math.PI / 180
      const side = mode === 'dusk' ? -1 : 1
      direction = [side * Math.cos(angle), Math.sin(angle), 0]
    }
    const height = direction[1]
    const fade = Math.max(0, Math.min(1, (height + .012) / .035))
    resources.sun.visible = fade > 0
    resources.sun.position.copy(camera.position).add({ x: direction[0] * 100, y: direction[1] * 100, z: direction[2] * 100 })
    resources.sun.quaternion.copy(camera.quaternion)
    const size = 200 * Math.tan(celestialStyle.sunHaloRadiusDegrees * Math.PI / 180)
    resources.sun.scale.set(size, size, 1)
    resources.uniforms.tint.value.copy(resources.horizon).lerp(resources.high, Math.max(0, Math.min(1, height * 3)))
    resources.uniforms.opacity.value = fade * (1 - Math.max(0, Math.min(1, cloudCoverage)) * .95)
    const phaseKey = snapshot.timeScale === 0 ? snapshot.utcMilliseconds : Math.floor(snapshot.utcMilliseconds / 60000)
    if (phaseKey !== resources.phaseKey || resources.phaseFrozen !== (snapshot.timeScale === 0)) {
      resources.phaseKey = phaseKey; resources.phaseFrozen = snapshot.timeScale === 0; resources.phase = lunarPhase(snapshot.utcMilliseconds)
    }
    const tilt = celestialStyle.sunNoonElevationDegrees * Math.PI / 180
    resources.sunDirection.set(...direction)
    if (mode === 'clock') resources.normal.set(0, Math.cos(tilt), Math.sin(tilt))
    else resources.normal.set(0, 0, 1)
    resources.tangent.crossVectors(resources.normal, resources.sunDirection).normalize()
    const angle = resources.phase.cycle * Math.PI * 2
    const latitude = resources.phase.latitudeDegrees * Math.PI / 180
    resources.moonDirection.copy(resources.sunDirection).multiplyScalar(Math.cos(angle) * Math.cos(latitude))
      .addScaledVector(resources.tangent, -Math.sin(angle) * Math.cos(latitude)).addScaledVector(resources.normal, Math.sin(latitude)).normalize()
    resources.moon.position.copy(camera.position).addScaledVector(resources.moonDirection, 100)
    resources.moon.lookAt(camera.position)
    const moonSize = 200 * Math.tan(celestialStyle.moonRadiusDegrees * Math.PI / 180)
    resources.moon.scale.set(moonSize, moonSize, 1)
    resources.moon.visible = resources.moonDirection.y > -.012
    resources.inverse.copy(resources.moon.quaternion).invert()
    resources.moonUniforms.lightDirection.value.copy(resources.sunDirection).applyQuaternion(resources.inverse)
    resources.moonUniforms.opacity.value = Math.max(0, Math.min(1, (resources.moonDirection.y + .012) / .035)) * (1 - Math.max(0, Math.min(1, cloudCoverage)) * .95)
    stars.points.position.copy(camera.position)
    const quiet = mode === 'clock' ? deepNightAmount(snapshot.localMinutes) : 0
    stars.geometry.setDrawRange(0, Math.round(celestialStyle.starCounts[profileId] * (.65 + .35 * quiet)))
    const visibility = starVisibility(height, cloudCoverage, resources.phase.illumination, resources.moonDirection.y)
    stars.uniforms.visibility.value = visibility * (.65 + .35 * quiet)
    stars.uniforms.dpr.value = gl.getPixelRatio()
    stars.points.visible = stars.uniforms.visibility.value > .001
    galaxy.mesh.position.copy(camera.position)
    galaxy.uniforms.visibility.value = visibility * quiet
    galaxy.mesh.visible = galaxy.uniforms.visibility.value > .001
  })
  return <><primitive object={galaxy.mesh} dispose={null} /><primitive object={stars.points} dispose={null} /><primitive object={resources.sun} dispose={null} /><primitive object={resources.moon} dispose={null} /></>
}
