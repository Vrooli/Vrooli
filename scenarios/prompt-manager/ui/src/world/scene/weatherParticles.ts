import { BufferAttribute, BufferGeometry, Color, ShaderMaterial } from 'three'
import type { WeatherPreset, WeatherTuning } from '../config'

export function createParticleGeometry(count: number, particles: WeatherTuning['particles']): BufferGeometry {
  const positions = new Float32Array(Math.max(1, count) * 3)
  for (let index = 0; index < count; index += 1) {
    const angle = index * particles.spiralAngleStep
    const radius = Math.sqrt(index / Math.max(1, count)) * particles.columnRadius
    positions[index * 3] = Math.sin(angle) * radius
    positions[index * 3 + 1] = (index * particles.verticalStride) % particles.columnHeight
    positions[index * 3 + 2] = Math.cos(angle) * radius
  }
  const result = new BufferGeometry()
  result.setAttribute('position', new BufferAttribute(positions, 3))
  return result
}

export function createParticleMaterial(preset: WeatherPreset, particles: WeatherTuning['particles']): ShaderMaterial {
  return new ShaderMaterial({
    transparent: true,
    depthWrite: false,
    uniforms: {
      uTime: { value: 0 },
      uSpeed: { value: preset.particleFallSpeed },
      uSize: { value: preset.particleSize },
      uColor: { value: new Color(preset.particleColor) },
      uColumnHeight: { value: particles.columnHeight },
      uPointSizeScale: { value: particles.pointSizeScale },
      uOpacity: { value: particles.opacity },
    },
    vertexShader: `
      uniform float uTime;
      uniform float uSpeed;
      uniform float uSize;
      uniform float uColumnHeight;
      uniform float uPointSizeScale;
      void main() {
        vec3 p = position;
        p.y = mod(position.y - uTime * uSpeed + uColumnHeight, uColumnHeight);
        vec4 mvPosition = modelViewMatrix * vec4(p, 1.0);
        gl_PointSize = uSize * uPointSizeScale / max(1.0, -mvPosition.z);
        gl_Position = projectionMatrix * mvPosition;
      }
    `,
    fragmentShader: `
      uniform vec3 uColor;
      uniform float uOpacity;
      void main() {
        vec2 point = gl_PointCoord - vec2(0.5);
        const float pointRadius = 0.5;
        if (dot(point, point) > pointRadius * pointRadius) discard;
        gl_FragColor = vec4(uColor, uOpacity);
      }
    `,
  })
}
